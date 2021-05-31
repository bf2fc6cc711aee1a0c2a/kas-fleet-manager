// Ensures the user has at least one kafka cluster created and stores it's id in a scenario variable:
//    Given I have created a kafka cluster as ${kid}
package cucumber

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/cucumber/godog"
)

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^I have created a kafka cluster as \${([^"]*)}$`, s.iHaveCreatedAKafkaClusterAs)
	})
}

func (s *TestScenario) iHaveCreatedAKafkaClusterAs(id string) error {
	session := s.Session()

	// lock the TestUser to avoid 2 scenarios with the same TestUser, creating the kafka concurrently.
	session.TestUser.Mu.Lock()
	defer session.TestUser.Mu.Unlock()

	apiClient := s.Suite.Helper.NewApiClient()
	kafkas, _, err := apiClient.DefaultApi.GetKafkas(session.TestUser.Ctx, &openapi.GetKafkasOpts{})
	if err != nil {
		return err
	}

	kafkaId := ""
	if len(kafkas.Items) != 0 {
		kafkaId = kafkas.Items[0].Id
	} else {
		kafka, _, err := apiClient.DefaultApi.CreateKafka(session.TestUser.Ctx, true, openapi.KafkaRequestPayload{
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
