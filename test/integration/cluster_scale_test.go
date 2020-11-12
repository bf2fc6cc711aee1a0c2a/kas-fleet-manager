package integration

import (
	"fmt"
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

	expectedReplicas := 3
	// scaleUp will result in one extra node if machinePool exists already (otherwise 2)
	if machinePoolExists(h, clusterID, t) {
		expectedReplicas = 1
	}

	overrideMachinePoolMockResponse(ocmServerBuilder, expectedReplicas)

	// create machine pool
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas++

	overrideMachinePoolMockResponse(ocmServerBuilder, expectedReplicas)

	// scale up by one node
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas--

	scaleDownAfterTest(ocmServerBuilder, h, expectedReplicas, clusterID)
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

	expectedReplicas := 3
	// scaleUp will result in one extra node if machinePool exists already (otherwise 2)
	if machinePoolExists(h, clusterID, t) {
		expectedReplicas = 1
	}
	overrideMachinePoolMockResponse(ocmServerBuilder, expectedReplicas)

	// create/ scale up machine pool
	scaleUpMachinePool(h, expectedReplicas, clusterID)

	expectedReplicas--

	scaleDownAfterTest(ocmServerBuilder, h, expectedReplicas, clusterID)
}

// get mock MachinePool with specified replicas number
func getMachinePoolForScaleTest(replicas int) *clustersmgmtv1.MachinePool {
	mockMachinePoolID := "managed"
	mockClusterBuilder := mocks.GetMockClusterBuilder(nil)
	mockMachinePoolBuilder := mocks.GetMockMachineBuilder(func(builder *clustersmgmtv1.MachinePoolBuilder) {
		(*builder).ID(mockMachinePoolID).Replicas(replicas).Cluster(mockClusterBuilder).
			HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/machine_pools/%s", mockMachinePoolID, mockMachinePoolID))
	})
	mockMachinePool, e := mocks.GetMockMachinePool(func(pool *clustersmgmtv1.MachinePool, err error) {
		p, err := mockMachinePoolBuilder.Build()
		if p == nil {
			panic(err)
		}
		*pool = *p
	})
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

// scaleDownAfterTest to have 0 extra nodes
func scaleDownAfterTest(ocmServerBuilder *mocks.MockConfigurableServerBuilder, h *test.Helper, expectedReplicas int, clusterID string) {
	// scale down the nodes to 0
	for ; 0 <= expectedReplicas; expectedReplicas-- {
		overrideMachinePoolMockResponse(ocmServerBuilder, expectedReplicas)
		scaleDownMachinePool(h, expectedReplicas, clusterID)
	}
}

// overrideMachinePoolMockResponse - override mock response for MachinePool patch
func overrideMachinePoolMockResponse(ocmServerBuilder *mocks.MockConfigurableServerBuilder, expectedReplicas int) {
	mockMachinePool := getMachinePoolForScaleTest(expectedReplicas)
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathMachinePool, http.MethodPatch, mockMachinePool, nil)
}
