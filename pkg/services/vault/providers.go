package vault

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
)

func ConfigProviders() provider.Map {
	return provider.Map{
		"Config":          di.Provide(NewConfig, di.As(new(config.ConfigModule))),
		"ServiceInjector": di.Provide(provider.Func(ServiceProviders)),
	}
}

func ServiceProviders(configContainer *di.Container) (provider.Map, error) {
	return provider.Map{
		"Config": di.Provide(func() (value *Config, err error) {
			err = configContainer.Resolve(&value)
			return
		}),
		"VaultService": di.Provide(NewVaultService),
	}, nil
}
