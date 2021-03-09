package connector

import (
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/cucumber/godog"
	"net/url"
	"testing"
	"time"
)

type extender struct {
	*cucumber.TestScenario
}

func (s *extender) getAndStoreAccessTokenUsingTheAddonParameterResponseAs(as string) error {
	session := s.Session()

	params := []openapi.AddonParameter{}
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

	// because I was seeing "Bearer token was issued in the future" errors...
	time.Sleep(2 * time.Second)

	return nil
}

func init() {
	// This is how we can contribute additional steps over the standard ones provided in the cucumber package.
	cucumber.StepModules = append(cucumber.StepModules, func(ctx *godog.ScenarioContext, s *cucumber.TestScenario) {
		e := &extender{s}
		ctx.Step(`^get and store access token using the addon parameter response as \${([^"]*)}$`, e.getAndStoreAccessTokenUsingTheAddonParameterResponseAs)
	})
}

func TestMain(m *testing.M) {

	// Startup all the services and mocks that are needed to test the
	// connector features.
	t := &testing.T{}

	connectorTypeService := mocks.NewConnectorTypeMock(t)
	defer connectorTypeService.Close()
	environments.Environment().Config.ConnectorsConfig.Enabled = true
	environments.Environment().Config.ConnectorsConfig.ConnectorTypesDir = ""
	environments.Environment().Config.ConnectorsConfig.ConnectorTypeSvcUrls = []string{connectorTypeService.URL}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	cucumber.TestMain(m, h)

}
