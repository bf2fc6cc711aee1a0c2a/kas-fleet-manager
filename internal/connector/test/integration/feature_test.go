package integration

import (
	"os"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

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

	h, teardown := test.NewHelperWithHooksAndDBsetup(t, ocmServer,
		[]string{"INSERT INTO connector_types (id, name, checksum) VALUES ('OldConnectorTypeId', 'Old Connector Type', 'fakeChecksum1')",
			"INSERT INTO connector_types (id, name, checksum) VALUES ('OldConnectorTypeStillInUseId', 'Old Connector Type still in use', 'fakeChecksum2')",
			"INSERT INTO connectors (id, name, connector_type_id) VALUES ('ConnectorUsingOldTypeId', 'Connector using old type', 'OldConnectorTypeStillInUseId')",
			"INSERT INTO connector_types (id, name, checksum) VALUES ('log_sink_0.1', 'Log Sink', 'fakeChecksum')",
			"INSERT INTO connector_type_labels (connector_type_id, label) VALUES ('log_sink_0.1', 'old_label')",
			"INSERT INTO connector_channels (channel) VALUES ('old_channel')",
			"INSERT INTO connector_type_channels (connector_type_id, connector_channel_channel) VALUES ('log_sink_0.1', 'old_channel')",
			"INSERT INTO connector_type_capabilities (connector_type_id, capability) VALUES ('log_sink_0.1', 'old_capability')",

			"INSERT INTO connector_channels (channel) VALUES ('stable')",
			"INSERT INTO connector_shard_metadata (connector_type_id, channel) VALUES ('log_sink_0.1', 'stable')",
		},
		func(c *config.ConnectorsConfig, kc *keycloak.KeycloakConfig, reconcilerConfig *workers.ReconcilerConfig) {
			c.ConnectorCatalogDirs = []string{"./internal/connector/test/integration/connector-catalog"}
			c.ConnectorEvalDuration, _ = time.ParseDuration("2s")
			c.ConnectorEvalOrganizations = []string{"13640210"}
			c.ConnectorNamespaceLifecycleAPI = true
			c.ConnectorEnableUnassignedConnectors = true
			// always set reconciler config to 1 second for connector tests
			reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
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
