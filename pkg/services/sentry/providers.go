package sentry

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(NewConfig, di.As(new(provider.ConfigModule))),
		di.ProvideValue(provider.AfterCreateServicesHook{
			Func: Initialize,
		}),
	)
}
