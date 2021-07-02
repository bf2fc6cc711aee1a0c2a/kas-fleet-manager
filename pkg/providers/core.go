package providers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	migrate2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/cmd/migrate"
	serve2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/cmd/serve"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
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
		di.Provide(serve2.NewServeCommand),
		di.Provide(migrate2.NewMigrateCommand),

		// Add other core config providers..
		vault.ConfigProviders(),
		sentry.ConfigProviders(),
		signalbus.ConfigProviders(),
		authorization.ConfigProviders(),

		di.Provide(provider.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(

		// provide the service constructors
		di.Provide(db.NewConnectionFactory),
		di.Provide(observatorium.NewObservatoriumClient),

		di.Provide(ocm.NewOCMConnection),
		di.Provide(ocm.NewClient),

		di.Provide(aws.NewDefaultClientFactory, di.As(new(aws.ClientFactory))),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
		di.Provide(func(c *config.KeycloakConfig) services.KafkaKeycloakService {
			return services.NewKeycloakService(c, c.KafkaRealm)
		}),
		di.Provide(func(c *config.KeycloakConfig) services.OsdKeycloakService {
			return services.NewKeycloakService(c, c.OSDClusterIDPRealm)
		}),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(provider.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(provider.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(provider.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(provider.BootService))),
	)
}
