package environments

import (
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
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
		"v":                                 "0",
		"logtostderr":                       "true",
		"ocm-base-url":                      "https://api-integration.6943.hive-integration.openshiftapps.com",
		"ams-base-url":                      "https://api-integration.6943.hive-integration.openshiftapps.com",
		"enable-https":                      "false",
		"enable-metrics-https":              "false",
		"enable-terms-acceptance":           "false",
		"ocm-debug":                         "false",
		"enable-ocm-mock":                   "true",
		"ocm-mock-mode":                     ocm.MockModeEmulateServer,
		"enable-sentry":                     "false",
		"enable-deny-list":                  "true",
		"enable-access-list":                "false",
		"enable-instance-limit-control":     "true",
		"max-allowed-instances":             "1",
		"mas-sso-base-url":                  "http://127.0.0.1:8180",
		"mas-sso-realm":                     "rhoas",
		"osd-idp-mas-sso-realm":             "rhoas-kafka-sre",
		"enable-kafka-external-certificate": "false",
		"aws-cluster-compute-machine-type":  "m5.xlarge",
		"allow-developer-instance":          "true",
		"quota-type":                        "quota-management-list",
		"enable-deletion-of-expired-kafka":  "true",
		"dataplane-cluster-scaling-type":    "manual",
		"strimzi-operator-addon-id":         "managed-kafka-qe",
		"kas-fleetshard-addon-id":           "kas-fleetshard-operator-qe",
		"admin-api-sso-base-url":            "http://127.0.0.1:8180",
		"admin-api-sso-endpoint-uri":        "/auth/realms/rhoas-kafka-sre",
		"admin-api-sso-realm":               "rhoas-kafka-sre",
	}
}

// The integration environment is specifically for automated integration testing using an emulated server
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (b IntegrationEnvLoader) ModifyConfiguration(env *environments.Env) error {
	// Support a one-off env to allow enabling db debug in testing
	var databaseConfig *db.DatabaseConfig
	var observabilityConfiguration *observatorium.ObservabilityConfiguration
	env.MustResolveAll(&databaseConfig, &observabilityConfiguration)

	if os.Getenv("DB_DEBUG") == "true" {
		databaseConfig.Debug = true
	}
	observabilityConfiguration.EnableMock = true
	return nil
}
