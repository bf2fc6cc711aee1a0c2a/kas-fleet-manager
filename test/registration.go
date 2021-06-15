package test

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
	"github.com/golang/glog"
	gm "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

type Hook func(helper *Helper)

// Register a test
// This should be run before every integration test
func RegisterIntegration(t *testing.T, server *httptest.Server, options ...di.Option) (*Helper, *openapi.APIClient, func()) {
	return RegisterIntegrationWithHooks(t, server, nil, options...)

}

// RegisterIntegrationWithHooks will init the Helper and start the server, and it allows to customize the configurations of the server via the hooks.
// The startHook will be invoked after the Helper object is inited but before the api server is started, which will allow caller to change configurations via the helper object.
// The teardownHook will be called before server is stopped, to allow the caller to reset configurations via the helper object.
func RegisterIntegrationWithHooks(t *testing.T, server *httptest.Server, configurationHook Hook, options ...di.Option) (*Helper, *openapi.APIClient, func()) {

	// Register the test with gomega
	gm.RegisterTestingT(t)

	options = append(options, kafka.ConfigProviders().AsOption())

	// Manually set environment name, ignoring environment variables
	validTestEnv := false
	envName := environments.GetEnvironmentStrFromEnv()
	for _, testEnv := range []string{environments.TestingEnv, environments.IntegrationEnv, environments.DevelopmentEnv} {
		if envName == testEnv {
			validTestEnv = true
			break
		}
	}
	if !validTestEnv {
		fmt.Println("OCM_ENV environment variable not set to a valid test environment, using default testing environment")
		envName = environments.TestingEnv
	}

	var err error
	env, err := environments.NewEnv(envName, options...)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}

	commandLine := pflag.NewFlagSet("test", pflag.PanicOnError)
	err = env.AddFlags(commandLine)
	if err != nil {
		glog.Fatalf("Unable to add environment flags: %s", err.Error())
	}
	if logLevel := os.Getenv("LOGLEVEL"); logLevel != "" {
		glog.Infof("Using custom loglevel: %s", logLevel)
		err = commandLine.Set("v", logLevel)
		if err != nil {
			glog.Warningf("Unable to set custom logLevel: %s", err.Error())
		}
	}
	err = commandLine.Parse(os.Args[1:])
	if err != nil {
		glog.Fatalf("Unable to parse command line options: %s", err.Error())
	}

	env.Config.OSDClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
	env.Config.Kafka.KafkaLifespan.EnableDeletionOfExpiredKafka = true
	db.KafkaAdditionalLeasesExpireTime = time.Now().Add(-time.Minute) // set kafkas lease as expired so that a new leader is elected for each of the leases

	// Create a new helper
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, env.Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Fatalf("failed to create a new auth helper %s", err.Error())
	}
	h := &Helper{
		T:             t,
		Env:           env,
		JWTPrivateKey: authHelper.JWTPrivateKey,
		JWTCA:         authHelper.JWTCA,
		AuthHelper:    authHelper,
	}

	// Set server if provided
	env.Config.ObservabilityConfiguration.EnableMock = true
	if server != nil {
		fmt.Printf("Setting OCM base URL to %s\n", server.URL)
		env.Config.OCM.BaseURL = server.URL
		if env.Config.OCM.MockMode == config.MockModeEmulateServer {
			workers.RepeatInterval = 1 * time.Second
		}
	}

	jwkURL, stopJWKMockServer := h.StartJWKCertServerMock()
	env.Config.Server.JwksURL = jwkURL
	env.Config.Keycloak.EnableAuthenticationOnKafka = false

	// the configuration hook might set config options that influence which config files are loaded,
	// by env.LoadConfig()
	if configurationHook != nil {
		configurationHook(h)
	}

	// loads the config files.
	err = env.LoadConfig()
	if err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}

	// the configuration hook might set config options that are changing settings that where just
	// loaded from the config files that were just loaded, so run it again.
	if configurationHook != nil {
		configurationHook(h)
	}

	err = env.CreateServices()
	if err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}

	if err = env.ServiceContainer.Resolve(&h.Services); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}

	// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
	h.startMetricsServer()
	h.startHealthCheckServer()

	// Reset the database to a seeded blank state
	h.ResetDB()
	h.ResetMetrics()
	h.StartServer()
	h.StartLeaderElectionWorker()
	h.StartSignalBusWorker()
	// Create an api client
	client := h.NewApiClient()

	return h, client, buildTeardownHelperFn(
		func() {
			//if teardownHook != nil {
			//	teardownHook(h)
			//}
		},
		h.StopSignalBusWorker,
		h.StopLeaderElectionWorker,
		h.StopServer,
		h.CleanDB,
		stopJWKMockServer,
		h.stopHealthCheckServer,
		h.stopHealthCheckServer,
		h.stopMetricsServer,
		env.Teardown,
	)
}

func buildTeardownHelperFn(funcs ...func()) func() {
	return func() {
		for _, f := range funcs {
			if f != nil {
				f()
			}
		}
	}
}
