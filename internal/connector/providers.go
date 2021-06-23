package connector

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	coreWorkers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(config.NewConnectorsConfig, di.As(new(provider.ConfigModule))),
		di.Provide(provider.Func(ServiceProviders)),
	)
}

func ServiceProviders(configContainer *di.Container) di.Option {
	return di.Options(
		di.Provide(func() (value *config.ConnectorsConfig, err error) {
			err = configContainer.Resolve(&value)
			return
		}),
		di.Provide(services.NewConnectorsService, di.As(new(services.ConnectorsService))),
		di.Provide(services.NewConnectorTypesService, di.As(new(services.ConnectorTypesService))),
		di.Provide(services.NewConnectorClusterService, di.As(new(services.ConnectorClusterService))),
		di.Provide(handlers.NewConnectorTypesHandler),
		di.Provide(handlers.NewConnectorsHandler),
		di.Provide(handlers.NewConnectorClusterHandler),
		di.Provide(routes.NewRouteLoader),
		di.Provide(workers.NewConnectorManager, di.As(new(coreWorkers.Worker))),
	)
}
