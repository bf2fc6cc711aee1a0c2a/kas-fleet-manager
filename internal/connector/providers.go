package connector

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/migrations"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers"
	coreWorkers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/goava/di"
)

func ConfigProviders(kafkaEnabled bool) di.Option {

	result := di.Options(
		di.Provide(config.NewConnectorsConfig, di.As(new(provider.ConfigModule))),
		di.Provide(provider.Func(serviceProviders)),
		di.Provide(migrations.New),
	)

	// If we are not running in the kas-fleet-manager.. we need to inject more types into the DI container
	if !kafkaEnabled {
		result = di.Options(
			di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": constants.DevelopmentEnv}),
			di.Provide(environments.NewProductionEnvLoader, di.Tags{"env": constants.ProductionEnv}),
			di.Provide(environments.NewStageEnvLoader, di.Tags{"env": constants.StageEnv}),
			di.Provide(environments.NewIntegrationEnvLoader, di.Tags{"env": constants.IntegrationEnv}),
			di.Provide(environments.NewTestingEnvLoader, di.Tags{"env": constants.TestingEnv}),
			providers.CoreConfigProviders(),
			result,
			di.Provide(provider.Func(serviceProvidersNoKafka)),
		)
	}

	return result
}

func serviceProviders() di.Option {
	return di.Options(
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

func serviceProvidersNoKafka() di.Option {
	return di.Options(
		di.Provide(handlers.NewAuthenticationBuilder),
	)
}
