package environments

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"os"
)

type TestingEnvLoader struct{}

var _ environments.EnvLoader = TestingEnvLoader{}

func NewTestingEnvLoader() environments.EnvLoader {
	return TestingEnvLoader{}
}

func (t TestingEnvLoader) Defaults() map[string]string {
	return map[string]string{}
}

// The testing environment is specifically for automated testing
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (t TestingEnvLoader) ModifyConfiguration(env *environments.Env) error {
	// Support a one-off env to allow enabling db debug in testing

	var databaseConfig *db.DatabaseConfig
	env.MustResolveAll(&databaseConfig)

	if os.Getenv("DB_DEBUG") == "true" {
		databaseConfig.Debug = true
	}
	return nil
}
