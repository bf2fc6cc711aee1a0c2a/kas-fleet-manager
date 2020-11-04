package integration

import (
	"net/http"
	"testing"

	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestClusterScaleUp(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	expectedReplicas := 2
	// scaleUp will result in one extra node if machinePool exists already (otherwise 2)
	if machinePoolExists(h, clusterID, t) {
		expectedReplicas = 1
	}

	mockMachinePool := getMachinePoolForScaleTest(expectedReplicas)
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)

	// create machine pool
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas++

	mockMachinePool = getMachinePoolForScaleTest(expectedReplicas)
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)

	// scale up by one node
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas--

	// scale down the nodes
	for ; 0 <= expectedReplicas; expectedReplicas-- {
		mockMachinePool = getMachinePoolForScaleTest(expectedReplicas)
		ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)
		scaleDownMachinePool(h, expectedReplicas, clusterID)
	}
}

func TestClusterScaleDown(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	expectedReplicas := 2
	// scaleUp will result in one extra node if machinePool exists already (otherwise 2)
	if machinePoolExists(h, clusterID, t) {
		expectedReplicas = 1
	}

	mockMachinePool := getMachinePoolForScaleTest(expectedReplicas)
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)

	// create/ scale up machine pool
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas--

	for ; 0 <= expectedReplicas; expectedReplicas-- {
		mockMachinePool = getMachinePoolForScaleTest(expectedReplicas)
		ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)
		scaleDownMachinePool(h, expectedReplicas, clusterID)
	}
}

// get mock MachinePool with specified replicas number
func getMachinePoolForScaleTest(replicas int) *clustersmgmtv1.MachinePool {
	mockClusterID := mocks.MockCluster.ID()
	mockCloudProviderID := mocks.MockCluster.CloudProvider().ID()
	mockClusterExternalID := mocks.MockCluster.ExternalID()
	mockClusterState := clustersmgmtv1.ClusterStateReady
	mockCloudProviderDisplayName := mocks.MockCluster.CloudProvider().DisplayName()
	mockCloudRegionID := mocks.MockCluster.CloudProvider().ID()
	mockMachinePoolID := "managed"
	mockCloudProviderBuilder := mocks.GetMockCloudProviderBuilder(mockCloudProviderID, mockCloudProviderDisplayName)
	mockCloudProviderRegionBuilder := mocks.GetMockCloudProviderRegionBuilder(mockCloudRegionID, mockCloudProviderID, mockCloudProviderDisplayName, mockCloudProviderBuilder, true, true)
	mockClusterBuilder := mocks.GetMockClusterBuilder(mockClusterID, mockClusterExternalID, mockClusterState, mockCloudProviderBuilder, mockCloudProviderRegionBuilder)
	mockMachinePoolBuilder := mocks.GetMockMachineBuilder(mockMachinePoolID, mockClusterID, replicas, mockClusterBuilder)
	mockMachinePool, e := mocks.GetMockMachinePool(mockMachinePoolBuilder)
	if e != nil {
		panic(e)
	}
	return mockMachinePool
}

// scaleUpMachinePool and confirm that it is scaled without error
func scaleUpMachinePool(h *test.Helper, expectedReplicas int, clusterID string) {
	machinePool, err := h.Env().Services.Cluster.ScaleUpMachinePool(clusterID)
	Expect(err).To(BeNil())
	Expect(machinePool.ID()).To(Equal(services.DefaultMachinePoolID))
	Expect(machinePool.Replicas()).To(Equal(expectedReplicas))
}

// scaleDownMachinePool and confirm that it is scaled without error
func scaleDownMachinePool(h *test.Helper, expectedReplicas int, clusterID string) {
	machinePool, err := h.Env().Services.Cluster.ScaleDownMachinePool(clusterID)
	Expect(err).To(BeNil())
	Expect(machinePool.ID()).To(Equal(services.DefaultMachinePoolID))
	Expect(machinePool.Replicas()).To(Equal(expectedReplicas))
}

// machinePoolExists returns true if MachinePool already exists for a cluster with specified clusterID
func machinePoolExists(h *test.Helper, clusterID string, t *testing.T) bool {
	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	machinePoolExists, err := ocmClient.MachinePoolExists(clusterID, services.DefaultMachinePoolID)
	if err != nil {
		t.Fatalf("Failed to get MachinePool details from cluster: %s", clusterID)
	}
	return machinePoolExists
}
