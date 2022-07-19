package kasfleetshardsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/golang-jwt/jwt/v4"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const AdminServerURI = "http://test-admin-server-uri.com"

var StandardCapacityInfo private.DataPlaneClusterUpdateStatusRequestCapacity = private.DataPlaneClusterUpdateStatusRequestCapacity{
	MaxUnits:       5,
	RemainingUnits: 3,
}

var DeveloperCapacityInfo private.DataPlaneClusterUpdateStatusRequestCapacity = private.DataPlaneClusterUpdateStatusRequestCapacity{
	MaxUnits:       50,
	RemainingUnits: 8,
}

// defaultUpdateDataplaneClusterStatusFunc - The default behaviour for updating data plane cluster status in each Kas Fleetshard Sync reconcile.
// Retrieves all clusters in the database in a 'waiting_for_kas_fleetshard_operator' state and updates it to 'ready' once all of the addons are installed.
var defaultUpdateDataplaneClusterStatusFunc mockKasFleetshardSyncUpdateDataPlaneClusterStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error {
	var clusterService services.ClusterService
	var ocmConfig *ocm.OCMConfig
	var kasFleetshardConfig *config.KasFleetshardConfig
	helper.Env.MustResolveAll(&clusterService, &ocmConfig, &kasFleetshardConfig)
	clusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterWaitingForKasFleetShardOperator,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterWaitingForKasFleetShardOperator, err)
	}

	for _, cluster := range clusters {
		managedKafkaAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.StrimziOperatorAddonID)
		if err != nil {
			return err
		}

		kasFleetShardOperatorAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.KasFleetshardAddonID)
		if err != nil {
			return err
		}

		if managedKafkaAddon.State() == clustersmgmtv1.AddOnInstallationStateReady && (kasFleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateReady || kasFleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateInstalling) {
			ctx, err := NewAuthenticatedContextForDataPlaneCluster(helper, cluster.ClusterID)
			if err != nil {
				return err
			}

			clusterStatusUpdateRequest := SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
			if _, err := privateClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, cluster.ClusterID, *clusterStatusUpdateRequest); err != nil {
				return fmt.Errorf("failed to update cluster status via agent endpoint: %v", err)
			}
		}
	}
	return nil
}

// defaultUpdateKafkaStatusFunc - The default behaviour for updating kafka status in each Kas Fleetshard Sync reconcile.
// This function retrieves and updates Kafkas in all ready and full data plane clusters.
// Any Kafkas marked for deletion are updated to 'deleting'
// Kafkas with any other status are updated to 'ready'
var defaultUpdateKafkaStatusFunc mockKasFleetshardSyncupdateKafkaClusterStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient) error {
	var clusterService services.ClusterService
	helper.Env.MustResolveAll(&clusterService)

	var dataplaneClusters []*api.Cluster
	readyDataplaneClusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterReady,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterReady, err)
	}

	dataplaneClusters = append(dataplaneClusters, readyDataplaneClusters...)

	fullDataplaneClusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterFull,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterFull, err)
	}

	dataplaneClusters = append(dataplaneClusters, fullDataplaneClusters...)

	for _, dataplaneCluster := range dataplaneClusters {
		ctx, err := NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)
		if err != nil {
			return err
		}

		kafkaList, _, err := privateClient.AgentClustersApi.GetKafkas(ctx, dataplaneCluster.ClusterID)
		if err != nil {
			return err
		}

		kafkaStatusList := make(map[string]private.DataPlaneKafkaStatus)
		for _, kafka := range kafkaList.Items {
			id := kafka.Metadata.Annotations.Bf2OrgId
			if kafka.Spec.Deleted {
				kafkaStatusList[id] = GetDeletedKafkaStatusResponse()
			} else {
				// Update any other clusters not in a 'deprovisioning' state to 'ready'
				kafkaStatusList[id] = GetReadyKafkaStatusResponse(dataplaneCluster.ClusterDNS)
			}
		}

		if _, err = privateClient.AgentClustersApi.UpdateKafkaClusterStatus(ctx, dataplaneCluster.ClusterID, kafkaStatusList); err != nil {
			return err
		}
	}
	return nil
}

type MockKasFleetshardSyncBuilder interface {
	// SetUpdateDataplaneClusterStatusFunc - Sets behaviour for updating dataplane clusters in each KAS Fleetshard sync reconcile
	SetUpdateDataplaneClusterStatusFunc(mockKasFleetshardSyncUpdateDataPlaneClusterStatusFunc)
	// SetUpdateKafkaStatusFunc - Sets behaviour for updating kafka clusters in each KAS Fleetshard sync reconcile
	SetUpdateKafkaStatusFunc(mockKasFleetshardSyncupdateKafkaClusterStatusFunc)
	// SetInterval - Sets the repeat interval for the mock KAS Fleetshard sync
	SetInterval(interval time.Duration)
	// Build - Builds a mock KAS Fleetshard sync
	Build() MockKasFleetshardSync
}

// Mock KAS Fleetshard Sync Builder
type mockKasFleetshardSyncBuilder struct {
	kfsync mockKasFleetshardSync
}

var _ MockKasFleetshardSyncBuilder = &mockKasFleetshardSyncBuilder{}

func NewMockKasFleetshardSyncBuilder(helper *coreTest.Helper, t *testing.T) MockKasFleetshardSyncBuilder {
	var ocmClient ocm.ClusterManagementClient
	helper.Env.MustResolveAll(&ocmClient)
	return &mockKasFleetshardSyncBuilder{
		kfsync: mockKasFleetshardSync{
			helper:                       helper,
			t:                            t,
			ocmClient:                    ocmClient,
			privateClient:                test.NewPrivateAPIClient(helper),
			updateDataplaneClusterStatus: defaultUpdateDataplaneClusterStatusFunc,
			updateKafkaClusterStatus:     defaultUpdateKafkaStatusFunc,
			interval:                     1 * time.Second,
		},
	}
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateDataplaneClusterStatusFunc(updateDataplaneClusterStatusFunc mockKasFleetshardSyncUpdateDataPlaneClusterStatusFunc) {
	m.kfsync.updateDataplaneClusterStatus = updateDataplaneClusterStatusFunc
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateKafkaStatusFunc(updateKafkaStatusFunc mockKasFleetshardSyncupdateKafkaClusterStatusFunc) {
	m.kfsync.updateKafkaClusterStatus = updateKafkaStatusFunc
}

func (m *mockKasFleetshardSyncBuilder) SetInterval(interval time.Duration) {
	m.kfsync.interval = interval
}

func (m *mockKasFleetshardSyncBuilder) Build() MockKasFleetshardSync {
	return &m.kfsync
}

type MockKasFleetshardSync interface {
	// Start - Starts the KAS Fleetshard sync reconciler
	Start()
	// Stop - Stops the KAS Fleetshard sync reconciler
	Stop()
}

type mockKasFleetshardSyncUpdateDataPlaneClusterStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error
type mockKasFleetshardSyncupdateKafkaClusterStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient) error

type mockKasFleetshardSync struct {
	helper                       *coreTest.Helper
	t                            *testing.T
	ocmClient                    ocm.Client
	ticker                       *time.Ticker
	privateClient                *private.APIClient
	interval                     time.Duration
	updateDataplaneClusterStatus mockKasFleetshardSyncUpdateDataPlaneClusterStatusFunc
	updateKafkaClusterStatus     mockKasFleetshardSyncupdateKafkaClusterStatusFunc
}

var _ MockKasFleetshardSync = &mockKasFleetshardSync{}

func (m *mockKasFleetshardSync) Start() {
	// starts reconcile immediately and then on every repeat interval
	// run reconcile
	m.t.Log("Starting mock fleetshard sync")
	m.ticker = time.NewTicker(m.interval)
	go func() {
		for range m.ticker.C {
			m.reconcileDataplaneClusters()
			m.reconcileKafkaClusters()
		}
	}()
}

func (m *mockKasFleetshardSync) Stop() {
	m.ticker.Stop()
}

func (m *mockKasFleetshardSync) reconcileDataplaneClusters() {
	if err := m.updateDataplaneClusterStatus(m.helper, m.privateClient, m.ocmClient); err != nil {
		m.t.Logf("Unable to update dataplane cluster status: %s", err)
	}
}

func (m *mockKasFleetshardSync) reconcileKafkaClusters() {
	if err := m.updateKafkaClusterStatus(m.helper, m.privateClient); err != nil {
		m.t.Logf("Failed to update Kafka cluster status: %s", err)
	}
}

// Returns an authenticated context to be used for calling the data plane endpoints
func NewAuthenticatedContextForDataPlaneCluster(h *coreTest.Helper, clusterID string) (context.Context, error) {
	var keycloakConfig *keycloak.KeycloakConfig
	var clusterService services.ClusterService
	h.Env.MustResolveAll(&keycloakConfig, &clusterService)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": keycloakConfig.SSOProviderRealm().ValidIssuerURI,
	}

	clientId, err := clusterService.GetClientId(clusterID)
	if err != nil {
		return nil, err
	}

	claims["clientId"] = clientId

	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx, err
}

// Returns a sample data plane cluster status request with available capacity
func SampleDataPlaneclusterStatusRequestWithAvailableCapacity() *private.DataPlaneClusterUpdateStatusRequest {
	return &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Capacity: map[string]private.DataPlaneClusterUpdateStatusRequestCapacity{
			types.STANDARD.String():  StandardCapacityInfo,
			types.DEVELOPER.String(): DeveloperCapacityInfo,
		},
		Strimzi: []private.DataPlaneClusterUpdateStatusRequestStrimzi{
			{
				Ready:   true,
				Version: "strimzi-cluster-operator.v0.23.0-0",
				KafkaVersions: []string{
					"2.7.0",
					"2.5.3",
					"2.6.2",
				},
				KafkaIbpVersions: []string{
					"2.7",
					"2.5",
					"2.6",
				},
			},
			{
				Ready:   true,
				Version: "strimzi-cluster-operator.v0.21.0-0",
				KafkaVersions: []string{
					"2.7.0",
					"2.3.1",
					"2.1.2",
				},
				KafkaIbpVersions: []string{
					"2.7",
					"2.1",
					"2.3",
				},
			},
		},
	}
}

// Return a Kafka status for a deleted cluster
func GetDeletedKafkaStatusResponse() private.DataPlaneKafkaStatus {
	return private.DataPlaneKafkaStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Deleted",
			},
		},
	}
}

func GetDefaultReportedKafkaVersion() string {
	return "2.7.0"
}

func GetDefaultReportedStrimziVersion() string {
	return "strimzi-cluster-operator.v0.23.0-0"
}

// Return a kafka status for a ready cluster
func GetReadyKafkaStatusResponse(clusterDNS string) private.DataPlaneKafkaStatus {
	return private.DataPlaneKafkaStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Versions: private.DataPlaneKafkaStatusVersions{
			Kafka:   GetDefaultReportedKafkaVersion(),
			Strimzi: GetDefaultReportedStrimziVersion(),
		},
		Routes: &[]private.DataPlaneKafkaStatusRoutes{
			{
				Name:   "test-route",
				Prefix: "",
				Router: clusterDNS,
			},
		},
		AdminServerURI: AdminServerURI,
	}
}

func GetErrorKafkaStatusResponse() private.DataPlaneKafkaStatus {
	return private.DataPlaneKafkaStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Error",
				Status: "False",
			},
		},
	}
}

func GetErrorWithCustomMessageKafkaStatusResponse(message string) private.DataPlaneKafkaStatus {
	res := GetErrorKafkaStatusResponse()
	res.Conditions[0].Message = message
	return res
}
