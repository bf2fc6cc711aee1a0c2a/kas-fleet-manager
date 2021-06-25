package environments

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	customOcm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(func(env *Env) config.EnvName {
			return config.EnvName(env.Name)
		}),

		// Add the env types
		di.Provide(newDevelopmentEnvLoader, di.Tags{"env": DevelopmentEnv}),
		di.Provide(newProductionEnvLoader, di.Tags{"env": ProductionEnv}),
		di.Provide(newStageEnvLoader, di.Tags{"env": StageEnv}),
		di.Provide(newIntegrationEnvLoader, di.Tags{"env": IntegrationEnv}),
		di.Provide(newTestingEnvLoader, di.Tags{"env": TestingEnv}),

		// Add config types
		di.Provide(config.NewApplicationConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewMetricsConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewHealthCheckConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewDatabaseConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewKasFleetshardConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewServerConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewOCMConfig, di.As(new(provider.ConfigModule))),
		di.Provide(func(c *config.ApplicationConfig) *config.OSDClusterConfig { return c.OSDClusterConfig }),
		di.Provide(func(c *config.ApplicationConfig) *config.KafkaConfig { return c.Kafka }),
		di.Provide(func(c *config.ApplicationConfig) *config.KeycloakConfig { return c.Keycloak }),
		di.Provide(func(c *config.ApplicationConfig) *config.AccessControlListConfig { return c.AccessControlList }),
		di.Provide(func(c *config.ApplicationConfig) *config.AWSConfig { return c.AWS }),
		di.Provide(func(c *config.ApplicationConfig) *config.ObservabilityConfiguration {
			return c.ObservabilityConfiguration
		}),
		di.Provide(provider.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(

		// provide the service constructors
		di.Provide(db.NewConnectionFactory),
		di.Provide(observatorium.NewObservatoriumClient),
		di.Provide(ocm.NewOCMClient),
		di.Provide(customOcm.NewClient),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(services.NewClusterService),
		di.Provide(services.NewConfigService),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(services.NewKafkaService, di.As(new(services.KafkaService))),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewKasFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneKafkaService, di.As(new(services.DataPlaneKafkaService))),
		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
		di.Provide(func(c *config.KeycloakConfig) services.KafkaKeycloakService {
			return services.NewKeycloakService(c, c.KafkaRealm)
		}),
		di.Provide(func(c *config.KeycloakConfig) services.OsdKeycloakService {
			return services.NewKeycloakService(c, c.OSDClusterIDPRealm)
		}),
		di.Provide(func(client *ocm.Client) (ocmClient *sdkClient.Connection) {
			return client.Connection
		}),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(provider.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(provider.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(provider.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(provider.BootService))),
	)
}
