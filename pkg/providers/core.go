package providers

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/cmd/migrate"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/cmd/serve"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/quota_management"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/account"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/authorization"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/goava/di"
)

func CoreConfigProviders() di.Option {
	return di.Options(
		di.Provide(func(env *environments.Env) environments.EnvName {
			return environments.EnvName(env.Name)
		}),

		// Add config types
		di.Provide(server.NewHealthCheckConfig, di.As(new(environments.ConfigModule))),
		di.Provide(db.NewDatabaseConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewServerConfig, di.As(new(environments.ConfigModule))),
		di.Provide(ocm.NewOCMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(keycloak.NewKeycloakConfig, di.As(new(environments.ConfigModule))),
		di.Provide(acl.NewAccessControlListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(quota_management.NewQuotaManagementListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewMetricsConfig, di.As(new(environments.ConfigModule))),

		// Add common CLI sub commands
		di.Provide(serve.NewServeCommand),
		di.Provide(migrate.NewMigrateCommand),

		// Add other core config providers..
		sentry.ConfigProviders(),
		authorization.ConfigProviders(),
		account.ConfigProviders(),

		di.Provide(environments.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(

		// provide the service constructors
		di.Provide(db.NewConnectionFactory),
		di.Provide(observatorium.NewObservatoriumClient),

		di.Provide(func(config *ocm.OCMConfig) ocm.ClusterManagementClient {
			conn, _, err := ocm.NewOCMConnection(config, config.BaseURL)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(func(config *ocm.OCMConfig) ocm.AMSClient {
			conn, _, err := ocm.NewOCMConnection(config, config.AmsUrl)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(aws.NewDefaultClientFactory, di.As(new(aws.ClientFactory))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
		di.Provide(func(c *keycloak.KeycloakConfig) services.DinosaurKeycloakService {
			return services.NewKeycloakService(c, c.DinosaurRealm)
		}),
		di.Provide(func(c *keycloak.KeycloakConfig) services.OsdKeycloakService {
			return services.NewKeycloakService(c, c.OSDClusterIDPRealm)
		}),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(environments.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(environments.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(environments.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(environments.BootService))),
	)
}
