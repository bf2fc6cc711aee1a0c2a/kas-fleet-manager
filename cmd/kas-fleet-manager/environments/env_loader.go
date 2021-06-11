package environments

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

type EnvLoader interface {
	Defaults() map[string]string
	Load(env *Env) error
}

type SimpleEnvLoader map[string]string

var _ EnvLoader = SimpleEnvLoader{}

func (b SimpleEnvLoader) Defaults() map[string]string {
	return b
}

func (b SimpleEnvLoader) Load(env *Env) error {
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
