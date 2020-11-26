package integration

import (
	"net/http"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestClusterComputeNodesScaling(t *testing.T) {
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

	expectedReplicas := constants.ClusterNodeScaleIncrement * 2 // 3 (initially) + 3 after scaling

	overrideClusterMockResponse(ocmServerBuilder, expectedReplicas)

	// scale up compute nodes to 6
	scaleUpComputeNodes(h, expectedReplicas, clusterID)

	expectedReplicas = expectedReplicas - constants.ClusterNodeScaleIncrement

	overrideClusterMockResponse(ocmServerBuilder, expectedReplicas)

	// scale down to 3
	scaleDownComputeNodes(h, expectedReplicas, clusterID)
}

// get mock Cluster with specified Compute replicas number
func getClusterForScaleTest(replicas int) *clustersmgmtv1.Cluster {
	nodesBuilder := clustersmgmtv1.NewClusterNodes().
		Compute(replicas)
	mockClusterBuilder := mocks.GetMockClusterBuilder(func(builder *clustersmgmtv1.ClusterBuilder) {
		(*builder).Nodes(nodesBuilder)
	})
	cluster, err := mockClusterBuilder.Build()
	if err != nil {
		panic(err)
	}
	return cluster
}

// scaleUpComputeNodes and confirm that it is scaled without error
func scaleUpComputeNodes(h *test.Helper, expectedReplicas int, clusterID string) {
	cluster, err := h.Env().Services.Cluster.ScaleUpComputeNodes(clusterID)
	Expect(err).To(BeNil())
	Expect(cluster.Nodes().Compute()).To(Equal(expectedReplicas))
}

// scaleDownComputeNodes and confirm that it is scaled without error
func scaleDownComputeNodes(h *test.Helper, expectedReplicas int, clusterID string) {
	cluster, err := h.Env().Services.Cluster.ScaleDownComputeNodes(clusterID)
	Expect(err).To(BeNil())
	Expect(cluster.Nodes().Compute()).To(Equal(expectedReplicas))
}

// overrideClusterMockResponse - override mock response for Cluster patch
func overrideClusterMockResponse(ocmServerBuilder *mocks.MockConfigurableServerBuilder, expectedReplicas int) {
	mockCluster := getClusterForScaleTest(expectedReplicas)
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathCluster, http.MethodPatch, mockCluster, nil)
}
