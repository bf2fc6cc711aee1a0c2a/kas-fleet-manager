package environments

import (
	"os"
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
	// Support a one-off env to allow enabling db debug in testing
	if os.Getenv("DB_DEBUG") == "true" {
		env.Config.Database.Debug = true
	}
	return env.LoadServices()
}
