// Ensures the user has at least one kafka cluster created and stores it's id in a scenario variable:
//    Given I have created a kafka cluster as ${kid}
package kafka

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/cucumber/godog"
)

func init() {
	// This is how we can contribute additional steps over the standard ones provided in the cucumber package.
	cucumber.StepModules = append(cucumber.StepModules, func(ctx *godog.ScenarioContext, s *cucumber.TestScenario) {
		e := &extender{s}
		ctx.Step(`^I have created a kafka cluster as \${([^"]*)}$`, e.iHaveCreatedAKafkaClusterAs)
	})
}

type extender struct {
	*cucumber.TestScenario
}

func (s *extender) iHaveCreatedAKafkaClusterAs(id string) error {
	session := s.Session()

	// lock the TestUser to avoid 2 scenarios with the same TestUser, creating the kafka concurrently.
	session.TestUser.Mu.Lock()
	defer session.TestUser.Mu.Unlock()

	apiClient := test.NewApiClient(s.Suite.Helper)
	kafkas, _, err := apiClient.DefaultApi.GetKafkas(session.TestUser.Ctx, &public.GetKafkasOpts{})
	if err != nil {
		return err
	}

	kafkaId := ""
	if len(kafkas.Items) != 0 {
		kafkaId = kafkas.Items[0].Id
	} else {
		kafka, _, err := apiClient.DefaultApi.CreateKafka(session.TestUser.Ctx, true, public.KafkaRequestPayload{
			Name:          "mykafka",
			CloudProvider: "aws",
			Region:        "us-east-1",
			MultiAz:       true,
		})
		if err != nil {
			return err
		}
		kafkaId = kafka.Id
	}
	s.Variables[id] = kafkaId

	return nil
}
