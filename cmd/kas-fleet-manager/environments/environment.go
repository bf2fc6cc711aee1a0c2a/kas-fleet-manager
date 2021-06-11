package environments

import (
	goerrors "errors"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"os"
	"sync"

	"github.com/goava/di"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"

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
	Name                string
	Config              *config.ApplicationConfig
	Services            Services
	Clients             Clients
	DBFactory           *db.ConnectionFactory
	QuotaServiceFactory services.QuotaServiceFactory
	ConfigContainer     *di.Container
	ServiceContainer    *di.Container
}

type Services struct {
	Kafka services.KafkaService
	//Connectors                services.ConnectorsService
	//ConnectorTypes            services.ConnectorTypesService
	//ConnectorCluster          services.ConnectorClusterService
	Cluster                   services.ClusterService
	CloudProviders            services.CloudProvidersService
	Config                    services.ConfigService
	Observatorium             services.ObservatoriumService
	Keycloak                  services.KafkaKeycloakService
	OsdIdpKeycloak            services.OsdKeycloakService
	DataPlaneCluster          services.DataPlaneClusterService
	DataPlaneKafkaService     services.DataPlaneKafkaService
	KasFleetshardAddonService services.KasFleetshardOperatorAddon
	SignalBus                 signalbus.SignalBus
	Vault                     services.VaultService
	ClusterPlmtStrategy       services.ClusterPlacementStrategy
}

type Clients struct {
	OCM           *ocm.Client
	Observatorium *observatorium.Client
}

type ConfigDefaults struct {
	Server   map[string]interface{}
	Metrics  map[string]interface{}
	Database map[string]interface{}
	OCM      map[string]interface{}
	Options  map[string]interface{}
}

var environment *Env
var once sync.Once

func init() {
	once.Do(func() {
		var err error
		environment, err = NewEnv(GetEnvironmentStrFromEnv())
		if err != nil {
			panic(err)
		}
	})
}

func NewEnv(name string, options ...di.Option) (*Env, error) {

	//di.SetTracer(di.StdTracer{})
	container, err := di.New(append(options,
		// Add the env types
		di.Provide(newDevelopmentEnvLoader, di.Tags{"env": DevelopmentEnv}),
		di.Provide(newProductionEnvLoader, di.Tags{"env": ProductionEnv}),
		di.Provide(newStageEnvLoader, di.Tags{"env": StageEnv}),
		di.Provide(newIntegrationEnvLoader, di.Tags{"env": IntegrationEnv}),
		di.Provide(newTestingEnvLoader, di.Tags{"env": TestingEnv}),

		// Add the config modules
		di.Provide(config.NewApplicationConfig, di.As(new(config.ConfigModule))),

		// Add the connector injections.
		connector.EnvInjections().AsOption(),
	)...)
	if err != nil {
		return nil, err
	}

	env := &Env{
		Name:            name,
		ConfigContainer: container,
	}
	err = container.Resolve(&env.Config)
	if err != nil {
		return nil, err
	}
	return env, nil

}

func GetEnvironmentStrFromEnv() string {
	envStr, specified := os.LookupEnv(EnvironmentStringKey)
	if !specified || envStr == "" {
		glog.Infof("Environment variable %q not specified, using default %q", EnvironmentStringKey, EnvironmentDefault)
		envStr = EnvironmentDefault
	}
	return envStr
}

func Environment() *Env {
	return environment
}

func SetEnvironment(e *Env) {
	environment = e
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

// Initialize loads the environment's resources
// This should be called after the e.Config has been set appropriately though AddFlags and pasing, done elsewhere
// The environment does NOT handle flag parsing
func (e *Env) Initialize() error {
	glog.Infof("Initializing %s environment", e.Name)

	modules := []config.ConfigModule{}
	if err := e.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
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
	err := e.ConfigContainer.Resolve(&namedEnv, di.Tags{"env": e.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", e.Name)
	}

	return namedEnv.Load(environment)
}

func (env *Env) LoadServices() error {

	var serviceInjections []config.ServiceInjector
	if err := env.ConfigContainer.Resolve(&serviceInjections); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return err
	}

	var opts []di.Option
	for i := range serviceInjections {
		opt, err := serviceInjections[i].Injections()
		if err != nil {
			return err
		}
		opts = append(opts, opt.AsOption())
	}

	// We need to build a new container here because LoadServices() can be called after a config
	// change, and we need to re-create all the services.
	container, err := di.New(append(opts,
		di.ProvideValue(env.Config),
		di.ProvideValue(env.DBFactory),

		di.Provide(func() signalbus.SignalBus {
			return signalbus.NewPgSignalBus(signalbus.NewSignalBus(), env.DBFactory)
		}),
		di.Provide(func() customOcm.Client {
			return customOcm.NewClient(env.Clients.OCM.Connection)
		}),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(services.NewClusterService),

		di.Provide(func() services.KeycloakService {
			return services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
		}, di.Tags{"realm": "kafka"}, di.As(new(services.KafkaKeycloakService))),

		di.Provide(func() services.KeycloakService {
			return services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.OSDClusterIDPRealm)
		}, di.Tags{"realm": "osd"}, di.As(new(services.OsdKeycloakService))),

		di.ProvideValue(*env.Config),
		di.Provide(services.NewConfigService),
		di.Provide(quota.NewDefaultQuotaServiceFactory),

		di.ProvideValue(env.Config.Kafka),
		di.ProvideValue(env.Config.AWS),

		di.Provide(services.NewKafkaService, di.As(new(services.KafkaService))),

		di.Provide(services.NewCloudProvidersService),
		di.ProvideValue(env.Clients.Observatorium),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewKasFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),

		di.ProvideValue(env.Config.Vault),
		di.Provide(services.NewVaultService),

		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneKafkaService, di.As(new(services.DataPlaneKafkaService))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
	)...)
	if err != nil {
		return err
	}

	if env.ServiceContainer != nil {
		env.ServiceContainer.Cleanup()
	}
	env.ServiceContainer = container

	if err := container.Resolve(&env.QuotaServiceFactory); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.Kafka); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.Cluster); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.CloudProviders); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.Observatorium); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.Keycloak, di.Tags{"realm": "kafka"}); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.OsdIdpKeycloak, di.Tags{"realm": "osd"}); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.KasFleetshardAddonService); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.SignalBus); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.ClusterPlmtStrategy); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.Vault); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.DataPlaneCluster); err != nil {
		return err
	}
	if err := container.Resolve(&env.Services.DataPlaneKafkaService); err != nil {
		return err
	}

	//if err := container.Resolve(&env.Services.Connectors); err != nil {
	//	return err
	//}
	//if err := container.Resolve(&env.Services.ConnectorTypes); err != nil {
	//	return err
	//}
	//if err := container.Resolve(&env.Services.ConnectorCluster); err != nil {
	//	return err
	//}
	if err := container.Resolve(&env.Services.Config); err != nil {
		return err
	}

	if err := env.Services.Config.Validate(); err != nil {
		return err
	}
	return nil
}

func (env *Env) LoadClients() error {
	var err error

	ocmConfig := ocm.Config{
		BaseURL:      env.Config.OCM.BaseURL,
		ClientID:     env.Config.OCM.ClientID,
		ClientSecret: env.Config.OCM.ClientSecret,
		SelfToken:    env.Config.OCM.SelfToken,
		TokenURL:     env.Config.OCM.TokenURL,
		Debug:        env.Config.OCM.Debug,
	}

	// Create OCM Authz client
	if env.Config.OCM.EnableMock {
		if env.Config.OCM.MockMode == config.MockModeEmulateServer {
			env.Clients.OCM, err = ocm.NewIntegrationClientMock(ocmConfig)
		} else {
			glog.Infof("Using Mock OCM Authz Client")
			env.Clients.OCM, err = ocm.NewClientMock(ocmConfig)
		}
	} else {
		env.Clients.OCM, err = ocm.NewClient(ocmConfig)
	}
	if err != nil {
		glog.Errorf("Unable to create OCM Authz client: %s", err.Error())
		return err
	}

	// Create Observatorium client
	observatoriumConfig := &observatorium.Configuration{
		BaseURL:   env.Config.ObservabilityConfiguration.ObservatoriumGateway + dataEndpoint + env.Config.ObservabilityConfiguration.ObservatoriumTenant,
		AuthToken: env.Config.ObservabilityConfiguration.AuthToken,
		Cookie:    env.Config.ObservabilityConfiguration.Cookie,
		Timeout:   env.Config.ObservabilityConfiguration.Timeout,
		Debug:     env.Config.ObservabilityConfiguration.Debug,
		Insecure:  env.Config.ObservabilityConfiguration.Insecure,
	}
	if env.Config.ObservabilityConfiguration.EnableMock {
		glog.Infof("Using Mock Observatorium Client")
		env.Clients.Observatorium, err = observatorium.NewClientMock(observatoriumConfig)
	} else {
		env.Clients.Observatorium, err = observatorium.NewClient(observatoriumConfig)
	}
	if err != nil {
		glog.Errorf("Unable to create Observatorium client: %s", err)
		return err
	}

	return nil
}

func (env *Env) InitializeSentry() error {
	options := sentry.ClientOptions{}

	if env.Config.Sentry.Enabled {
		key := env.Config.Sentry.Key
		url := env.Config.Sentry.URL
		project := env.Config.Sentry.Project
		glog.Infof("Sentry error reporting enabled to %s on project %s", url, project)
		options.Dsn = fmt.Sprintf("https://%s@%s/%s", key, url, project)
	} else {
		// Setting the DSN to an empty string effectively disables sentry
		// See https://godoc.org/github.com/getsentry/sentry-go#ClientOptions Dsn
		glog.Infof("Disabling Sentry error reporting")
		options.Dsn = ""
	}

	options.Transport = &sentry.HTTPTransport{
		Timeout: env.Config.Sentry.Timeout,
	}
	options.Debug = env.Config.Sentry.Debug
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
		if err := env.DBFactory.Close(); err != nil {
			glog.Fatalf("Unable to close db connection: %s", err.Error())
		}
		env.Clients.OCM.Close()
	}
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
