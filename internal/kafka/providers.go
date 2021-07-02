package kafka

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/cloudprovider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/cluster"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cmd/serviceaccounts"
	config2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/migrations"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers"
	"github.com/goava/di"
)

func EnvConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": constants.DevelopmentEnv}),
		di.Provide(environments.NewProductionEnvLoader, di.Tags{"env": constants.ProductionEnv}),
		di.Provide(environments.NewStageEnvLoader, di.Tags{"env": constants.StageEnv}),
		di.Provide(environments.NewIntegrationEnvLoader, di.Tags{"env": constants.IntegrationEnv}),
		di.Provide(environments.NewTestingEnvLoader, di.Tags{"env": constants.TestingEnv}),
	)
}

func ConfigProviders() di.Option {
	return di.Options(

		EnvConfigProviders(),
		providers.CoreConfigProviders(),

		// Configuration for the Kafka service...
		di.Provide(config.NewAWSConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewSupportedProvidersConfig, di.As(new(provider.ConfigModule)), di.As(new(provider.ServiceValidator))),
		di.Provide(config.NewObservabilityConfigurationConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config2.NewKafkaConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewDataplaneClusterConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config2.NewKasFleetshardConfig, di.As(new(provider.ConfigModule))),

		// Additional CLI subcommands
		di.Provide(cluster.NewClusterCommand),
		di.Provide(kafka.NewKafkaCommand),
		di.Provide(cloudprovider.NewCloudProviderCommand),
		di.Provide(observatorium.NewRunObservatoriumCommand),
		di.Provide(serviceaccounts.NewServiceAccountCommand),
		di.Provide(errors.NewErrorsCommand),
		di.Provide(provider.Func(ServiceProviders)),
		di.Provide(migrations.New),
	)
}

func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(services.NewClusterService),
		di.Provide(services.NewKafkaService, di.As(new(services.KafkaService))),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewKasFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneKafkaService, di.As(new(services.DataPlaneKafkaService))),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(routes.NewRouteLoader),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(workers.NewClusterManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewAcceptedKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewPreparingKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewDeletingKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewProvisioningKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewReadyKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewKafkaCNAMEManager, di.As(new(workers.Worker))),
	)
}
