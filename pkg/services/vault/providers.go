package vault

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(NewConfig, di.As(new(provider.ConfigModule))),
		di.Provide(provider.Func(ServiceProviders)),
	)
}

func ServiceProviders(configContainer *di.Container) di.Option {
	return di.Options(
		di.Provide(func() (value *Config, err error) {
			err = configContainer.Resolve(&value)
			return
		}),
		di.Provide(NewVaultService),
	)
}
