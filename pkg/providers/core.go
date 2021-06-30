package providers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/migrate"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/serve"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	customOcm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
)

func CoreConfigProviders() di.Option {
	return di.Options(
		di.Provide(func(env *environments.Env) config.EnvName {
			return config.EnvName(env.Name)
		}),

		// Add config types
		di.Provide(config.NewHealthCheckConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewDatabaseConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewServerConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewOCMConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewKeycloakConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewAccessControlListConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewMetricsConfig, di.As(new(provider.ConfigModule))),

		// Add common CLI sub commands
		di.Provide(serve.NewServeCommand),
		di.Provide(migrate.NewMigrateCommand),

		// Add other core config providers..
		vault.ConfigProviders(),
		sentry.ConfigProviders(),
		signalbus.ConfigProviders(),

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
