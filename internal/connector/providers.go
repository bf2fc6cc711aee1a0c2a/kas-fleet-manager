package connector

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	coreWorkers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	coreConfig "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"

	"github.com/goava/di"
)

func ConfigProviders() provider.Map {
	return provider.Map{
		"ConfigModule":    di.Provide(config.NewConnectorsConfig, di.As(new(coreConfig.ConfigModule))),
		"ServiceInjector": di.Provide(provider.Func(ServiceProviders)),
	}
}

func ServiceProviders(configContainer *di.Container) (provider.Map, error) {
	return provider.Map{
		"Config": di.Provide(func() (value *config.ConnectorsConfig, err error) {
			err = configContainer.Resolve(&value)
			return
		}),
		"ConnectorsService":       di.Provide(services.NewConnectorsService, di.As(new(services.ConnectorsService))),
		"ConnectorTypesService":   di.Provide(services.NewConnectorTypesService, di.As(new(services.ConnectorTypesService))),
		"ConnectorClusterService": di.Provide(services.NewConnectorClusterService, di.As(new(services.ConnectorClusterService))),
		"ConnectorTypesHandler":   di.Provide(handlers.NewConnectorTypesHandler),
		"ConnectorsHandler":       di.Provide(handlers.NewConnectorsHandler),
		"ConnectorClusterHandler": di.Provide(handlers.NewConnectorClusterHandler),
		"RouteLoader":             di.Provide(routes.NewRouteLoader),
		"ConnectorManager":        di.Provide(workers.NewConnectorManager, di.As(new(coreWorkers.Worker))),
	}, nil
}
