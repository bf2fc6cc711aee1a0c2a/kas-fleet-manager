package sentry

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
)

func ConfigProviders() provider.Map {
	return provider.Map{
		"Config":          di.Provide(NewConfig, di.As(new(provider.ConfigModule))),
		"ServiceInjector": di.Provide(provider.Func(ServiceProviders)),
		"Initialize": di.ProvideValue(provider.AfterCreateServicesHook{
			Func: Initialize,
		}),
	}
}

func ServiceProviders(configContainer *di.Container) (provider.Map, error) {
	return provider.Map{
		"Config": di.Provide(func() (value *Config, err error) {
			err = configContainer.Resolve(&value)
			return
		}),
	}, nil
}
