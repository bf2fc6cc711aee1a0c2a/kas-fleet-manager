package integration

import (
	"os"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
)

func TestMain(m *testing.M) {

	// Startup all the services and mocks that are needed to test the
	// connector features.
	t := &testing.T{}
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, teardown := test.NewHelperWithHooks(t, ocmServer,
		func(c *config.ConnectorsConfig, kc *keycloak.KeycloakConfig) {
			c.ConnectorCatalogDirs = []string{"./internal/connector/test/integration/connector-catalog"}
			c.ConnectorEvalDuration, _ = time.ParseDuration("30m")
			c.ConnectorEvalOrganizations = []string{"13640210"}

			kc.KeycloakClientExpire = true
		},
		connector.ConfigProviders(false),
	)
	defer teardown()

	status := cucumber.TestMain(h)
	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
