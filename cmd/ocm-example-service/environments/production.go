package environments

import (
	"gitlab.cee.redhat.com/service/ocm-example-service/pkg/db"
)

var productionConfigDefaults map[string]string = map[string]string{
	"v":                     "1",
	"ocm-authz-debug":       "false",
	"enable-ocm-authz-mock": "false",
	"enable-sentry":         "true",
}

func loadProduction(env *Env) error {
	env.DBFactory = db.NewConnectionFactory(env.Config.Database)

	err := env.LoadClients()
	if err != nil {
		return err
	}
	env.LoadServices()

	return env.InitializeSentry()
}
