package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func TestDataPlaneCluster_ClusterStatusTransitionsToReadySuccessfully(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	commonKafkaVersions := []string{"2.8.0", "1.3.6", "2.7.0"}
	commonKafkaIBPVersions := []string{"2.8", "1.3", "2.7"}
	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
		private.DataPlaneClusterUpdateStatusRequestStrimzi{
			Version:          "strimzi-cluster-operator.v.5.12.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		private.DataPlaneClusterUpdateStatusRequestStrimzi{
			Version:          "strimzi-cluster-operator.v.5.8.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
		private.DataPlaneClusterUpdateStatusRequestStrimzi{
			Version:          "strimzi-cluster-operator.v.3.0.0-0",
			Ready:            true,
			KafkaVersions:    commonKafkaVersions,
			KafkaIbpVersions: commonKafkaIBPVersions,
		},
	}
	expectedCommonKafkaVersions := []api.KafkaVersion{
		api.KafkaVersion{Version: "1.3.6"},
		api.KafkaVersion{Version: "2.7.0"},
		api.KafkaVersion{Version: "2.8.0"},
	}
	expectedCommonKafkaIBPVersions := []api.KafkaIBPVersion{
		api.KafkaIBPVersion{Version: "1.3"},
		api.KafkaIBPVersion{Version: "2.7"},
		api.KafkaIBPVersion{Version: "2.8"},
	}
	expectedAvailableStrimziVersions := []api.StrimziVersion{
		{Version: "strimzi-cluster-operator.v.3.0.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
		{Version: "strimzi-cluster-operator.v.5.8.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
		{Version: "strimzi-cluster-operator.v.5.12.0-0", Ready: true, KafkaVersions: expectedCommonKafkaVersions, KafkaIBPVersions: expectedCommonKafkaIBPVersions},
	}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_BadRequestWhenNonexistingCluster(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID := "test-cluster-id"
	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound)) // We expect 404 error in this test because the cluster ID does not exist
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound)) // We expect 404 error in this test because the cluster ID does not exist
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_UnauthorizedWhenNoAuthProvided(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	ctx := context.Background()
	testDataPlaneclusterID := "test-cluster-id"

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_NotFoundWhenNoProperAuthRole(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedRoleForDataPlaneCluster(h, testDataPlaneclusterID)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_NotFoundWhenNotAllowedClusterID(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	privateAPIClient := test.NewPrivateAPIClient(h)
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedClusterIDForDataPlaneCluster(h)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, private.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_GetManagedKafkaAgentCRSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)
	config, resp, err := privateAPIClient.AgentClustersApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred())
	Expect(config.Spec.Observability.Repository).ShouldNot(BeEmpty())
	Expect(config.Spec.Observability.Channel).ShouldNot(BeEmpty())
	Expect(config.Spec.Observability.AccessToken).ShouldNot(BeNil())
}

func TestDataPlaneCluster_ClusterStatusTransitionsToFullWhenNoMoreKafkaCapacity(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)
	ocmServerBuilder.SetSubscriptionSearchResponse(mockedClusterSubscritions("123"), nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())

	// Now create an OSD dummy cluster in the DB with same characteristics and
	// mark it as ready. This is done so no scale up is triggered
	// once the test data plane cluster is marked as full because the
	// dynamic scaling feature, that we enable in this test, would perform
	// an OSD cluster scale-up if all the clusters there are are full
	dummyClusterID := api.NewID()
	dummyCluster := api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:     dummyClusterID,
		MultiAZ:       cluster.MultiAZ,
		Region:        cluster.Region,
		CloudProvider: cluster.CloudProvider,
		Status:        api.ClusterReady,
	}

	db := test.TestServices.DBFactory.New()
	if err := db.Save(&dummyCluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}
	// Create dummy kafka and assign it to dummy cluster to make it not empty.
	// This is done so it is not scaled down by the dynamic scaling
	// functionality
	dummyKafka := dbapi.KafkaRequest{
		ClusterID:     dummyClusterID,
		MultiAZ:       false,
		Region:        cluster.Region,
		CloudProvider: cluster.CloudProvider,
		Name:          "dummy-kafka",
		Status:        constants2.KafkaRequestStatusReady.String(),
	}

	if err := db.Save(&dummyKafka).Error; err != nil {
		t.Error("failed to create a dummy kafka request")
		return
	}

	// Create dummy kafka and assign it to test data plane cluster to make it not
	// empty. This is done so it is not scaled down by the dynamic scaling
	// functionality
	dummyKafka.ID = api.NewID()
	dummyKafka.ClusterID = testDataPlaneclusterID
	if err := db.Save(&dummyKafka).Error; err != nil {
		t.Error("failed to create a dummy kafka request")
		return
	}

	// We enable Dynamic Scaling at this point and not in the startHook due to
	// we want to ensure the pre-existing OSD cluster entry is stored in the DB
	// before enabling the dynamic scaling logic
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{1000000}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{0}[0]

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterFull))

	// We force status to 'ready' at DB level to ensure no cluster is recreated
	// again
	err = test.TestServices.DBFactory.DB.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("status", api.ClusterReady).Error
	Expect(err).ToNot(HaveOccurred())
}
func TestDataPlaneCluster_ClusterStatusTransitionsToWaitingForKASFleetOperatorWhenOperatorIsNotReady(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	// enable dynamic autoscaling
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.Conditions[0].Status = "False"

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterWaitingForKasFleetShardOperator))
}

func DataplaneClusterConfig(h *coreTest.Helper) (dataplaneClusterConfig *config.DataplaneClusterConfig) {
	h.Env.MustResolve(&dataplaneClusterConfig)
	return
}

func TestDataPlaneCluster_TestScaleUpAndDown(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedCluster, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedCluster, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	// only run this test when real OCM API is being used
	if test.TestServices.OCMConfig.MockMode == ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	// enable auto scaling
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	ocmClient := test.TestServices.OCMClient

	ocmCluster, err := ocmClient.GetCluster(testDataPlaneclusterID)
	initialComputeNodes := ocmCluster.Nodes().Compute()
	Expect(initialComputeNodes).NotTo(BeNil())
	Expect(err).ToNot(HaveOccurred())
	expectedNodesAfterScaleUp := initialComputeNodes + 3

	kafkaCapacityConfig := KafkaConfig(h).KafkaCapacity
	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.ResizeInfo.NodeDelta = &[]int32{3}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.Connections = &[]int32{int32(kafkaCapacityConfig.TotalMaxConnections) * 30}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.Partitions = &[]int32{int32(kafkaCapacityConfig.MaxPartitions) * 30}[0]
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{int32(kafkaCapacityConfig.TotalMaxConnections) - 1}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{int32(kafkaCapacityConfig.MaxPartitions) - 1}[0]
	clusterStatusUpdateRequest.NodeInfo.Ceiling = &[]int32{int32(expectedNodesAfterScaleUp)}[0]
	clusterStatusUpdateRequest.NodeInfo.Current = &[]int32{int32(initialComputeNodes)}[0]
	clusterStatusUpdateRequest.NodeInfo.CurrentWorkLoadMinimum = &[]int32{3}[0]
	clusterStatusUpdateRequest.NodeInfo.Floor = &[]int32{3}[0]

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterComputeNodeScalingUp))

	checkComputeNodesFunc := func(clusterID string, expectedNodes int) func() (bool, error) {
		return func() (bool, error) {
			currOcmCluster, err := ocmClient.GetCluster(clusterID)
			if err != nil {
				return false, err
			}

			if currOcmCluster.Nodes().Compute() != expectedNodes {
				return false, nil
			}
			metrics, err := ocmClient.GetExistingClusterMetrics(clusterID)
			if err != nil {
				return false, err
			}
			if int(metrics.Nodes().Compute()) != expectedNodes {
				return false, nil
			}

			return true, nil
		}
	}

	// Check that desired and existing compute nodes end being the
	// expected ones
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(5*time.Second, 60*time.Minute).
		RetryLogMessagef("Waiting for cluster '%s' to scale up to %d nodes", testDataPlaneclusterID, expectedNodesAfterScaleUp).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			return checkComputeNodesFunc(testDataPlaneclusterID, expectedNodesAfterScaleUp)()
		}).Build().Poll()

	Expect(err).ToNot(HaveOccurred())

	// We force a scale-down by setting one of the remaining fields to be
	// higher than the scale-down threshold.
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{*clusterStatusUpdateRequest.ResizeInfo.Delta.Connections + 1}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{int32(*clusterStatusUpdateRequest.ResizeInfo.Delta.Partitions) + 1}[0]
	clusterStatusUpdateRequest.NodeInfo.Current = &[]int32{int32(expectedNodesAfterScaleUp)}[0]
	resp, err = privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())
	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))

	// Check that desired and existing compute nodes end being the
	// expected ones
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(5*time.Second, 60*time.Minute).
		RetryLogMessagef("Waiting for cluster '%s' to scale down to %d nodes", testDataPlaneclusterID, initialComputeNodes).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			return checkComputeNodesFunc(testDataPlaneclusterID, initialComputeNodes)()
		}).Build().Poll()

	Expect(err).ToNot(HaveOccurred())
}

func TestDataPlaneCluster_TestOSDClusterScaleUp(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedCluster, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedCluster, nil)
	ocmServerBuilder.SetSubscriptionSearchResponse(mockedClusterSubscritions(mockedCluster.ID()), nil)
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	// We enable Dynamic Scaling at this point and not in the startHook due to
	// we want to ensure the pre-existing OSD cluster entry is stored in the DB
	// before enabling the dynamic scaling logic
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	initialExpectedOSDClusters := 1
	// Check that at this moment we should only have one cluster
	db := test.TestServices.DBFactory.New()
	var count int64
	err = db.Model(&api.Cluster{}).Count(&count).Error
	Expect(err).ToNot(HaveOccurred())
	Expect(count).To(Equal(int64(initialExpectedOSDClusters)))

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	ocmClient := test.TestServices.OCMClient

	ocmCluster, err := ocmClient.GetCluster(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())

	initialComputeNodes := ocmCluster.Nodes().Compute()
	Expect(initialComputeNodes).NotTo(BeNil())
	// Simulate there's no capacity and we've already reached ceiling to
	// set status as full and force the cluster mgr reconciler to create a new
	// OSD cluster
	kafkaCapacityConfig := KafkaConfig(h).KafkaCapacity
	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.ResizeInfo.NodeDelta = &[]int32{3}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.Connections = &[]int32{int32(kafkaCapacityConfig.TotalMaxConnections) * 30}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.Partitions = &[]int32{int32(kafkaCapacityConfig.MaxPartitions) * 30}[0]
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{0}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{0}[0]
	clusterStatusUpdateRequest.NodeInfo.Ceiling = &[]int32{int32(initialComputeNodes)}[0]
	clusterStatusUpdateRequest.NodeInfo.Current = &[]int32{int32(initialComputeNodes)}[0]
	clusterStatusUpdateRequest.NodeInfo.CurrentWorkLoadMinimum = &[]int32{3}[0]
	clusterStatusUpdateRequest.NodeInfo.Floor = &[]int32{3}[0]

	// Swap mock clusters post response to create a cluster with a different
	// ClusterID so we simulate two different clusters are created
	newMockedCluster, err := mockedClusterWithClusterID("new-mocked-test-cluster-id")
	Expect(err).ToNot(HaveOccurred())
	ocmServerBuilder.SwapRouterResponse(mocks.EndpointPathClusters, http.MethodPost, newMockedCluster, nil)

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterFull))

	// Wait until the new cluster is created in the DB
	Eventually(func() int64 {
		var count int64
		err := db.Model(&api.Cluster{}).Count(&count).Error
		if err != nil {
			return -1
		}
		return count
	}, 5*time.Minute, 5*time.Second).Should(Equal(int64(initialExpectedOSDClusters + 1)))

	clusterCreationTimeout := 3 * time.Hour
	var newCluster *api.Cluster
	// wait for the new cluster to have an ID
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(5*time.Second, clusterCreationTimeout).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			clusters := []api.Cluster{}
			db.Find(&clusters)
			err = db.Where("cluster_id <> ?", testDataPlaneclusterID).Find(&clusters).Error
			if err != nil {
				return false, err
			}

			if len(clusters) != 1 {
				return false, nil
			}

			newCluster = &clusters[0]
			return newCluster.ClusterID != "", nil
		}).
		RetryLogMessage("Waiting for the new cluster to have an ID").
		Build().Poll()

	Expect(err).NotTo(HaveOccurred())

	// We force status to 'ready' at DB level to ensure no cluster is recreated
	// again when deleting the new cluster
	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("status", api.ClusterReady).Error
	Expect(err).ToNot(HaveOccurred())
	err = db.Save(&dbapi.KafkaRequest{ClusterID: testDataPlaneclusterID, Status: string(constants2.KafkaRequestStatusReady)}).Error
	Expect(err).ToNot(HaveOccurred())
	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", newCluster.ClusterID).Update("status", api.ClusterReady).Error
	Expect(err).ToNot(HaveOccurred())

	err = common.WaitForClusterToBeDeleted(test.TestServices.DBFactory, &test.TestServices.ClusterService, newCluster.ClusterID)

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster deletion: %v", err)
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsEmptyAndClusterStrimziVersionsIsEmptyItRemainsEmpty(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{}
	expectedAvailableStrimziVersions := []api.StrimziVersion{}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))

}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsNilAndClusterStrimziVersionsIsEmptyItRemainsEmpty(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := test.NewPrivateAPIClient(h)

	clusterStatusUpdateRequest := kasfleetshardsync.SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	clusterStatusUpdateRequest.Strimzi = nil
	expectedAvailableStrimziVersions := []api.StrimziVersion{}
	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsEmptyAndClusterStrimziVersionsIsNotEmptyItRemainsUnchanged(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
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
	Expect(err).ToNot(HaveOccurred())
	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
	Expect(err).NotTo(HaveOccurred())

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsIsNilAndClusterStrimziVersionsIsNotEmptyItRemainsUnchanged(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
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
	Expect(err).ToNot(HaveOccurred())
	cluster, err := test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
	Expect(err).NotTo(HaveOccurred())

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
}

func TestDataPlaneCluster_WhenReportedStrimziVersionsAreDifferentClusterStrimziVersionsIsUpdated(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := common.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	db := test.TestServices.DBFactory.New()

	initialAvailableStrimziVersionsStr, err := json.Marshal([]api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v.8.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.9.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.10.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
	})

	Expect(err).NotTo(HaveOccurred())

	err = db.Model(&api.Cluster{}).Where("cluster_id = ?", testDataPlaneclusterID).Update("available_strimzi_versions", initialAvailableStrimziVersionsStr).Error
	Expect(err).ToNot(HaveOccurred())

	ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, testDataPlaneclusterID)
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
	Expect(err).ToNot(HaveOccurred())
	availableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	Expect(availableStrimziVersions).To(Equal([]api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v.8.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.9.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v.10.0.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
			},
		},
	}))
	Expect(err).ToNot(HaveOccurred())

	resp, err := privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	cluster, err = test.TestServices.ClusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
	availableStrimziVersions, err = cluster.GetAvailableStrimziVersions()
	Expect(err).ToNot(HaveOccurred())

	expectedCommonKafkaVersions := []api.KafkaVersion{
		api.KafkaVersion{Version: "2.7.0"},
		api.KafkaVersion{Version: "2.8.0"},
	}
	expectedCommonKafkaIBPVersions := []api.KafkaIBPVersion{
		api.KafkaIBPVersion{Version: "2.7"},
		api.KafkaIBPVersion{Version: "2.8"},
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
	Expect(availableStrimziVersions).To(Equal(expectedAvailableStrimziVersions))
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
		Total: private.DataPlaneClusterUpdateStatusRequestTotal{
			IngressThroughputPerSec: &[]string{""}[0],
			EgressThroughputPerSec:  &[]string{""}[0],
			Connections:             &[]int32{0}[0],
			DataRetentionSize:       &[]string{""}[0],
			Partitions:              &[]int32{0}[0],
		},
		NodeInfo: &private.DatePlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{0}[0],
			Floor:                  &[]int32{0}[0],
			Current:                &[]int32{0}[0],
			CurrentWorkLoadMinimum: &[]int32{0}[0],
		},
		Remaining: private.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:             &[]int32{0}[0],
			Partitions:              &[]int32{0}[0],
			IngressThroughputPerSec: &[]string{""}[0],
			EgressThroughputPerSec:  &[]string{""}[0],
			DataRetentionSize:       &[]string{""}[0],
		},
		ResizeInfo: &private.DatePlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &private.DatePlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:             &[]int32{0}[0],
				Partitions:              &[]int32{0}[0],
				IngressThroughputPerSec: &[]string{""}[0],
				EgressThroughputPerSec:  &[]string{""}[0],
				DataRetentionSize:       &[]string{""}[0],
			},
		},
	}
}

func mockedClusterSubscritions(clusterID string) *amsv1.SubscriptionList {
	subscriptionListBuilder := amsv1.NewSubscriptionList()
	subscriptionBuilder := amsv1.NewSubscription()
	subscriptionBuilder.ClusterID(clusterID)

	subscriptionsMetricsBuilder := amsv1.NewSubscriptionMetrics()
	clusterMetricsNodeBuilder := amsv1.NewClusterMetricsNodes()
	clusterMetricsNodeBuilder.Compute(6)
	subscriptionsMetricsBuilder.Nodes(clusterMetricsNodeBuilder)

	subscriptionBuilder.Metrics(subscriptionsMetricsBuilder)

	subscriptionListBuilder.Items(subscriptionBuilder)
	subscriptionList, _ := subscriptionListBuilder.Build()

	return subscriptionList
}

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}

func mockedClusterWithClusterID(clusterID string) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterBuilder.ID(clusterID)
	return clusterBuilder.Build()
}
