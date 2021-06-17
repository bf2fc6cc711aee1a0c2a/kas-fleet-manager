package environments

import (
	"context"
	goerrors "errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	sentry2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	customOcm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

const (
	TestingEnv           string = "testing"
	DevelopmentEnv       string = "development"
	ProductionEnv        string = "production"
	StageEnv             string = "stage"
	IntegrationEnv       string = "integration"
	EnvironmentStringKey string = "OCM_ENV"
	EnvironmentDefault   string = DevelopmentEnv
)

type Env struct {
	Name             string
	Config           *config.ApplicationConfig
	ConfigContainer  *di.Container
	ServiceContainer *di.Container
}

func GetEnvironmentStrFromEnv() string {
	envStr, specified := os.LookupEnv(EnvironmentStringKey)
	if !specified || envStr == "" {
		glog.Infof("Environment variable %q not specified, using default %q", EnvironmentStringKey, EnvironmentDefault)
		envStr = EnvironmentDefault
	}
	return envStr
}

// Adds environment flags, using the environment's config struct, to the flagset 'flags'
func (e *Env) AddFlags(flags *pflag.FlagSet) error {

	var namedEnv EnvLoader
	err := e.ConfigContainer.Resolve(&namedEnv, di.Tags{"env": e.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", e.Name)
	}

	modules := []provider.ConfigModule{}
	if err := e.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return err
	}
	for i := range modules {
		modules[i].AddFlags(flags)
	}

	return setConfigDefaults(flags, namedEnv.Defaults())
}

func NewEnv(name string, options ...di.Option) (env *Env, err error) {
	env = &Env{
		Name: name,
	}

	//di.SetTracer(di.StdTracer{})
	env.ConfigContainer, err = di.New(append(options,
		di.ProvideValue(env),
		di.ProvideValue(config.EnvName(env.Name)),

		// Add the env types
		di.Provide(newDevelopmentEnvLoader, di.Tags{"env": DevelopmentEnv}),
		di.Provide(newProductionEnvLoader, di.Tags{"env": ProductionEnv}),
		di.Provide(newStageEnvLoader, di.Tags{"env": StageEnv}),
		di.Provide(newIntegrationEnvLoader, di.Tags{"env": IntegrationEnv}),
		di.Provide(newTestingEnvLoader, di.Tags{"env": TestingEnv}),

		di.Provide(config.NewApplicationConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewMetricsConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewHealthCheckConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewDatabaseConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewKasFleetshardConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewServerConfig, di.As(new(provider.ConfigModule))),
		di.Provide(config.NewOCMConfig, di.As(new(provider.ConfigModule))),

		di.Provide(func(c *config.ApplicationConfig) *config.ObservabilityConfiguration {
			return c.ObservabilityConfiguration
		}),
		di.Provide(func(c *config.ApplicationConfig) *config.OSDClusterConfig { return c.OSDClusterConfig }),
		di.Provide(func(c *config.ApplicationConfig) *config.KafkaConfig { return c.Kafka }),
		di.Provide(func(c *config.ApplicationConfig) *config.KeycloakConfig { return c.Keycloak }),
		di.Provide(func(c *config.ApplicationConfig) *config.AccessControlListConfig { return c.AccessControlList }),
		di.Provide(func(c *config.ApplicationConfig) *config.AWSConfig { return c.AWS }),

		vault.ConfigProviders().AsOption(),
		sentry2.ConfigProviders().AsOption(),
	)...)
	if err != nil {
		return nil, err
	}

	err = env.ConfigContainer.Resolve(&env.Config)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// CreateServices loads the environment's resources
// This should be called after the environment has been configured appropriately though AddFlags and parsing,
// done elsewhere. The environment does NOT handle flag parsing
func (env *Env) CreateServices() error {

	glog.Infof("Initializing %s environment", env.Name)

	modules := []provider.ConfigModule{}
	if err := env.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return err
	}

	for i := range modules {
		err := modules[i].ReadFiles()
		if err != nil {
			err = errors.Errorf("unable to read configuration files: %s", err)
			glog.Error(err)
			sentry.CaptureException(err)
			return err
		}
	}

	var namedEnv EnvLoader
	err := env.ConfigContainer.Resolve(&namedEnv, di.Tags{"env": env.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", env.Name)
	}
	err = namedEnv.ModifyConfiguration(env)
	if err != nil {
		return err
	}

	type injections struct {
		di.Inject
		ServiceInjections         []provider.Provider
		BeforeCreateServicesHooks []provider.BeforeCreateServicesHook `optional:"true"`
		AfterCreateServicesHooks  []provider.AfterCreateServicesHook  `optional:"true"`
	}
	in := injections{}
	if err := env.ConfigContainer.Resolve(&in); err != nil {
		return err
	}

	for _, hook := range in.BeforeCreateServicesHooks {
		env.MustInvoke(hook.Func)
	}

	var opts []di.Option
	for i := range in.ServiceInjections {
		opt, err := in.ServiceInjections[i].Providers()
		if err != nil {
			return err
		}
		opts = append(opts, opt.AsOption())
	}

	env.ServiceContainer, err = di.New(append(opts,
		di.ProvideValue(env),
		di.ProvideValue(config.EnvName(env.Name)),

		// We wont need these providers that get values from the ConfigContainer
		// once we can add parent containers: https://github.com/goava/di/pull/34
		di.Provide(func() (value *config.ApplicationConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.KafkaConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.AWSConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.ObservabilityConfiguration, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.KeycloakConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.OCMConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.ServerConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.DatabaseConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.HealthCheckConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.MetricsConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.KasFleetshardConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.AccessControlListConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),

		di.Provide(db.NewConnectionFactory),

		di.Provide(observatorium.NewObservatoriumClient),
		di.Provide(ocm.NewOCMClient),
		di.Provide(func(client *ocm.Client) (ocmClient *sdkClient.Connection) {
			return client.Connection
		}),
		di.Provide(customOcm.NewClient),

		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(services.NewClusterService),

		di.Provide(func() services.KafkaKeycloakService {
			return services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
		}),

		di.Provide(func() services.OsdKeycloakService {
			return services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.OSDClusterIDPRealm)
		}),

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

		// Types registered with the StartupService are started when the app boots up
		di.Provide(server.NewAPIServer, di.As(new(provider.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(provider.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(provider.BootService))),
		di.Provide(func(dbFactory *db.ConnectionFactory) *signalbus.PgSignalBus {
			return signalbus.NewPgSignalBus(signalbus.NewSignalBus(), dbFactory)
		}, di.As(new(signalbus.SignalBus)), di.As(new(provider.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(provider.BootService))),
	)...)
	if err != nil {
		return err
	}

	var configService services.ConfigService
	env.MustResolve(&configService)
	if err := configService.Validate(); err != nil {
		return err
	}

	for _, hook := range in.AfterCreateServicesHooks {
		env.MustInvoke(hook.Func)
	}

	return nil
}

func (env *Env) MustInvoke(invocation di.Invocation, options ...di.InvokeOption) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	if err := container.Invoke(invocation, options...); err != nil {
		glog.Fatalf("%s di failure: %v", containerName, err)
	}
}

func (env *Env) MustResolve(ptr di.Pointer, options ...di.ResolveOption) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	if err := container.Resolve(ptr, options...); err != nil {
		glog.Fatalf("%s di failure: %v", containerName, err)
	}
}

func (env *Env) MustResolveAll(ptrs ...di.Pointer) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	for _, ptr := range ptrs {
		if err := container.Resolve(ptr); err != nil {
			glog.Fatalf("%s di failure: %v", containerName, err)
		}
	}
}

func (env *Env) Run(ctx context.Context) {
	env.Start()
	<-ctx.Done()
	env.Stop()
}

func (env *Env) Start() {
	env.MustInvoke(func(services []provider.BootService) {
		for i := range services {
			services[i].Start()
		}
	})
}

func (env *Env) Stop() {
	env.MustInvoke(func(services []provider.BootService) {
		for i := range services {
			i = len(services) - 1 - i // to stop in reverse order
			services[i].Stop()
		}
	})
}

func (env *Env) Cleanup() {
	//if env.Name != TestingEnv {
	//	var dbFactory *db.ConnectionFactory
	//	var ocmClient *ocm.Client
	//	env.MustResolveAll(&dbFactory, &ocmClient)
	//
	//	if err := dbFactory.Close(); err != nil {
	//		glog.Fatalf("Unable to close db connection: %s", err.Error())
	//	}
	//	ocmClient.Close()
	//}
	env.ServiceContainer.Cleanup()
	env.ConfigContainer.Cleanup()
}

func setConfigDefaults(flags *pflag.FlagSet, defaults map[string]string) error {
	for name, value := range defaults {
		err := flags.Set(name, value)
		if err != nil {
			glog.Errorf("Error setting flag %s: %v", name, err)
			return err
		}
	}
	return nil
}
