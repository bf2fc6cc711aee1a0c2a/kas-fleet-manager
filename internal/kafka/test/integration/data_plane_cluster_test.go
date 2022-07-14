package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	dataplanemocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func TestDataPlaneCluster_GetManagedKafkaAgentCRAndObserveCapacityInfo(t *testing.T) {
	g := gomega.NewWithT(t)
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()
	var dataplaneConfig *config.DataplaneClusterConfig
	h, _, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, func(d *config.DataplaneClusterConfig) {
		dataplaneConfig = d
	})
	defer tearDown()

	cluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderOCM
		cluster.SupportedInstanceType = "standard"
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "some-cluster-id"
		cluster.CloudProvider = "aws"
		cluster.Region = "us-east-1"
		cluster.Status = "ready"
		cluster.ProviderSpec = nil
		cluster.ClusterSpec = nil
		cluster.DynamicCapacityInfo = api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`))
	})
	db := test.TestServices.DBFactory.New()
	err = db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)
	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, cluster.ClusterID)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// turn on autoscaling and observe that the node capacity info are sent
	dataplaneConfig.DataPlaneClusterScalingType = config.AutoScaling

	managedKafkaAgentCR, resp, err := privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, cluster.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(managedKafkaAgentCR.Spec.Capacity).To(gomega.HaveLen(1))
	capacity, ok := managedKafkaAgentCR.Spec.Capacity[cluster.SupportedInstanceType]
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(capacity.MaxNodes).To(gomega.Equal(int32(1)))

	// turn off autoscaling and observe that capacity info is now empty
	dataplaneConfig.DataPlaneClusterScalingType = config.NoScaling
	managedKafkaAgentCR, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, cluster.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(managedKafkaAgentCR.Spec.Capacity).To(gomega.HaveLen(0))
}

func TestDataPlaneCluster_ClusterStatusTransitionsToReadySuccessfully(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	privateAPIClient := test.NewPrivateAPIClient(h)

	commonKafkaVersions := []string{"2.8.0", "1.3.6", "2.7.0"}
	commonKafkaIBPVersions := []string{"2.8", "1.3", "2.7"}
	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
		{
			Version:          "strimzi-cluster-operator.v.5.12.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.5.8.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.3.0.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
	}
	expectedCommonKafkaVersions := []api.KafkaVersion{
		{Version: "1.3.6"},
		{Version: "2.7.0"},
		{Version: "2.8.0"},
	}
	expectedCommonKafkaIBPVersions := []api.KafkaIBPVersion{
		{Version: "1.3"},
		{Version: "2.7"},
		{Version: "2.8"},
	}
	expectedAvailableStrimziVersions := []api.StrimziVersion{
		{Version: "strimzi-cluster-operator.v.3.0.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
		{Version: "strimzi-cluster-operator.v.5.8.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
		{Version: "strimzi-cluster-operator.v.5.12.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
	}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))

	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_BadRequestWhenNonexistingCluster(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID := "test-cluster-id"
	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	privateAPIClient := test.NewPrivateAPIClient(h)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil))
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest)) // We Expect 400 error in this test because the cluster ID does not exist

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest)) // We expect 400 error in this test because the cluster ID does not exist
}

func TestDataPlaneCluster_UnauthorizedWhenNoAuthProvided(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	ctx := context.Background()
	testDataPlaneclusterID := "test-cluster-id"

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
}

func TestDataPlaneCluster_NotFoundWhenNoProperAuthRole(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedRoleForDataPlaneCluster(h, testDataPlaneclusterID)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
}

func TestDataPlaneCluster_NotFoundWhenNotAllowedClusterID(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedClusterIDForDataPlaneCluster(h)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
}

func TestDataPlaneCluster_GetManagedKafkaAgentCRSuccess(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)
	config, resp, err := privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(config.Spec.Observability.Repository).ShouldNot(gomega.BeEmpty())
	g.Expect(config.Spec.Observability.Channel).ShouldNot(gomega.BeEmpty())
	g.Expect(config.Spec.Observability.AccessToken).ShouldNot(gomega.BeNil())
}

func TestDataPlaneCluster_ClusterStatusTransitionsToWaitingForKASFleetOperatorWhenOperatorIsNotReady(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	// enable dynamic autoscaling
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.Conditions[0].Status = "False"

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterWaitingForKasFleetShardOperator))
}

func DataplaneClusterConfig(h *coreTest.Helper) (dataplaneClusterConfig *config.DataplaneClusterConfig) {
	h.Env.MustResolve(&dataplaneClusterConfig)
	return
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsEmptyAndClusterStrimziVersionsIsEmptyItRemainsEmpty(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{}
	expectedAvailableStrimziVersions := []api.StrimziVersion{}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))

	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))

}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsNilAndClusterStrimziVersionsIsEmptyItRemainsEmpty(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = nil
	expectedAvailableStrimziVersions := []api.StrimziVersion{}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))

	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsEmptyAndClusterStrimziVersionsIsNotEmptyItRemainsUnchanged(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{}
	expectedAvailableStrimziVersions := []api.StrimziVersion{
		{Version: "strimzi-cluster-operator.v.8.0.0-0", Ready: true},
		{Version: "strimzi-cluster-operator.v.9.0.0-0", Ready: false},
		{Version: "strimzi-cluster-operator.v.10.0.0-0", Ready: true},
	}
	db := test.TestServices.DBFactory.New()
	initialAvailableStrimziVersionsStr := `[{"version": "strimzi-cluster-operator.v.8.0.0-0", "ready": true}, {"version": "strimzi-cluster-operator.v.9.0.0-0", "ready": false}, {"version": "strimzi-cluster-operator.v.10.0.0-0", "ready": true}]`
	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("available_strimzi_versions", initialAvailableStrimziVersionsStr).Error
	g.Expect(err).ToNot(gomega.HaveOccurred())
	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsNilAndClusterStrimziVersionsIsNotEmptyItRemainsUnchanged(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = nil
	expectedAvailableStrimziVersions := []api.StrimziVersion{
		{Version: "strimzi-cluster-operator.v.8.0.0-0", Ready: true},
		{Version: "strimzi-cluster-operator.v.9.0.0-0", Ready: false},
		{Version: "strimzi-cluster-operator.v.10.0.0-0", Ready: true},
	}
	db := test.TestServices.DBFactory.New()
	initialAvailableStrimziVersionsStr := `[{"version": "strimzi-cluster-operator.v.8.0.0-0", "ready": true}, {"version": "strimzi-cluster-operator.v.9.0.0-0", "ready": false}, {"version": "strimzi-cluster-operator.v.10.0.0-0", "ready": true}]`
	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("available_strimzi_versions", initialAvailableStrimziVersionsStr).Error
	g.Expect(err).ToNot(gomega.HaveOccurred())
	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsAreDifferentClusterStrimziVersionsIsUpdated(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock cluster object")

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(getClusterErr).ToNot(gomega.HaveOccurred(), "failed to retrieve cluster details")
	g.Expect(testDataPlaneclusterID).ToNot(gomega.BeEmpty(), "cluster not found")

	db := test.TestServices.DBFactory.New()

	initialAvailableStrimziVersionsStr, err := json.Marshal([]api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v.8.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.9.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.10.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
	})

	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("available_strimzi_versions", initialAvailableStrimziVersionsStr).Error
	g.Expect(err).ToNot(gomega.HaveOccurred())

	ctx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	privateAPIClient := test.NewPrivateAPIClient(h)

	commonKafkaVersions := []string{"2.8.0", "2.7.0"}
	commonKafkaIBPVersions := []string{"2.8", "2.7"}

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
		{
			Version:          "strimzi-cluster-operator.v.5.0.0-0",
			Ready:            false,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.7.0.0-0",
			Ready:            false,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.3.0.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
	}
	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	g.Expect(availableStrimziVersions).To(gomega.Equal([]api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v.8.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.9.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.10.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.7"},
			},
		},
	}))
	g.Expect(err).ToNot(gomega.HaveOccurred())

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNoContent))

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster).ToNot(gomega.BeNil())
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	expectedCommonKafkaVersions := []api.KafkaVersion{
		{Version: "2.7.0"},
		{Version: "2.8.0"},
	}
	expectedCommonKafkaIBPVersions := []api.KafkaIBPVersion{
		{Version: "2.7"},
		{Version: "2.8"},
	}
	expectedAvailableStrimziVersions := []api.StrimziVersion{
		{
			Version:          "strimzi-cluster-operator.v.3.0.0-0",
			Ready:            true,
			KafkaVersions:    expectedCommonKafkaVersions,
			KafkaIBPVersions: expectedCommonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.5.0.0-0",
			Ready:            false,
			KafkaVersions:    expectedCommonKafkaVersions,
			KafkaIBPVersions: expectedCommonKafkaIBPVersions,
		},
		{
			Version:          "strimzi-cluster-operator.v.7.0.0-0",
			Ready:            false,
			KafkaVersions:    expectedCommonKafkaVersions,
			KafkaIBPVersions: expectedCommonKafkaIBPVersions,
		},
	}
	g.Expect(availableStrimziVersions).To(gomega.Equal(expectedAvailableStrimziVersions))
}

func KafkaConfig(h *coreTest.Helper) (c *config.KafkaConfig) {
	h.Env.MustResolve(&c)
	return
}

func newAuthContextWithNotAllowedRoleForDataPlaneCluster(h *coreTest.Helper, clusterID string) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"not_allowed_role_example"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx
}

func newAuthContextWithNotAllowedClusterIDForDataPlaneCluster(h *coreTest.Helper) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": "differentcluster",
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx
}

// sampleValidBaseDataPlaneClusterStatusRequest returns a valid
func sampleValidBaseDataPlaneClusterStatusRequest() *private.DataPlaneClusterUpdateStatusRequest {
	return &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Capacity: map[string]private.DataPlaneClusterUpdateStatusRequestCapacity{
			api.StandardTypeSupport.String(): {
				MaxUnits:       4,
				RemainingUnits: 0,
			},
			api.DeveloperTypeSupport.String(): {
				MaxUnits:       90,
				RemainingUnits: 89,
			},
		},
	}
}

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}

// nolint
func mockedClusterWithClusterID(clusterID string) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterBuilder.ID(clusterID)
	return clusterBuilder.Build()
}
