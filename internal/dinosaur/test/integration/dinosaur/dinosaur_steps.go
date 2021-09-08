// Ensures the user has at least one dinosaur cluster created and stores it's id in a scenario variable:
//    Given I have created a dinosaur cluster as ${kid}
package dinosaur

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/cucumber"
	"github.com/cucumber/godog"
)

func init() {
	// This is how we can contribute additional steps over the standard ones provided in the cucumber package.
	cucumber.StepModules = append(cucumber.StepModules, func(ctx *godog.ScenarioContext, s *cucumber.TestScenario) {
		e := &extender{s}
		ctx.Step(`^I have created a dinosaur cluster as \${([^"]*)}$`, e.iHaveCreatedADinosaurClusterAs)
	})
}

type extender struct {
	*cucumber.TestScenario
}

func (s *extender) iHaveCreatedADinosaurClusterAs(id string) error {
	session := s.Session()

	// lock the TestUser to avoid 2 scenarios with the same TestUser, creating the dinosaur concurrently.
	session.TestUser.Mu.Lock()
	defer session.TestUser.Mu.Unlock()

	apiClient := test.NewApiClient(s.Suite.Helper)
	dinosaurs, _, err := apiClient.DefaultApi.GetDinosaurs(session.TestUser.Ctx, &public.GetDinosaursOpts{})
	if err != nil {
		return err
	}

	dinosaurId := ""
	if len(dinosaurs.Items) != 0 {
		dinosaurId = dinosaurs.Items[0].Id
	} else {
		dinosaur, _, err := apiClient.DefaultApi.CreateDinosaur(session.TestUser.Ctx, true, public.DinosaurRequestPayload{
			Name:          "mydinosaur",
			CloudProvider: "aws",
			Region:        "us-east-1",
			MultiAz:       true,
		})
		if err != nil {
			return err
		}
		dinosaurId = dinosaur.Id
	}
	s.Variables[id] = dinosaurId

	return nil
}
