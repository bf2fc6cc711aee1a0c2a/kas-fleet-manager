package providers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/segment"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/cmd/migrate"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/cmd/serve"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"

	"github.com/segmentio/analytics-go/v3"
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
		di.Provide(keycloak.NewKeycloakConfig, di.As(new(environments.ConfigModule)), di.As(new(environments.ServiceValidator))),
		di.Provide(acl.NewAccessControlListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewMetricsConfig, di.As(new(environments.ConfigModule))),
		di.Provide(workers.NewReconcilerConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewContextConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewAdminAuthZConfig, di.As(new(environments.ConfigModule)), di.As(new(environments.ServiceValidator))),

		// Add common CLI sub commands
		di.Provide(serve.NewServeCommand),
		di.Provide(migrate.NewMigrateCommand),

		// Add other core config providers..
		sentry.ConfigProviders(),
		signalbus.ConfigProviders(),
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
		di.Provide(func(c *keycloak.KeycloakConfig) sso.KafkaKeycloakService {
			return sso.NewKeycloakServiceBuilder().
				ForKFM().
				WithConfiguration(c).
				Build()
		}),
		di.Provide(func(c *keycloak.KeycloakConfig) sso.OsdKeycloakService {
			return sso.NewKeycloakServiceBuilder().
				ForOSD().
				WithConfiguration(c).
				WithRealmConfig(c.OSDClusterIDPRealm).
				Build()
		}),

		di.Provide(func() *segment.SegmentClientFactory {
			// The API Key should be read from a secret file
			// For more information on analytics.Config fields, see https://pkg.go.dev/gopkg.in/segmentio/analytics-go.v3#Config
			//
			// Note on API errors: Segment API returns a 200 response for all API requests apart from certain errors which return 400
			// see https://segment.com/docs/connections/sources/catalog/libraries/server/http-api/#errors for more details.
			// e.g. The API will return a 200 even if the API Key is invalid.
			segmentClient, err := segment.NewClientWithConfig("API_KEY", analytics.Config{
				// Determines when the client will upload all messages in the queue.
				// By default this is set to 20. The client will only send messages to Segment if the queue reaches this number or if the
				// Interval was reached first (by default this is every 5 seconds)
				BatchSize: 1,
				Logger:    logger.Logger,
				Verbose:   true,
			})
			if err != nil {
				logger.Logger.Error(err)
			}
			return segmentClient
		}),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(environments.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(environments.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(environments.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(environments.BootService))),
	)
}
