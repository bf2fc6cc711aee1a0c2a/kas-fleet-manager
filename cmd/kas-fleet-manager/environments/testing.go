package environments

import (
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

type TestingEnvLoader struct{}

var _ EnvLoader = TestingEnvLoader{}

func newTestingEnvLoader() EnvLoader {
	return TestingEnvLoader{}
}

func (t TestingEnvLoader) Defaults() map[string]string {
	return map[string]string{}
}

// The testing environment is specifically for automated testing
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (t TestingEnvLoader) Load(env *Env) error {
	env.DBFactory = db.NewMockConnectionFactory(env.Config.Database)

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
