package integration

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/cucumber/godog"
)

var helper *test.Helper
var teardown func()
var opts = cucumber.DefaultOptions()

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func TestMain(m *testing.M) {
	for _, arg := range os.Args[1:] {
		if arg == "-test.v=true" || arg == "-test.v" || arg == "-v" { // go test transforms -v option
			opts.Format = "pretty"
		}
	}

	flag.Parse()
	pflag.Parse()

	if len(pflag.Args()) != 0 {
		opts.Paths = pflag.Args()
	}

	// Startup all the services and mocks that are needed to test the
	// connector features.
	t := &testing.T{}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	helper, teardown = test.NewHelperWithHooksAndDBsetup(t, ocmServer,
		[]string{"INSERT INTO connector_types (id, name, checksum) VALUES ('OldConnectorTypeId', 'Old Connector Type', 'fakeChecksum1')",
			"INSERT INTO connector_type_labels (connector_type_id, label) VALUES ('OldConnectorTypeId', 'old_connector_type_label')",
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
			c.ConnectorCatalogDirs = []string{"./internal/connector/test/integration/resources/connector-catalog"}
			c.ConnectorMetadataDirs = []string{"./internal/connector/test/integration/resources/connector-metadata"}
			c.ConnectorEvalDuration, _ = time.ParseDuration("2s")
			c.ConnectorEvalOrganizations = []string{"13640210"}
			c.ConnectorNamespaceLifecycleAPI = true
			c.ConnectorEnableUnassignedConnectors = true
			// always set reconciler config to 1 second for connector tests
			reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
			// set sso provider if set in env
			if os.Getenv("SSO_PROVIDER_TYPE") == "redhat_sso" && os.Getenv("SSO_BASE_URL") != "" {
				kc.SelectSSOProvider = "redhat_sso"
				kc.SsoBaseUrl = os.Getenv("SSO_BASE_URL")
			}
		},
		connector.ConfigProviders(false),
	)
	defer teardown()

	os.Exit(m.Run())
}

func TestFeatures(t *testing.T) {
	g := gomega.NewWithT(t)

	for i := range opts.Paths {
		root := opts.Paths[i]

		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			g.Expect(err).To(gomega.BeNil())

			if info.IsDir() {
				return nil
			}

			name := filepath.Base(info.Name())
			ext := filepath.Ext(info.Name())

			if ext != ".feature" {
				return nil
			}

			testName := strings.TrimSuffix(name, ext)
			testName = strings.ReplaceAll(testName, "-", "_")

			t.Run(testName, func(t *testing.T) {
				// To preserve the current behavior, the test are market to be "safely" run in parallel, however
				// we may think to introduce a new naming convention i.e. files that ends with _parallel would
				// cause t.Parallel() to be invoked, other tests won't, so they won't be executed concurrently.
				//
				// This could help reducing/removing the need of explicit lock
				t.Parallel()

				o := opts
				o.TestingT = t
				o.Paths = []string{path.Join(root, info.Name())}

				s := cucumber.NewTestSuite(helper)

				status := godog.TestSuite{
					Name:                "connectors",
					Options:             &o,
					ScenarioInitializer: s.InitializeScenario,
				}.Run()

				g.Expect(status).To(gomega.Equal(0))
			})

			return nil
		})

		g.Expect(err).To(gomega.BeNil())
	}
}
