package test

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"net/http/httptest"
	"testing"

	gm "github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

// Register a test
// This should be run before every integration test
func RegisterIntegration(t *testing.T, server *httptest.Server) (*Helper, *openapi.APIClient, func()) {
	return RegisterIntegrationWithHooks(t, server, nil, nil)
}

type Hook func(helper *Helper)

// RegisterIntegrationWithHooks will init the Helper and start the server, and it allows to customize the configurations of the server via the hooks.
// The startHook will be invoked after the Helper object is inited but before the api server is started, which will allow caller to change configurations via the helper object.
// The teardownHook will be called before server is stopped, to allow the caller to reset configurations via the helper object.
func RegisterIntegrationWithHooks(t *testing.T, server *httptest.Server, startHook Hook, teardownHook Hook) (*Helper, *openapi.APIClient, func()) {
	// Register the test with gomega
	gm.RegisterTestingT(t)
	// Create a new helper
	helper := NewHelper(t, server)
	if startHook != nil {
		startHook(helper)
	}
	if server != nil && helper.Env().Config.OCM.MockMode == config.MockModeEmulateServer {
		helper.SetServer(server)
	}
	helper.Env().Config.ObservabilityConfiguration.EnableMock = true
	helper.StartServer()
	// Reset the database to a seeded blank state
	helper.ResetDB()
	// Start Leader Election Manager
	helper.StartLeaderElectionWorker()

	helper.ResetMetrics()
	// Create an api client
	client := helper.NewApiClient()
	return helper, client, buildTeardownHelperFn(helper, teardownHook)
}

func buildTeardownHelperFn(h *Helper, teardownHook Hook) func() {
	return func() {
		if teardownHook != nil {
			teardownHook(h)
		}
		h.StopServer()
		h.StopLeaderElectionWorker()
	}
}

func RegisterTestingT(t *testing.T) {
	gm.RegisterTestingT(t)
}
