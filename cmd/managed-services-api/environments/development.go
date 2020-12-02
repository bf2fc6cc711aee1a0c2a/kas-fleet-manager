package environments

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
)

// The development environment is intended for use while developing features, requiring manual verification
var developmentConfigDefaults map[string]string = map[string]string{
	"v":                      "10",
	"enable-authz":           "true",
	"ocm-debug":              "true",
	"ocm-base-url":           "https://api.stage.openshift.com",
	"enable-ocm-mock":        "false",
	"enable-https":           "false",
	"enable-metrics-https":   "false",
	"api-server-hostname":    "localhost",
	"api-server-bindaddress": "localhost:8000",
	"enable-sentry":          "false",
	"enable-allow-list":      "true",
	"max-allowed-instances":  "1",
	"auto-osd-creation":      "false",
}

func loadDevelopment(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

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
