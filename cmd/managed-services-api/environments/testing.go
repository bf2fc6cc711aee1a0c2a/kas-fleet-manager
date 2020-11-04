package environments

import (
	"os"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
)

var testingConfigDefaults map[string]string = map[string]string{
	"v":                    "0",
	"logtostderr":          "true",
	"ocm-base-url":         "https://api-integration.6943.hive-integration.openshiftapps.com",
	"enable-https":         "false",
	"enable-metrics-https": "false",
	"enable-authz":         "true",
	"ocm-debug":            "false",
	"enable-ocm-mock":      "true",
	"enable-sentry":        "false",
}

// The testing environment is specifically for automated testing
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func loadTesting(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	// Support a one-off env to allow enabling db debug in testing
	if os.Getenv("DB_DEBUG") == "true" {
		env.Config.Database.Debug = true
	}

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
