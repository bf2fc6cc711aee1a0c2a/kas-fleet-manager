package environments

import (
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

var integrationConfigDefaults map[string]string = map[string]string{
	"v":                                 "0",
	"logtostderr":                       "true",
	"ocm-base-url":                      "https://api-integration.6943.hive-integration.openshiftapps.com",
	"enable-https":                      "false",
	"enable-metrics-https":              "false",
	"enable-authz":                      "true",
	"ocm-debug":                         "false",
	"enable-ocm-mock":                   "true",
	"ocm-mock-mode":                     config.MockModeEmulateServer,
	"enable-sentry":                     "false",
	"enable-deny-list":                  "true",
	"enable-instance-limit-control":     "true",
	"max-allowed-instances":             "1",
	"mas-sso-base-url":                  "https://keycloak-mas-sso-stage.apps.app-sre-stage-0.k3s7.p1.openshiftapps.com",
	"mas-sso-realm":                     "rhoas",
	"osd-idp-mas-sso-realm":             "rhoas-kafka-sre",
	"enable-kafka-external-certificate": "false",
	"cluster-compute-machine-type":      "m5.xlarge",
	"ingress-controller-replicas":       "3",
	"enable-quota-service":              "false",
	"enable-dynamic-scaling":            "false",
	"enable-deletion-of-expired-kafka":  "false",
	"dataplane-cluster-scaling-type":    "auto",
}

// The integration environment is specifically for automated integration testing using an emulated server
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func loadIntegration(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	// Support a one-off env to allow enabling db debug in testing
	if os.Getenv("DB_DEBUG") == "true" {
		env.Config.Database.Debug = true
	}

	env.Config.ObservabilityConfiguration.EnableMock = true

	err := env.LoadClients()
	if err != nil {
		return err
	}
	err = env.LoadServices()
	if err != nil {
		return err
	}

	return env.InitializeSentry()
}
