package cucumber_test

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/cucumber/godog"
	"testing"
)

// In this example no scenarios are actually run because the "features" subdirectory does not exist.
func Example() {

	// Typically added to a TestMain function like:
	// func TestMain(m *testing.M)
	{
		ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
		defer ocmServer.Close()
		h, _, teardown := test.RegisterIntegration(&testing.T{}, ocmServer)
		defer teardown()
		cucumber.TestMain(h)
	}

	// Output:
	// Setting OCM base URL to http://127.0.0.1:9876
}

type extender struct {
	*cucumber.TestScenario
}

func (s *extender) debug(as string) error {
	fmt.Println(s.Expand(as))
	return nil
}

// You can also add additional step implementations that have access to the scenario state.
func Example_customSteps() {

	// With extender defined as:
	//
	//  type extender struct {
	//  	*cucumber.TestScenario
	//  }
	//
	//  func (s *extender) debug(as string) error {
	//  	fmt.Println(s.Expand(as))
	//  	return nil
	//  }

	cucumber.StepModules = append(cucumber.StepModules, func(ctx *godog.ScenarioContext, s *cucumber.TestScenario) {
		e := &extender{s}
		ctx.Step(`^debug "([^"]*)"$`, e.debug)
	})

}
