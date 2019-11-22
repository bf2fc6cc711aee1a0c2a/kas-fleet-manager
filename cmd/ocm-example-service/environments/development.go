package environments

import (
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/db"
)

// The development environment is intended for use while developing features, requiring manual verification
var developmentConfigDefaults map[string]string = map[string]string{
	"v":                      "10",
	"enable-authz":           "true",
	"ocm-debug":              "true",
	"enable-ocm-mock":        "false",
	"enable-https":           "false",
	"enable-metrics-https":   "false",
	"api-server-hostname":    "localhost",
	"api-server-bindaddress": "localhost:8000",
	"enable-sentry":          "false",
}

func loadDevelopment(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	err := env.LoadClients()
	if err != nil {
		return err
	}
	env.LoadServices()

	return env.InitializeSentry()
}
