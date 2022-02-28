package integration

import (
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"net/url"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/chirino/graphql"
	"github.com/chirino/graphql/schema"
	"github.com/cucumber/godog"
)

type extender struct {
	*cucumber.TestScenario
}

func (s *extender) iResetTheVaultCounters() error {
	var service vault.VaultService
	if err := s.Suite.Helper.Env.ServiceContainer.Resolve(&service); err != nil {
		return err
	}
	if vault, ok := service.(*vault.TmpVaultService); ok {
		vault.ResetCounters()
	}
	return nil
}

func (s *extender) theVaultDeleteCounterShouldBe(expected int64) error {
	// we can only check the delete count on the TmpVault service impl...
	var service vault.VaultService
	if err := s.Suite.Helper.Env.ServiceContainer.Resolve(&service); err != nil {
		return err
	}

	if vault, ok := service.(*vault.TmpVaultService); ok {
		actual := vault.Counters().Deletes
		if actual != expected {
			return fmt.Errorf("vault delete counter does not match expected: %v, actual: %v", expected, actual)
		}
	}
	return nil
}

func (s *extender) getAndStoreAccessTokenUsingTheAddonParameterResponseAs(as string, clientID string) error {
	session := s.Session()

	params := []public.AddonParameter{}
	err := json.Unmarshal(session.RespBytes, &params)
	if err != nil {
		return err
	}

	byId := map[string]string{}
	for _, p := range params {
		byId[p.Id] = p.Value
	}

	u, err := url.Parse(fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", byId["mas-sso-base-url"], byId["mas-sso-realm"]))
	if err != nil {
		return err
	}
	u.User = url.UserPassword(byId["client-id"], byId["client-secret"])
	tokenUrl := u.String()
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	resp, err := session.Client.PostForm(tokenUrl, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status code %d when getting the access token", resp.StatusCode)
	}
	props := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&props)
	if err != nil {
		return err
	}

	accessToken := props["access_token"]
	if accessToken == nil || accessToken == "" {
		return fmt.Errorf("access token not found in the response")
	}

	s.Variables[as] = fmt.Sprintf("%s", accessToken)
	s.Variables[clientID] = byId["client-id"]

	// because I was seeing "Bearer token was issued in the future" errors...
	time.Sleep(2 * time.Second)

	return nil
}

const clientIdList = "_client_id_list"

func (s *extender) deleteKeycloakClients(sc *godog.Scenario, err error) {

	if clientIds, ok := s.Variables[clientIdList].([]string); ok {
		env := s.Suite.Helper.Env
		var keycloakService services.KafkaKeycloakService
		env.MustResolve(&keycloakService)

		for _, clientID := range clientIds {
			if err := keycloakService.DeleteServiceAccountInternal(clientID); err != nil {
				glog.Errorf("Error deleting keycloak client with clientId %s: %s", clientID, err)
			}
		}
	}
}

func (s *extender) rememberKeycloakClientForCleanup(clientID string) error {
	clientIDs, ok := s.Variables[clientIdList].([]string)
	if !ok {
		s.Variables[clientIdList] = []string{s.Variables[clientID].(string)}
	} else {
		s.Variables[clientIdList] = append(clientIDs, s.Variables[clientID].(string))
	}
	return nil
}

func (s *extender) updateConnectorCatalogOfTypeAndChannelWithShardMetadata(connectorTypeId, channel string, metadata *godog.DocString) error {
	content, err := s.Expand(metadata.Content, []string{"defs", "ref"})
	if err != nil {
		return err
	}

	shardMetadata := map[string]interface{}{}
	err = json.Unmarshal([]byte(content), &shardMetadata)
	if err != nil {
		return err
	}

	var connectorManager *workers.ConnectorManager
	if err := s.Suite.Helper.Env.ServiceContainer.Resolve(&connectorManager); err != nil {
		return err
	}

	ccc := &config.ConnectorChannelConfig{
		ShardMetadata: shardMetadata,
	}
	serr := connectorManager.ReconcileConnectorCatalogEntry(connectorTypeId, channel, ccc)
	if serr != nil {
		return serr
	}
	return nil
}

func (s *extender) iPOSTToAGraphQLQuery(path string, query *godog.DocString) error {

	doc := &schema.QueryDocument{}
	err := doc.Parse(query.Content)
	if err != nil {
		return err
	}
	op, err := doc.GetOperation("")
	if err != nil {
		return err
	}

	vars := map[string]interface{}{}
	request := graphql.Request{
		Query:     query.Content,
		Variables: vars,
	}
	for _, v := range op.Vars {
		name := strings.TrimPrefix(v.Name, "$")
		if value, ok := s.Variables[name]; ok {
			vars[name] = value
		} else {
			return fmt.Errorf("graphql operation input var $%s not found in the session variable", name)
		}
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return s.SendHttpRequestWithJsonBodyAndStyle("POST", path, &godog.DocString{Content: string(data)}, false, false)
}

func init() {
	// This is how we can contribute additional steps over the standard ones provided in the cucumber package.
	cucumber.StepModules = append(cucumber.StepModules, func(ctx *godog.ScenarioContext, s *cucumber.TestScenario) {
		e := &extender{s}
		ctx.Step(`^get and store access token using the addon parameter response as \${([^"]*)} and clientID as \${([^"]*)}$`, e.getAndStoreAccessTokenUsingTheAddonParameterResponseAs)
		ctx.Step(`^the vault delete counter should be (\d+)$`, e.theVaultDeleteCounterShouldBe)
		ctx.Step(`^I reset the vault counters$`, e.iResetTheVaultCounters)
		ctx.Step(`^update connector catalog of type "([^"]*)" and channel "([^"]*)" with shard metadata:$`, e.updateConnectorCatalogOfTypeAndChannelWithShardMetadata)
		ctx.Step(`^I POST to "([^"]*)" a GraphQL query:$`, e.iPOSTToAGraphQLQuery)
		ctx.Step(`I remember keycloak client for cleanup with clientID: \${([^"]*)}$`, e.rememberKeycloakClientForCleanup)

		ctx.AfterScenario(e.deleteKeycloakClients)
	})
}
