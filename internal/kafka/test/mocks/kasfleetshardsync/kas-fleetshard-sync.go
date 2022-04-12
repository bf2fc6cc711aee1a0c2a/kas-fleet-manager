package kasfleetshardsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/golang-jwt/jwt/v4"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// defaultUpdateDataplaneClusterStatusFunc - The default behaviour for updating data plane cluster status in each Kas Fleetshard Sync reconcile.
// Retrieves all clusters in the database in a 'waiting_for_kas_fleetshard_operator' state and updates it to 'ready' once all of the addons are installed.
var defaultUpdateDataplaneClusterStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error {
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
			ctx := NewAuthenticatedContextForDataPlaneCluster(helper, cluster.ClusterID)
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
var defaultUpdateKafkaStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient) error {
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
		ctx := NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)

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
	SetUpdateDataplaneClusterStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error)
	// SetUpdateKafkaStatusFunc - Sets behaviour for updating kafka clusters in each KAS Fleetshard sync reconcile
	SetUpdateKafkaStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error)
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

func (m *mockKasFleetshardSyncBuilder) SetUpdateDataplaneClusterStatusFunc(updateDataplaneClusterStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error) {
	m.kfsync.updateDataplaneClusterStatus = updateDataplaneClusterStatusFunc
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateKafkaStatusFunc(updateKafkaStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient) error) {
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

type mockKasFleetshardSync struct {
	helper                       *coreTest.Helper
	t                            *testing.T
	ocmClient                    ocm.Client
	ticker                       *time.Ticker
	privateClient                *private.APIClient
	interval                     time.Duration
	updateDataplaneClusterStatus func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error
	updateKafkaClusterStatus     func(helper *coreTest.Helper, privateClient *private.APIClient) error
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
func NewAuthenticatedContextForDataPlaneCluster(h *coreTest.Helper, clusterID string) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": keycloakConfig.SSOProviderRealm().ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"clientId": fmt.Sprintf("kas-fleetshard-agent-%s", clusterID),
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx
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
		Total: private.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			Connections:                   &[]int32{1000000}[0],
			DataRetentionSize:             &[]string{"test"}[0],
			Partitions:                    &[]int32{1000000}[0],
		},
		NodeInfo: &private.DatePlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
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
		Remaining: private.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{1000000}[0], // TODO set the values taking the scale-up value if possible or a deterministic way to know we'll pass it
			Partitions:                    &[]int32{1000000}[0],
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			DataRetentionSize:             &[]string{"test"}[0],
		},
		ResizeInfo: &private.DatePlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &private.DatePlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{10000}[0],
				Partitions:                    &[]int32{10000}[0],
				IngressEgressThroughputPerSec: &[]string{"test"}[0],
				DataRetentionSize:             &[]string{"test"}[0],
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
