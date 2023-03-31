package test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/workers/cluster_mgrs"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
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
	OCMClient             ocm.ClusterManagementClient
	OCMConfig             *ocm.OCMConfig
	KafkaService          services.KafkaService
	ObservatoriumClient   *observatorium.Client
	ClusterManager        *cluster_mgrs.ClusterManager
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
		Func: func(dataplaneClusterConfig *config.DataplaneClusterConfig, kafkaConfig *config.KafkaConfig, observabilityConfiguration *observatorium.ObservabilityConfiguration,
			kasFleetshardConfig *config.KasFleetshardConfig, providerConfig *config.ProviderConfig,
			keycloakConfig *keycloak.KeycloakConfig) {
			observabilityConfiguration.EnableMock = true
			dataplaneClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
			dataplaneClusterConfig.RawKubernetesConfig = nil                      // disable applying resources for standalone clusters
			dataplaneClusterConfig.DynamicScalingConfig.EnableDynamicScaleUpManagerScaleUpTrigger = false
			dataplaneClusterConfig.DynamicScalingConfig.EnableDynamicScaleDownManagerScaleDownTrigger = false
			dataplaneClusterConfig.EnableKafkaSreIdentityProviderConfiguration = keycloakConfig.SelectSSOProvider == keycloak.MAS_SSO // only enable IDP configuration if provider type is mas_sso

			// disable the requirement to have a dataplane observability config file defined
			observabilityConfiguration.DataPlaneObservabilityConfig.Enabled = false

			// enable only aws for integration tests
			providerConfig.ProvidersConfig = config.ProviderConfiguration{
				SupportedProviders: config.ProviderList{
					{
						Name:    "aws",
						Default: true,
						Regions: []config.Region{
							{
								Name:    "us-east-1",
								Default: true,
								SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
									"standard":  {},
									"developer": {},
								},
							},
						},
					},
				},
			}
		},
	}))

	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}

	finalTeardownFun := func() {
		deleteLeftOverServiceAccounts(h)
		teardown()
	}

	return h, NewApiClient(h), finalTeardownFun
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

func NewMockDataplaneCluster(name string, capacity int) config.ManualCluster {
	return config.ManualCluster{
		Name:                  name,
		CloudProvider:         mocks.MockCluster.CloudProvider().ID(),
		Region:                mocks.MockCluster.Region().ID(),
		MultiAZ:               true,
		Schedulable:           true,
		KafkaInstanceLimit:    capacity,
		Status:                api.ClusterReady,
		SupportedInstanceType: "developer,standard",
	}
}

func deleteLeftOverServiceAccounts(h *test.Helper) {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolve(&keycloakConfig)
	defer h.Env.Cleanup()

	db := h.DBFactory().DB

	// delete cluster service accounts
	var clusters []*api.Cluster
	if err := db.Model(&api.Cluster{}).Scan(&clusters).Error; err != nil {
		glog.Fatalf("Unable to scan clusters: %s", err.Error())
	}

	for _, cluster := range clusters {
		deleteClustersServiceAccountsFromSSOService(cluster, keycloakConfig)
	}

	// delete kafka service accounts
	var kafkas []*dbapi.KafkaRequest
	if err := db.Model(&dbapi.KafkaRequest{}).Scan(&kafkas).Error; err != nil {
		glog.Fatalf("Unable to scan kafkas: %s", err.Error())
	}

	for _, kafka := range kafkas {
		deleteKafkasServiceAccountsFromSSOService(kafka, keycloakConfig)
	}
}

func deleteClustersServiceAccountsFromSSOService(cluster *api.Cluster, keycloakConfig *keycloak.KeycloakConfig) {
	if !shared.StringEmpty(cluster.ClientID) { // only delete the agent service account if it is set avoid unneeded calls to SSO if it is not set
		kafkaSsoService := sso.NewKeycloakServiceBuilder().
			ForKFM().
			WithConfiguration(keycloakConfig).
			Build()

		err := kafkaSsoService.DeleteServiceAccountInternal(cluster.ClientID)
		if err != nil {
			glog.Warningf("Failed to delete Fleetshard client with id %q from SSO for cluster %q due to %q. The sso provider is %q", cluster.ClientID, cluster.ClusterID, err.Error(), keycloakConfig.SelectSSOProvider)
		}
	}

	if keycloakConfig.SelectSSOProvider != keycloak.MAS_SSO {
		return
	}

	osdSSOService := sso.NewKeycloakServiceBuilder().
		ForOSD().
		WithConfiguration(keycloakConfig).
		WithRealmConfig(keycloakConfig.OSDClusterIDPRealm).
		Build()

	err := osdSSOService.DeRegisterClientInSSO(cluster.ID)

	if err != nil {
		glog.Warningf("Failed to delete IDP configuration client from SSO for cluster %q due to %q. The sso provider is %q", cluster.ClusterID, err.Error(), keycloakConfig.SelectSSOProvider)
	}
}

func deleteKafkasServiceAccountsFromSSOService(kafka *dbapi.KafkaRequest, keycloakConfig *keycloak.KeycloakConfig) {
	if shared.StringEmpty(kafka.CanaryServiceAccountClientID) { // only delete the canary service account if it is set avoid unneeded calls to SSO if it is not set
		return
	}

	kafkaSsoService := sso.NewKeycloakServiceBuilder().
		ForKFM().
		WithConfiguration(keycloakConfig).
		Build()

	err := kafkaSsoService.DeleteServiceAccountInternal(kafka.CanaryServiceAccountClientID)

	if err != nil {
		glog.Warningf("Failed to delete Kafka canary service account with id %q from SSO service for kafka %q due to %q. The sso provider is %q", kafka.CanaryServiceAccountClientID, kafka.Name, err.Error(), keycloakConfig.SelectSSOProvider)
	}
}
