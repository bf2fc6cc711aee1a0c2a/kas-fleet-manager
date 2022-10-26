package environments

import (
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
)

type IntegrationEnvLoader struct{}

var _ environments.EnvLoader = IntegrationEnvLoader{}

func NewIntegrationEnvLoader() environments.EnvLoader {
	return IntegrationEnvLoader{}
}

func (b IntegrationEnvLoader) Defaults() map[string]string {
	return map[string]string{
		"v":                          "0",
		"logtostderr":                "true",
		"ocm-base-url":               "https://api-integration.6943.hive-integration.openshiftapps.com",
		"enable-https":               "false",
		"enable-metrics-https":       "false",
		"enable-terms-acceptance":    "false",
		"ocm-debug":                  "false",
		"enable-ocm-mock":            "true",
		"ocm-mock-mode":              ocm.MockModeEmulateServer,
		"enable-sentry":              "false",
		"enable-deny-list":           "true",
		"sso-provider-type":          "mas_sso",
		"enable-access-list":         "false",
		"mas-sso-base-url":           "http://127.0.0.1:8180",
		"redhat-sso-base-url":        "https://sso.stage.redhat.com",
		"mas-sso-realm":              "rhoas",
		"connector-eval-duration":    "48h",
		"osd-idp-mas-sso-realm":      "rhoas-kafka-sre",
		"admin-api-sso-base-url":     "http://127.0.0.1:8180",
		"admin-api-sso-endpoint-uri": "/auth/realms/rhoas-kafka-sre",
		"admin-api-sso-realm":        "rhoas-kafka-sre",
	}
}

// The integration environment is specifically for automated integration testing using an emulated server
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (b IntegrationEnvLoader) ModifyConfiguration(env *environments.Env) error {
	// Support a one-off env to allow enabling db debug in testing
	var databaseConfig *db.DatabaseConfig
	env.MustResolveAll(&databaseConfig)
	if os.Getenv("DB_DEBUG") == "true" {
		databaseConfig.Debug = true
	}
	return nil
}
