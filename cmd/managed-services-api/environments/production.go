package environments

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
)

var productionConfigDefaults map[string]string = map[string]string{
	"v":               "1",
	"ocm-debug":       "false",
	"enable-ocm-mock": "false",
	"enable-sentry":   "true",
}

func loadProduction(env *Env) error {
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
