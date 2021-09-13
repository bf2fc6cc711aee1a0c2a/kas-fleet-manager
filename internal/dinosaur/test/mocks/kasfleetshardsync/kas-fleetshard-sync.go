package kasfleetshardsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test"
	"github.com/dgrijalva/jwt-go"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// defaultUpdateDataplaneClusterStatusFunc - The default behaviour for updating data plane cluster status in each Kas Fleetshard Sync reconcile.
// Retrieves all clusters in the database in a 'waiting_for_kas_fleetshard_operator' state and updates it to 'ready' once all of the addons are installed.
var defaultUpdateDataplaneClusterStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error {
	var clusterService services.ClusterService
	var ocmConfig *ocm.OCMConfig
	helper.Env.MustResolveAll(&clusterService, &ocmConfig)
	clusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterWaitingForKasFleetShardOperator,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterWaitingForKasFleetShardOperator, err)
	}

	for _, cluster := range clusters {
		managedDinosaurAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.StrimziOperatorAddonID)
		if err != nil {
			return err
		}

		kasFleetShardOperatorAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.KasFleetshardAddonID)
		if err != nil {
			return err
		}

		// The KAS Fleetshard Operator and Sync containers are restarted every ~5mins due to the informer state being out of sync.
		// Because of this, the addonInstallation state in ocm may never get updated to a 'ready' state as the install plan
		// status gets updated everytime the container gets restarted even though the operator itself has been successfully installed.
		// This should get better in the next release of the fleetshard operator. Until then, the cluster can be deemed ready if
		// it's in an 'installing' state, therwise this poll may timeout.
		if managedDinosaurAddon.State() == clustersmgmtv1.AddOnInstallationStateReady &&
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

// defaultUpdateDinosaurStatusFunc - The default behaviour for updating dinosaur status in each Kas Fleetshard Sync reconcile.
// This function retrieves and updates Dinosaurs in all ready and full data plane clusters.
// Any Dinosaurs marked for deletion are updated to 'deleting'
// Dinosaurs with any other status are updated to 'ready'
var defaultUpdateDinosaurStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient) error {
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

		dinosaurList, _, err := privateClient.AgentClustersApi.GetPineapples(ctx, dataplaneCluster.ClusterID)
		if err != nil {
			return err
		}

		dinosaurStatusList := make(map[string]private.DataPlanePineappleStatus)
		for _, dinosaur := range dinosaurList.Items {
			id := dinosaur.Metadata.Annotations.Bf2OrgId
			if dinosaur.Spec.Deleted {
				dinosaurStatusList[id] = GetDeletedDinosaurStatusResponse()
			} else {
				// Update any other clusters not in a 'deprovisioning' state to 'ready'
				dinosaurStatusList[id] = GetReadyDinosaurStatusResponse()
			}
		}

		if _, err = privateClient.AgentClustersApi.UpdatePineappleClusterStatus(ctx, dataplaneCluster.ClusterID, dinosaurStatusList); err != nil {
			return err
		}
	}
	return nil
}

type MockKasFleetshardSyncBuilder interface {
	// SetUpdateDataplaneClusterStatusFunc - Sets behaviour for updating dataplane clusters in each KAS Fleetshard sync reconcile
	SetUpdateDataplaneClusterStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error)
	// SetUpdateDinosaurStatusFunc - Sets behaviour for updating dinosaur clusters in each KAS Fleetshard sync reconcile
	SetUpdateDinosaurStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error)
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
			updateDinosaurClusterStatus:  defaultUpdateDinosaurStatusFunc,
			interval:                     1 * time.Second,
		},
	}
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateDataplaneClusterStatusFunc(updateDataplaneClusterStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error) {
	m.kfsync.updateDataplaneClusterStatus = updateDataplaneClusterStatusFunc
}

func (m *mockKasFleetshardSyncBuilder) SetUpdateDinosaurStatusFunc(updateDinosaurStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient) error) {
	m.kfsync.updateDinosaurClusterStatus = updateDinosaurStatusFunc
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
	updateDinosaurClusterStatus  func(helper *coreTest.Helper, privateClient *private.APIClient) error
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
			m.reconcileDinosaurClusters()
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

func (m *mockKasFleetshardSync) reconcileDinosaurClusters() {
	if err := m.updateDinosaurClusterStatus(m.helper, m.privateClient); err != nil {
		m.t.Logf("Failed to update Dinosaur cluster status: %s", err)
	}
}

// Returns an authenticated context to be used for calling the data plane endpoints
func NewAuthenticatedContextForDataPlaneCluster(h *coreTest.Helper, clusterID string) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": keycloakConfig.DinosaurRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
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
		PineappleOperator: []private.DataPlaneClusterUpdateStatusRequestPineappleOperator{
			{
				Ready:   true,
				Version: "strimzi-cluster-operator.v0.23.0-0",
			},
			{
				Ready:   true,
				Version: "strimzi-cluster-operator.v0.21.0-0",
			},
		},
		Remaining: private.DataPlaneClusterUpdateStatusRequestRemaining{
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

// Return a Dinosaur status for a deleted cluster
func GetDeletedDinosaurStatusResponse() private.DataPlanePineappleStatus {
	return private.DataPlanePineappleStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Deleted",
			},
		},
	}
}

func GetDefaultReportedDinosaurVersion() string {
	return test.TestServices.DinosaurConfig.DefaultDinosaurVersion
}

func GetDefaultReportedStrimziVersion() string {
	return "strimzi-cluster-operator.v0.23.0-0"
}

// Return a dinosaur status for a ready cluster
func GetReadyDinosaurStatusResponse() private.DataPlanePineappleStatus {
	return private.DataPlanePineappleStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Versions: private.DataPlanePineappleStatusVersions{
			Pineapple:         GetDefaultReportedDinosaurVersion(),
			PineappleOperator: GetDefaultReportedStrimziVersion(),
		},
	}
}

func GetErrorDinosaurStatusResponse() private.DataPlanePineappleStatus {
	return private.DataPlanePineappleStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Error",
				Status: "False",
			},
		},
	}
}

func GetErrorWithCustomMessageDinosaurStatusResponse(message string) private.DataPlanePineappleStatus {
	res := GetErrorDinosaurStatusResponse()
	res.Conditions[0].Message = message
	return res
}
