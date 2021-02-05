package environments

import (
	"fmt"
	"os"
	"sync"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	customOcm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
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
	Name      string
	Config    *config.ApplicationConfig
	Services  Services
	Clients   Clients
	DBFactory *db.ConnectionFactory
}

type Services struct {
	Kafka          services.KafkaService
	Connectors     services.ConnectorsService
	ConnectorTypes services.ConnectorTypesService
	Cluster        services.ClusterService
	CloudProviders services.CloudProvidersService
	Config         services.ConfigService
	Observatorium  services.ObservatoriumService
	Keycloak       services.KeycloakService
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
		environment = &Env{}

		// Create the configuration
		environment.Config = config.NewApplicationConfig()
		environment.Name = GetEnvironmentStrFromEnv()
	})
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
	var defaults map[string]string

	switch e.Name {
	case DevelopmentEnv:
		defaults = developmentConfigDefaults
	case TestingEnv:
		defaults = testingConfigDefaults
	case ProductionEnv:
		defaults = productionConfigDefaults
	case StageEnv:
		defaults = stageConfigDefaults
	case IntegrationEnv:
		defaults = integrationConfigDefaults
	default:
		return fmt.Errorf("Unsupported environment %q", e.Name)
	}

	e.Config.AddFlags(flags)
	return setConfigDefaults(flags, defaults)
}

// Initialize loads the environment's resources
// This should be called after the e.Config has been set appropriately though AddFlags and pasing, done elsewhere
// The environment does NOT handle flag parsing
func (e *Env) Initialize() error {
	glog.Infof("Initializing %s environment", e.Name)

	err := environment.Config.ReadFiles()
	if err != nil {
		err = fmt.Errorf("Unable to read configuration files: %s", err)
		glog.Error(err)
		sentry.CaptureException(err)
		return err
	}

	switch e.Name {
	case DevelopmentEnv:
		err = loadDevelopment(environment)
	case TestingEnv:
		err = loadTesting(environment)
	case ProductionEnv:
		err = loadProduction(environment)
	case StageEnv:
		err = loadStage(environment)
	case IntegrationEnv:
		err = loadIntegration(environment)
	default:
		err = fmt.Errorf("Unsupported environment %q", e.Name)
	}
	return err
}

func (env *Env) LoadServices() error {
	ocmClient := customOcm.NewClient(env.Clients.OCM.Connection)
	clusterService := services.NewClusterService(env.DBFactory, ocmClient, env.Config.AWS, env.Config.ClusterCreationConfig)
	keycloakService := services.NewKeycloakService(env.Config.Keycloak)
	syncsetService := services.NewSyncsetService(ocmClient)
	kafkaService := services.NewKafkaService(env.DBFactory, syncsetService, clusterService, keycloakService, env.Config.Kafka, env.Config.AWS)
	cloudProviderService := services.NewCloudProvidersService(ocmClient)
	configService := services.NewConfigService(env.Config.SupportedProviders.ProvidersConfig, *env.Config.AllowList, *env.Config.ClusterCreationConfig, *env.Config.ObservabilityConfiguration)
	ObservatoriumService := services.NewObservatoriumService(env.Clients.Observatorium, kafkaService)

	env.Services.Kafka = kafkaService
	env.Services.Cluster = clusterService
	env.Services.CloudProviders = cloudProviderService
	env.Services.Observatorium = ObservatoriumService
	env.Services.Keycloak = keycloakService

	env.Services.Connectors = services.NewConnectorsService(env.DBFactory)
	env.Services.ConnectorTypes = services.NewConnectorTypesService(env.Config.ConnectorsConfig)
	if env.Config.ConnectorsConfig.Enabled {
		err := env.Services.ConnectorTypes.DiscoverExtensions()
		if err != nil {
			return err
		}
	}

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
		if err := flags.Set(name, value); err != nil {
			glog.Errorf("Error setting flag %s: %v", name, err)
			return err
		}
	}
	return nil
}
