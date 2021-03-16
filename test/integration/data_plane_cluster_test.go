package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	ocm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/dgrijalva/jwt-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/onsi/gomega"
)

func TestDataPlaneCluster_ClusterStatusTransitionsToReadySuccessfully(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := utils.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()

	clusterStatusUpdateRequest := sampleDataPlaneclusterStatusRequestWithAvailableCapacity()
	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.ClusterCreationConfig)
	cluster, err := clusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))
}

func TestDataPlaneCluster_BadRequestWhenNonexistingCluster(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, openapi.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest)) // We expect 400 error in this test because the cluster ID does not exist
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.DefaultApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest)) // We expect 400 error in this test because the cluster ID does not exist
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_UnauthorizedWhenNoAuthProvided(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	privateAPIClient := h.NewPrivateAPIClient()
	ctx := context.Background()
	testDataPlaneclusterID := "test-cluster-id"

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, openapi.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.DefaultApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_NotFoundWhenNoProperAuthRole(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	privateAPIClient := h.NewPrivateAPIClient()
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedRoleForDataPlaneCluster(h, testDataPlaneclusterID)

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, openapi.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.DefaultApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_NotFoundWhenNotAllowedClusterID(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	privateAPIClient := h.NewPrivateAPIClient()
	testDataPlaneclusterID := "test-cluster-id"
	ctx := newAuthContextWithNotAllowedClusterIDForDataPlaneCluster(h)

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, openapi.DataPlaneClusterUpdateStatusRequest{})
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())

	_, resp, err = privateAPIClient.DefaultApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	Expect(err).To(HaveOccurred())
}

func TestDataPlaneCluster_GetManagedKafkaAgentCRSuccess(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := utils.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()
	config, resp, err := privateAPIClient.DefaultApi.GetKafkaAgent(ctx, testDataPlaneclusterID)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred())
	Expect(config.Spec.Observability.Repository).ShouldNot(BeEmpty())
	Expect(config.Spec.Observability.Channel).ShouldNot(BeEmpty())
}

func TestDataPlaneCluster_ClusterStatusTransitionsToFullWhenNoMoreKafkaCapacity(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := utils.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()

	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{1000000}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{0}[0]

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.ClusterCreationConfig)
	cluster, err := clusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterFull))
}
func TestDataPlaneCluster_ClusterStatusTransitionsToWaitingForKASFleetOperatorWhenOperatorIsNotReady(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	testDataPlaneclusterID, getClusterErr := utils.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()

	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.Conditions[0].Status = "False"

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.ClusterCreationConfig)
	cluster, err := clusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterWaitingForKasFleetShardOperator))
}

func TestDataPlaneCluster_TestScaleUpAndDown(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.EnableKasFleetshardSync = false
	}
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedCluster, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedCluster, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	// only run this test when real OCM API is being used
	if h.Env().Config.OCM.MockMode == config.MockModeEmulateServer {
		t.SkipNow()
	}

	testDataPlaneclusterID, getClusterErr := utils.GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterWaitingForKasFleetShardOperator)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if testDataPlaneclusterID == "" {
		t.Fatalf("Cluster not found")
	}

	ctx := newAuthenticatedContexForDataPlaneCluster(h, testDataPlaneclusterID)
	privateAPIClient := h.NewPrivateAPIClient()

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	ocmCluster, err := ocmClient.GetCluster(testDataPlaneclusterID)
	initialComputeNodes := ocmCluster.Nodes().Compute()
	Expect(initialComputeNodes).NotTo(BeNil())
	Expect(err).ToNot(HaveOccurred())
	expectedNodesAfterScaleUp := initialComputeNodes + 3

	kafkaCapacityConfig := h.Env().Config.Kafka.KafkaCapacity
	clusterStatusUpdateRequest := sampleValidBaseDataPlaneClusterStatusRequest()
	clusterStatusUpdateRequest.ResizeInfo.NodeDelta = &[]int32{3}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.Connections = &[]int32{int32(kafkaCapacityConfig.TotalMaxConnections) * 30}[0]
	clusterStatusUpdateRequest.ResizeInfo.Delta.MaxPartitions = &[]int32{int32(kafkaCapacityConfig.MaxPartitions) * 30}[0]
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{int32(kafkaCapacityConfig.TotalMaxConnections) - 1}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{int32(kafkaCapacityConfig.MaxPartitions) - 1}[0]
	clusterStatusUpdateRequest.NodeInfo.Ceiling = &[]int32{int32(expectedNodesAfterScaleUp)}[0]
	clusterStatusUpdateRequest.NodeInfo.Current = &[]int32{int32(initialComputeNodes)}[0]
	clusterStatusUpdateRequest.NodeInfo.CurrentWorkLoadMinimum = &[]int32{3}[0]
	clusterStatusUpdateRequest.NodeInfo.Floor = &[]int32{3}[0]

	resp, err := privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())

	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.ClusterCreationConfig)
	cluster, err := clusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterFull))

	checkComputeNodesFunc := func(clusterID string, expectedNodes int) func() bool {
		return func() bool {
			currOcmCluster, err := ocmClient.GetCluster(clusterID)
			if err != nil {
				return false
			}

			if currOcmCluster.Nodes().Compute() != expectedNodes {
				return false
			}

			if currOcmCluster.Metrics().Nodes().Compute() != expectedNodes {
				return false
			}

			return true
		}
	}

	// Check that desired and existing compute nodes end being the
	// expected ones
	Eventually(checkComputeNodesFunc(testDataPlaneclusterID, expectedNodesAfterScaleUp), 60*time.Minute, 5*time.Second).Should(BeTrue())

	// We force a scale-down by setting one of the remaining fields to be
	// higher than the scale-down threshold.
	clusterStatusUpdateRequest.Remaining.Connections = &[]int32{*clusterStatusUpdateRequest.ResizeInfo.Delta.Connections + 1}[0]
	clusterStatusUpdateRequest.Remaining.Partitions = &[]int32{int32(*clusterStatusUpdateRequest.ResizeInfo.Delta.MaxPartitions) + 1}[0]
	clusterStatusUpdateRequest.NodeInfo.Current = &[]int32{int32(expectedNodesAfterScaleUp)}[0]
	resp, err = privateAPIClient.DefaultApi.UpdateAgentClusterStatus(ctx, testDataPlaneclusterID, *clusterStatusUpdateRequest)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	Expect(err).ToNot(HaveOccurred())
	cluster, err = clusterService.FindClusterByID(testDataPlaneclusterID)
	Expect(err).ToNot(HaveOccurred())
	Expect(cluster).ToNot(BeNil())
	Expect(cluster.Status).To(Equal(api.ClusterReady))

	// Check that desired and existing compute nodes end being the
	// expected ones
	Eventually(checkComputeNodesFunc(testDataPlaneclusterID, initialComputeNodes), 60*time.Minute, 5*time.Second).Should(BeTrue())
}

func newAuthenticatedContexForDataPlaneCluster(h *test.Helper, clusterID string) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": h.AppConfig.Keycloak.KafkaRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, token)

	return ctx
}

func newAuthContextWithNotAllowedRoleForDataPlaneCluster(h *test.Helper, clusterID string) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"not_allowed_role_example"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, token)

	return ctx
}

func newAuthContextWithNotAllowedClusterIDForDataPlaneCluster(h *test.Helper) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": "differentcluster",
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, token)

	return ctx
}

func sampleDataPlaneclusterStatusRequestWithAvailableCapacity() *openapi.DataPlaneClusterUpdateStatusRequest {
	return &openapi.DataPlaneClusterUpdateStatusRequest{
		Conditions: []openapi.DataPlaneClusterUpdateStatusRequestConditions{
			openapi.DataPlaneClusterUpdateStatusRequestConditions{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			Connections:                   &[]int32{1000000}[0],
			DataRetentionSize:             &[]string{"test"}[0],
			Partitions:                    &[]int32{1000000}[0],
		},
		NodeInfo: openapi.DataPlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
		},
		Remaining: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{1000000}[0], // TODO set the values taking the scale-up value if possible or a deterministic way to know we'll pass it
			Partitions:                    &[]int32{1000000}[0],
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			DataRetentionSize:             &[]string{"test"}[0],
		},
		ResizeInfo: openapi.DataPlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &openapi.DataPlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{10000}[0],
				MaxPartitions:                 &[]int32{10000}[0],
				IngressEgressThroughputPerSec: &[]string{"test"}[0],
				DataRetentionSize:             &[]string{"test"}[0],
			},
		},
	}
}

// sampleValidBaseDataPlaneClusterStatusRequest returns a valid
func sampleValidBaseDataPlaneClusterStatusRequest() *openapi.DataPlaneClusterUpdateStatusRequest {
	return &openapi.DataPlaneClusterUpdateStatusRequest{
		Conditions: []openapi.DataPlaneClusterUpdateStatusRequestConditions{
			openapi.DataPlaneClusterUpdateStatusRequestConditions{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{""}[0],
			Connections:                   &[]int32{0}[0],
			DataRetentionSize:             &[]string{""}[0],
			Partitions:                    &[]int32{0}[0],
		},
		NodeInfo: openapi.DataPlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{0}[0],
			Floor:                  &[]int32{0}[0],
			Current:                &[]int32{0}[0],
			CurrentWorkLoadMinimum: &[]int32{0}[0],
		},
		Remaining: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{0}[0],
			Partitions:                    &[]int32{0}[0],
			IngressEgressThroughputPerSec: &[]string{""}[0],
			DataRetentionSize:             &[]string{""}[0],
		},
		ResizeInfo: openapi.DataPlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &openapi.DataPlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{0}[0],
				MaxPartitions:                 &[]int32{0}[0],
				IngressEgressThroughputPerSec: &[]string{""}[0],
				DataRetentionSize:             &[]string{""}[0],
			},
		},
	}
}

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterMetricsBuilder := clustersmgmtv1.NewClusterMetrics()
	clusterMetricsBuilder.Nodes(clusterNodeBuilder)
	clusterBuilder.Metrics(clusterMetricsBuilder)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}
