package environments

import (
	"fmt"
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
	Container           *di.Container
}

type Services struct {
	Kafka                     services.KafkaService
	Connectors                services.ConnectorsService
	ConnectorTypes            services.ConnectorTypesService
	ConnectorCluster          services.ConnectorClusterService
	Cluster                   services.ClusterService
	CloudProviders            services.CloudProvidersService
	Config                    services.ConfigService
	Observatorium             services.ObservatoriumService
	Keycloak                  services.KeycloakService
	OsdIdpKeycloak            services.KeycloakService
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
		environment, err = NewEnv()
		if err != nil {
			panic(err)
		}
	})
}

func NewEnv(options ...di.Option) (*Env, error) {

	//di.SetTracer(di.StdTracer{})
	container, err := di.New(append(options,
		di.Provide(newDevelopmentEnvLoader, di.Tags{"env": DevelopmentEnv}),
		di.Provide(newProductionEnvLoader, di.Tags{"env": ProductionEnv}),
		di.Provide(newStageEnvLoader, di.Tags{"env": StageEnv}),
		di.Provide(newIntegrationEnvLoader, di.Tags{"env": IntegrationEnv}),
		di.Provide(newTestingEnvLoader, di.Tags{"env": TestingEnv}),
		di.Provide(config.NewApplicationConfig, di.As(new(config.ConfigModule))),
		di.Provide(config.NewConnectorsConfig, di.As(new(config.ConfigModule))),
	)...)
	if err != nil {
		return nil, err
	}

	env := &Env{
		Name:      GetEnvironmentStrFromEnv(),
		Container: container,
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

// Adds environment flags, using the environment's config struct, to the flagset 'flags'
func (e *Env) AddFlags(flags *pflag.FlagSet) error {

	var namedEnv EnvLoader
	err := e.Container.Resolve(&namedEnv, di.Tags{"env": e.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", e.Name)
	}

	modules := []config.ConfigModule{}
	err = e.Container.Resolve(&modules)
	if err != nil {
		return errors.Wrapf(err, "no config modules found")
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
	if err := e.Container.Resolve(&modules); err != nil {
		return errors.Wrapf(err, "no config modules found")
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
	err := e.Container.Resolve(&namedEnv, di.Tags{"env": e.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", e.Name)
	}

	return namedEnv.Load(environment)
}

func (env *Env) LoadServices() error {

	signalBus := signalbus.NewPgSignalBus(signalbus.NewSignalBus(), env.DBFactory)
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)
	clusterProviderFactory := clusters.NewDefaultProviderFactory(ocmClient, env.Config)
	clusterService := services.NewClusterService(env.DBFactory, clusterProviderFactory)
	kafkaKeycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.KafkaRealm)
	OsdIdpKeycloakService := services.NewKeycloakService(env.Config.Keycloak, env.Config.Keycloak.OSDClusterIDPRealm)
	configService := services.NewConfigService(*env.Config)
	quotaServiceFactory := quota.NewDefaultQuotaServiceFactory(ocmClient, env.DBFactory, configService)
	kafkaService := services.NewKafkaService(env.DBFactory, clusterService, kafkaKeycloakService, env.Config.Kafka, env.Config.AWS, quotaServiceFactory)
	cloudProviderService := services.NewCloudProvidersService(clusterProviderFactory)
	ObservatoriumService := services.NewObservatoriumService(env.Clients.Observatorium, kafkaService)
	kasFleetshardAddonService := services.NewKasFleetshardOperatorAddon(kafkaKeycloakService, configService, clusterProviderFactory)
	clusterPlmtStrategy := services.NewClusterPlacementStrategy(configService, clusterService)

	env.QuotaServiceFactory = quotaServiceFactory

	env.Services.Kafka = kafkaService
	env.Services.Cluster = clusterService
	env.Services.CloudProviders = cloudProviderService
	env.Services.Observatorium = ObservatoriumService
	env.Services.Keycloak = kafkaKeycloakService
	env.Services.OsdIdpKeycloak = OsdIdpKeycloakService
	env.Services.KasFleetshardAddonService = kasFleetshardAddonService
	env.Services.SignalBus = signalBus
	env.Services.ClusterPlmtStrategy = clusterPlmtStrategy

	vaultService, err := services.NewVaultService(env.Config.Vault)
	if err != nil {
		return err
	}
	env.Services.Vault = vaultService

	dataPlaneClusterService := services.NewDataPlaneClusterService(clusterService, env.Config)
	dataPlaneKafkaService := services.NewDataPlaneKafkaService(kafkaService, clusterService)
	env.Services.DataPlaneCluster = dataPlaneClusterService
	env.Services.DataPlaneKafkaService = dataPlaneKafkaService

	connectorsConfig := &config.ConnectorsConfig{}
	if err := env.Container.Resolve(&connectorsConfig); err != nil {
		return err
	}

	env.Services.Connectors = services.NewConnectorsService(env.DBFactory, signalBus, vaultService, env.Services.ConnectorTypes)
	env.Services.ConnectorTypes = services.NewConnectorTypesService(connectorsConfig, env.DBFactory)
	env.Services.ConnectorCluster = services.NewConnectorClusterService(env.DBFactory, signalBus, vaultService, env.Services.ConnectorTypes)

	// load the new config service and ensure it's valid (pre-req checks are performed)
	env.Services.Config = configService
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
