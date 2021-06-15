package environments

import (
	goerrors "errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
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
	dataEndpoint         string = "/api/metrics/v1/"
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

	modules := []config.ConfigModule{}
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

		// Add the env types
		di.Provide(newDevelopmentEnvLoader, di.Tags{"env": DevelopmentEnv}),
		di.Provide(newProductionEnvLoader, di.Tags{"env": ProductionEnv}),
		di.Provide(newStageEnvLoader, di.Tags{"env": StageEnv}),
		di.Provide(newIntegrationEnvLoader, di.Tags{"env": IntegrationEnv}),
		di.Provide(newTestingEnvLoader, di.Tags{"env": TestingEnv}),

		di.Provide(config.NewApplicationConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewMetricsConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewHealthCheckConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewDatabaseConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewSentryConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewKasFleetshardConfig, di.As(new(config.ConfigModule))),

		di.Provide(func(c *config.ApplicationConfig) *config.ObservabilityConfiguration {
			return c.ObservabilityConfiguration
		}),

		vault.ConfigProviders().AsOption(),
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

func (env *Env) LoadConfigAndCreateServices() error {
	err := env.LoadConfig()
	if err != nil {
		return err
	}
	err = env.CreateServices()
	if err != nil {
		return err
	}
	return nil
}

// LoadConfig loads the environment's resources
// This should be called after the e.Config has been set appropriately though AddFlags and pasing, done elsewhere
// The environment does NOT handle flag parsing
func (env *Env) LoadConfig() error {
	glog.Infof("Initializing %s environment", env.Name)

	modules := []config.ConfigModule{}
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
	return namedEnv.Load(env)
}

func (env *Env) CreateServices() error {

	var serviceInjections []provider.Provider
	if err := env.ConfigContainer.Resolve(&serviceInjections); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return err
	}

	var opts []di.Option
	for i := range serviceInjections {
		opt, err := serviceInjections[i].Providers()
		if err != nil {
			return err
		}
		opts = append(opts, opt.AsOption())
	}

	container, err := di.New(append(opts,
		di.ProvideValue(env),
		di.ProvideValue(env.Config),
		di.ProvideValue(env.Config.Kafka),
		di.ProvideValue(env.Config.AWS),
		di.ProvideValue(env.Config.Server),
		di.ProvideValue(env.Config.OCM),
		di.ProvideValue(env.Config.ObservabilityConfiguration),
		di.ProvideValue(env.Config.Keycloak),

		// We wont need these providers that get values from the ConfigContainer
		// once we can add parent containers: https://github.com/goava/di/pull/34
		di.Provide(func() (value *config.DatabaseConfig, err error) {
			err = env.ConfigContainer.Resolve(&value)
			return
		}),
		di.Provide(func() (value *config.SentryConfig, err error) {
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

		di.Provide(db.NewConnectionFactory),
		di.Provide(func(dbFactory *db.ConnectionFactory) *signalbus.PgSignalBus {
			return signalbus.NewPgSignalBus(signalbus.NewSignalBus(), dbFactory)
		}, di.As(new(signalbus.SignalBus))),

		di.Provide(NewObservatoriumClient),
		di.Provide(NewOCMClient),
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

		di.Provide(server.NewAPIServer),
		di.Provide(server.NewMetricsServer),
		di.Provide(server.NewHealthCheckServer),

		di.Provide(workers.NewLeaderElectionManager),
	)...)
	if err != nil {
		return err
	}

	env.ServiceContainer = container

	var configService services.ConfigService
	env.MustResolve(&configService)
	if err := configService.Validate(); err != nil {
		return err
	}

	return env.ServiceContainer.Invoke(InitializeSentry)
}

func (env *Env) MustInvoke(invocation di.Invocation, options ...di.InvokeOption) {
	container := env.ServiceContainer
	if container == nil {
		container = env.ConfigContainer
	}
	if err := container.Invoke(invocation, options...); err != nil {
		glog.Fatalf("di failure: %v", err)
	}
}

func (env *Env) MustResolve(ptr di.Pointer, options ...di.ResolveOption) {
	container := env.ServiceContainer
	if container == nil {
		container = env.ConfigContainer
	}
	if err := container.Resolve(ptr, options...); err != nil {
		glog.Fatalf("di failure: %v", err)
	}
}

func (env *Env) MustResolveAll(ptrs ...di.Pointer) {
	container := env.ServiceContainer
	if container == nil {
		container = env.ConfigContainer
	}
	for _, ptr := range ptrs {
		if err := container.Resolve(ptr); err != nil {
			glog.Fatalf("di failure: %v", err)
		}
	}
}

func NewOCMClient(OCM *config.OCMConfig) (client *ocm.Client, err error) {
	ocmConfig := ocm.Config{
		BaseURL:      OCM.BaseURL,
		ClientID:     OCM.ClientID,
		ClientSecret: OCM.ClientSecret,
		SelfToken:    OCM.SelfToken,
		TokenURL:     OCM.TokenURL,
		Debug:        OCM.Debug,
	}

	// Create OCM Authz client
	if OCM.EnableMock {
		if OCM.MockMode == config.MockModeEmulateServer {
			client, err = ocm.NewIntegrationClientMock(ocmConfig)
		} else {
			glog.Infof("Using Mock OCM Authz Client")
			client, err = ocm.NewClientMock(ocmConfig)
		}
	} else {
		client, err = ocm.NewClient(ocmConfig)
	}
	if err != nil {
		glog.Errorf("Unable to create OCM Authz client: %s", err.Error())
	}
	return
}

func NewObservatoriumClient(c *config.ObservabilityConfiguration) (client *observatorium.Client, err error) {
	// Create Observatorium client
	observatoriumConfig := &observatorium.Configuration{
		BaseURL:   c.ObservatoriumGateway + dataEndpoint + c.ObservatoriumTenant,
		AuthToken: c.AuthToken,
		Cookie:    c.Cookie,
		Timeout:   c.Timeout,
		Debug:     c.Debug,
		Insecure:  c.Insecure,
	}
	if c.EnableMock {
		glog.Infof("Using Mock Observatorium Client")
		client, err = observatorium.NewClientMock(observatoriumConfig)
	} else {
		client, err = observatorium.NewClient(observatoriumConfig)
	}
	if err != nil {
		glog.Errorf("Unable to create Observatorium client: %s", err)
	}
	return
}

func InitializeSentry(env *Env, c *config.SentryConfig) error {
	options := sentry.ClientOptions{}

	if c.Enabled {
		key := c.Key
		url := c.URL
		project := c.Project
		glog.Infof("Sentry error reporting enabled to %s on project %s", url, project)
		options.Dsn = fmt.Sprintf("https://%s@%s/%s", key, url, project)
	} else {
		// Setting the DSN to an empty string effectively disables sentry
		// See https://godoc.org/github.com/getsentry/sentry-go#ClientOptions Dsn
		glog.Infof("Disabling Sentry error reporting")
		options.Dsn = ""
	}

	options.Transport = &sentry.HTTPTransport{
		Timeout: c.Timeout,
	}
	options.Debug = c.Debug
	options.AttachStacktrace = true
	options.Environment = env.Name

	hostname, err := os.Hostname()
	if err != nil && hostname != "" {
		options.ServerName = hostname
	}
	// TODO figure out some way to set options.Release and options.Dist

	err = sentry.Init(options)
	if err != nil {
		glog.Errorf("Unable to initialize sentry integration: %s", err.Error())
		return err
	}
	return nil
}

func (env *Env) Teardown() {
	if env.Name != TestingEnv {
		var dbFactory *db.ConnectionFactory
		var ocmClient *ocm.Client
		env.MustResolveAll(&dbFactory, &ocmClient)

		if err := dbFactory.Close(); err != nil {
			glog.Fatalf("Unable to close db connection: %s", err.Error())
		}
		ocmClient.Close()
	}
	env.ServiceContainer.Cleanup()
	env.ServiceContainer = nil
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
