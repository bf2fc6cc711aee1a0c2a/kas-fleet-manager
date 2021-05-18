package cucumber_test

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/cucumber/godog"
	"os"
	"testing"
)

// In this example no scenarios are actually run because the "features" subdirectory does not exist.
func Example() {

	// Typically added to a TestMain function like:
	// func TestMain(m *testing.M)
	{
		_ = os.Setenv("OCM_ENV", "integration")
		ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
		defer ocmServer.Close()
		h, _, teardown := test.RegisterIntegration(&testing.T{}, ocmServer)
		defer teardown()

		cucumber.TestMain(h)
	}

	// Output:
	// I0306 09:56:53.618514    6608 environment.go:117] Initializing integration environment
	// I0306 09:56:53.628052    6608 environment.go:222] Using Mock Observatorium Client
	// I0306 09:56:53.628052    6608 environment.go:247] Disabling Sentry error reporting
	// I0306 09:56:53.629113    6608 metrics_server.go:48] start metrics server
	// I0306 09:56:53.629113    6608 metrics_server.go:62] Serving Metrics without TLS at localhost:8080
	// I0306 09:56:53.629113    6608 healthcheck_server.go:56] Serving HealthCheck without TLS at localhost:8083
	// I0306 09:56:53.629113    6608 environment.go:222] Using Mock Observatorium Client
	// I0306 09:56:53.634369    6608 api_server.go:275] Serving without TLS at localhost:8000
	//
	//
	// No scenarios
	// No steps
	// 0s
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
