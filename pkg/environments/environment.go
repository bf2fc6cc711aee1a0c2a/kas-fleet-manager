package environments

import (
	"context"
	goerrors "errors"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sentry"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/goava/di"
	"github.com/pkg/errors"

	sentryGo "github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
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
		ConfigProviders(),
		vault.ConfigProviders(),
		sentry.ConfigProviders(),
		signalbus.ConfigProviders(),
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

	// Read in all config files
	modules := []provider.ConfigModule{}
	if err := env.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return err
	}
	for i := range modules {
		err := modules[i].ReadFiles()
		if err != nil {
			err = errors.Errorf("unable to read configuration files: %s", err)
			glog.Error(err)
			sentryGo.CaptureException(err)
			return err
		}
	}

	// Allow named env to customize the configuration.
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

	// Right before we create the services, run the any before create service hooks
	// to allow for post config file loading customization.
	for _, hook := range in.BeforeCreateServicesHooks {
		env.MustInvoke(hook.Func)
	}

	// Lets build a list of providers that will be used to create the service container.
	serviceProviders := []di.Option{di.ProvideValue(env)}
	for _, s := range in.ServiceInjections {
		serviceProviders = append(serviceProviders, s.Providers())
	}
	env.ServiceContainer, err = di.New(serviceProviders...)
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
	if env.ServiceContainer != nil {
		env.ServiceContainer.Cleanup()
	}
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
