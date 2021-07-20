package test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	coreWorkers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/goava/di"
	"github.com/golang/glog"
)

type Services struct {
	di.Inject
	DBFactory             *db.ConnectionFactory
	KeycloakConfig        *keycloak.KeycloakConfig
	KafkaConfig           *config.KafkaConfig
	MetricsServer         *server.MetricsServer
	HealthCheckServer     *server.HealthCheckServer
	Workers               []coreWorkers.Worker
	LeaderElectionManager *coreWorkers.LeaderElectionManager
	SignalBus             signalbus.SignalBus
	APIServer             *server.ApiServer
	BootupServices        []environments.BootService
	CloudProvidersService services.CloudProvidersService
	ClusterService        services.ClusterService
	OCMClient             ocm.Client
	OCMConfig             *ocm.OCMConfig
	KafkaService          services.KafkaService
	ObservatoriumClient   *observatorium.Client
	ClusterManager        *workers.ClusterManager
	ServerConfig          *server.ServerConfig
}

var TestServices Services

// Register a test
// This should be run before every integration test
func NewKafkaHelper(t *testing.T, server *httptest.Server) (*test.Helper, *public.APIClient, func()) {
	return NewKafkaHelperWithHooks(t, server, nil)
}

func NewKafkaHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *public.APIClient, func()) {
	h, teardown := test.NewHelperWithHooks(t, server, configurationHook, kafka.ConfigProviders(), di.ProvideValue(environments.BeforeCreateServicesHook{
		Func: func(dataplaneClusterConfig *config.DataplaneClusterConfig, kafkaConfig *config.KafkaConfig, observabilityConfiguration *observatorium.ObservabilityConfiguration) {
			dataplaneClusterConfig.StrimziOperatorVersion = "strimzi-cluster-operator.v0.23.0-0"
			kafkaConfig.KafkaLifespan.EnableDeletionOfExpiredKafka = true
			observabilityConfiguration.EnableMock = true
			dataplaneClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
			dataplaneClusterConfig.RawKubernetesConfig = nil                      // disable applying resources for standalone clusters
		},
	}))
	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	return h, NewApiClient(h), teardown
}

func NewApiClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

func NewPrivateAPIClient(helper *test.Helper) *private.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := private.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := private.NewAPIClient(openapiConfig)
	return client
}

func NewAdminPrivateAPIClient(helper *test.Helper) *adminprivate.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := adminprivate.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := adminprivate.NewAPIClient(openapiConfig)
	return client
}
