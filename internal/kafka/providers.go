package kafka

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"

	"github.com/goava/di"
)

func ConfigProviders() provider.Map {
	return provider.Map{
		"ServiceInjector": di.Provide(provider.Func(ServiceProviders)),
	}
}

func ServiceProviders(configContainer *di.Container) (provider.Map, error) {
	return provider.Map{
		"RouteLoader": di.Provide(routes.NewRouteLoader),
	}, nil
}
