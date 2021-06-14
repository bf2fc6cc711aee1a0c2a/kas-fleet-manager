package kasfleetshardsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/dgrijalva/jwt-go"

	privateopenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// defaultUpdateDataplaneClusterStatusFunc - The default behaviour for updating data plane cluster status in each Kas Fleetshard Sync reconcile.
// Retrieves all clusters in the database in a 'waiting_for_kas_fleetshard_operator' state and updates it to 'ready' once all of the addons are installed.
var defaultUpdateDataplaneClusterStatusFunc = func(helper *test.Helper, privateClient *privateopenapi.APIClient, ocmClient ocm.Client) error {
	clusters, err := helper.Env().Services.Cluster.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterWaitingForKasFleetShardOperator,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterWaitingForKasFleetShardOperator, err)
	}

	for _, cluster := range clusters {
		managedKafkaAddon, err := ocmClient.GetAddon(cluster.ClusterID, helper.Env().Config.OCM.StrimziOperatorAddonID)
		if err != nil {
			return err
		}

		kasFleetShardOperatorAddon, err := ocmClient.GetAddon(cluster.ClusterID, helper.Env().Config.OCM.KasFleetshardAddonID)
		if err != nil {
			return err
		}

		// The KAS Fleetshard Operator and Sync containers are restarted every ~5mins due to the informer state being out of sync.
		// Because of this, the addonInstallation state in ocm may never get updated to a 'ready' state as the install plan
		// status gets updated everytime the container gets restarted even though the operator itself has been successfully installed.
		// This should get better in the next release of the fleetshard operator. Until then, the cluster can be deemed ready if
		// it's in an 'installing' state, therwise this poll may timeout.
		if managedKafkaAddon.State() == clustersmgmtv1.AddOnInstallationStateReady &&
			(kasFleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateReady ||
				kasFleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateInstalling) {
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
var defaultUpdateKafkaStatusFunc = func(helper *test.Helper, privateClient *privateopenapi.APIClient) error {
	clusterService := helper.Env().Services.Cluster

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

		kafkaStatusList := make(map[string]privateopenapi.DataPlaneKafkaStatus)
		for _, kafka := range kafkaList.Items {
			id := kafka.Metadata.Annotations.Id
			if kafka.Spec.Deleted {
				kafkaStatusList[id] = GetDeletedKafkaStatusResponse()
			} else {
				// Update any other clusters not in a 'deprovisioning' state to 'ready'
				kafkaStatusList[id] = GetReadyKafkaStatusResponse()
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
	SetUpdateDataplaneClusterStatusFunc(func(helper *test.Helper, privateClient *privateopenapi.APIClient, ocmClient ocm.Client) error)
	// SetUpdateKafkaStatusFunc - Sets behaviour for updating kafka clusters in each KAS Fleetshard sync reconcile
	SetUpdateKafkaStatusFunc(func(helper *test.Helper, privateClient *privateopenapi.APIClient) error)
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

func NewMockKasFleetshardSyncBuilder(helper *test.Helper, t *testing.T) MockKasFleetshardSyncBuilder {
	return &mockKasFleetshardSyncBuilder{
		kfsync: mockKasFleetshardSync{
			helper:                       helper,
			t:                            t,
			ocmClient:                    ocm.NewClient(helper.Env().Clients.OCM.Connection),
			privateClient:                helper.NewPrivateAPIClient(),
			updateDataplaneClusterStatus: defaultUpdateDataplaneClusterStatusFunc,
			updateKafkaClusterStatus:     defaultUpdateKafkaStatusFunc,
			interval:                     10 * time.Second,
		},
	}
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateDataplaneClusterStatusFunc(updateDataplaneClusterStatusFunc func(helper *test.Helper, privateClient *privateopenapi.APIClient, ocmClient ocm.Client) error) {
	m.kfsync.updateDataplaneClusterStatus = updateDataplaneClusterStatusFunc
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateKafkaStatusFunc(updateKafkaStatusFunc func(helper *test.Helper, privateClient *privateopenapi.APIClient) error) {
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
	helper                       *test.Helper
	t                            *testing.T
	ocmClient                    ocm.Client
	ticker                       *time.Ticker
	privateClient                *privateopenapi.APIClient
	interval                     time.Duration
	updateDataplaneClusterStatus func(helper *test.Helper, privateClient *privateopenapi.APIClient, ocmClient ocm.Client) error
	updateKafkaClusterStatus     func(helper *test.Helper, privateClient *privateopenapi.APIClient) error
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
func NewAuthenticatedContextForDataPlaneCluster(h *test.Helper, clusterID string) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": h.AppConfig.Keycloak.KafkaRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), privateopenapi.ContextAccessToken, token)

	return ctx
}

// Returns a sample data plane cluster status request with available capacity
func SampleDataPlaneclusterStatusRequestWithAvailableCapacity() *privateopenapi.DataPlaneClusterUpdateStatusRequest {
	return &privateopenapi.DataPlaneClusterUpdateStatusRequest{
		Conditions: []privateopenapi.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: privateopenapi.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			Connections:                   &[]int32{1000000}[0],
			DataRetentionSize:             &[]string{"test"}[0],
			Partitions:                    &[]int32{1000000}[0],
		},
		NodeInfo: &privateopenapi.DataPlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
		},
		Remaining: privateopenapi.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{1000000}[0], // TODO set the values taking the scale-up value if possible or a deterministic way to know we'll pass it
			Partitions:                    &[]int32{1000000}[0],
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			DataRetentionSize:             &[]string{"test"}[0],
		},
		ResizeInfo: &privateopenapi.DataPlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &privateopenapi.DataPlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{10000}[0],
				Partitions:                    &[]int32{10000}[0],
				IngressEgressThroughputPerSec: &[]string{"test"}[0],
				DataRetentionSize:             &[]string{"test"}[0],
			},
		},
	}
}

// Return a Kafka status for a deleted cluster
func GetDeletedKafkaStatusResponse() privateopenapi.DataPlaneKafkaStatus {
	return privateopenapi.DataPlaneKafkaStatus{
		Conditions: []privateopenapi.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Deleted",
			},
		},
	}
}

// Return a kafka status for a ready cluster
func GetReadyKafkaStatusResponse() privateopenapi.DataPlaneKafkaStatus {
	return privateopenapi.DataPlaneKafkaStatus{
		Conditions: []privateopenapi.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
}

func GetErrorKafkaStatusResponse() privateopenapi.DataPlaneKafkaStatus {
	return privateopenapi.DataPlaneKafkaStatus{
		Conditions: []privateopenapi.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Error",
				Status: "False",
			},
		},
	}
}

func GetErrorWithCustomMessageKafkaStatusResponse(message string) privateopenapi.DataPlaneKafkaStatus {
	res := GetErrorKafkaStatusResponse()
	res.Conditions[0].Message = message
	return res
}
