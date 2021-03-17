// Creating a user in a random organization:
//      Given a user named "Bob"
// Creating a user in a given organization:
//      Given a user named "Jimmy" in organization "13639843"
// Logging into a user session:
//      Given I am logged in as "Jimmy"
// Setting the Authorization header of the current user session:
//      Given I set the Authorization header to "Bearer ${agent_token}"
package cucumber

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/cucumber/godog"
)

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^a user named "([^"]*)"$`, s.Suite.createUserNamed)
		ctx.Step(`^a user named "([^"]*)" in organization "([^"]*)"$`, s.Suite.createUserNamedInOrganization)
		ctx.Step(`^I am logged in as "([^"]*)"$`, s.iAmLoggedInAs)
		ctx.Step(`^I set the Authorization header to "([^"]*)"$`, s.iSetTheAuthorizationHeaderTo)
	})
}

func (s *TestSuite) createUserNamed(name string) error {
	// this value if taken from config/allow-list-configuration.yaml
	s.Mu.Lock()
	orgId := s.nextOrgId
	s.nextOrgId += 1
	s.Mu.Unlock()
	return s.createUserNamedInOrganization(name, fmt.Sprintf("%d", orgId))
}

func (s *TestSuite) createUserNamedInOrganization(name string, orgid string) error {
	// users are shared concurrently across scenarios.. so lock while we create the user...
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if s.users[name] != nil {
		return nil
	}

	// setup pre-requisites to performing requests
	account := s.Helper.NewAccountWithNameAndOrg(name, orgid)
	token, err := s.Helper.AuthHelper.CreateSignedJWT(account, nil)
	if err != nil {
		return err
	}

	s.users[name] = &TestUser{
		Name:  name,
		Token: token,
		Ctx:   context.WithValue(context.Background(), openapi.ContextAccessToken, token),
	}
	return nil
}
func (s *TestScenario) iAmLoggedInAs(name string) error {
	s.Session().AuthorizationHeader = ""
	s.CurrentUser = name
	return nil
}

func (s *TestScenario) iSetTheAuthorizationHeaderTo(value string) error {
	value = s.Expand(value)
	s.Session().AuthorizationHeader = value
	return nil
}
