package kafka

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/migrations"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafkatlscertmgmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/cluster_mgrs"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/kafka_mgrs/promotion"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	observatoriumClient "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	environments2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/providers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"
	"github.com/goava/di"
)

func EnvConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": environments2.DevelopmentEnv}),
		di.Provide(environments.NewProductionEnvLoader, di.Tags{"env": environments2.ProductionEnv}),
		di.Provide(environments.NewStageEnvLoader, di.Tags{"env": environments2.StageEnv}),
		di.Provide(environments.NewIntegrationEnvLoader, di.Tags{"env": environments2.IntegrationEnv}),
		di.Provide(environments.NewTestingEnvLoader, di.Tags{"env": environments2.TestingEnv}),
	)
}

func ConfigProviders() di.Option {
	return di.Options(

		EnvConfigProviders(),
		providers.CoreConfigProviders(),

		// Configuration for the Kafka service...
		di.Provide(config.NewAWSConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewGCPConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),

		di.Provide(config.NewSupportedProvidersConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),
		di.Provide(observatoriumClient.NewObservabilityConfigurationConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),
		di.Provide(config.NewKafkaConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),
		di.Provide(config.NewDataplaneClusterConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),
		di.Provide(config.NewKasFleetshardConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(quota_management.NewQuotaManagementListConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewCertificateManagementConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),

		// Additional CLI subcommands
		di.Provide(environments2.Func(ServiceProviders)),
		di.Provide(migrations.New),

		metrics.ConfigProviders(),
	)
}

func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(services.NewClusterService),
		di.Provide(services.NewKafkaService, di.As(new(services.KafkaService))),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewSupportedKafkaInstanceTypesService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewKasFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneKafkaService, di.As(new(services.DataPlaneKafkaService))),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(routes.NewRouteLoader),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(cluster_mgrs.NewClusterManager, di.As(new(workers.Worker))),
		di.Provide(cluster_mgrs.NewDynamicScaleUpManager, di.As(new(workers.Worker))),
		di.Provide(cluster_mgrs.NewCleanupClustersManager, di.As(new(workers.Worker))),
		di.Provide(cluster_mgrs.NewDeprovisioningClustersManager, di.As(new(workers.Worker))),
		di.Provide(cluster_mgrs.NewDynamicScaleDownManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewAcceptedKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewPreparingKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewDeletingKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewProvisioningKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewReadyKafkaManager, di.As(new(workers.Worker))),
		di.Provide(kafka_mgrs.NewKafkaCNAMEManager, di.As(new(workers.Worker))),
		di.Provide(promotion.NewPromotionKafkaManager, di.As(new(workers.Worker))),
		di.Provide(acl.NewEnterpriseClustersAccessControlMiddleware),
		di.Provide(kafkatlscertmgmt.NewKafkaTLSCertificateManagementService),
	)
}
