package integration

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"net/http"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/fleetshardsync"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

const (
	mockResourceNamespace = "my-dinosaur-namespace"
	mockDinosaurClusterName  = "my-cluster"
)

func TestObservatorium_ResourceStateMetric(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	service := services.NewObservatoriumService(test.TestServices.ObservatoriumClient, test.TestServices.DinosaurService)
	dinosaurState, err := service.GetDinosaurState(mockDinosaurClusterName, mockResourceNamespace)
	Expect(err).NotTo(HaveOccurred(), "Error getting dinosaur state:  %v", err)
	Expect(dinosaurState.State).NotTo(BeEmpty(), "Should return state")
}

func TestObservatorium_GetMetrics(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 2)})
	})
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	seedDinosaur, _, err := client.DefaultApi.CreateDinosaur(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded dinosaur request: %s", err.Error())
	}

	service := services.NewObservatoriumService(test.TestServices.ObservatoriumClient, test.TestServices.DinosaurService)
	metricsList := &observatorium.DinosaurMetrics{}
	q := observatorium.MetricsReqParams{}
	q.ResultType = observatorium.RangeQuery
	q.FillDefaults()

	token, err := h.AuthHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Errorf("failed to create token: %s", err.Error())
	}
	context := auth.SetTokenInContext(context.Background(), token)
	_, err = service.GetMetricsByDinosaurId(context, metricsList, seedDinosaur.Id, q)
	Expect(err).NotTo(HaveOccurred(), "Error getting dinosaur metrics:  %v", err)
	Expect(len(*metricsList)).NotTo(Equal(0), "Should return length greater then zero")

	// Delete created dinosaurs
	deleteTestDinosaur(t, h, ctx, client, seedDinosaur.Id)
}

func TestObservatorium_GetMetricsByQueryRange(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 2)})
	})
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	seedDinosaur, _, err := client.DefaultApi.CreateDinosaur(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded dinosaur request: %s", err.Error())
	}

	foundDinosaur, _ := common.WaitForDinosaurToReachStatus(ctx, test.TestServices.DBFactory, client, seedDinosaur.Id, constants2.DinosaurRequestStatusReady)

	// 200 OK
	dinosaur, resp, err := client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get dinosaur request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/dinosaurs_mgmt/v1/dinosaurs/%s", dinosaur.Id)))
	Expect(dinosaur.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(dinosaur.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(dinosaur.Name).To(Equal(mockDinosaurName))
	Expect(dinosaur.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))

	// 404 Not Found
	dinosaur, resp, _ = client.DefaultApi.GetDinosaurById(ctx, fmt.Sprintf("not-%s", seedDinosaur.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Dinosaur cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	dinosaur, _, _ = client.DefaultApi.GetDinosaurById(context, seedDinosaur.Id)
	Expect(dinosaur.Id).NotTo(BeEmpty())
	Expect(err).NotTo(HaveOccurred(), "Error occurred when loading clients: %v", err)
	filters := public.GetMetricsByRangeQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByRangeQuery(context, dinosaur.Id, 5, 30, &filters)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(metrics.Items)).NotTo(Equal(0))

	firstMetric := metrics.Items[0]
	Expect(firstMetric.Values[0].Timestamp).NotTo(BeNil())
	Expect(firstMetric.Values[0].Value).NotTo(BeNil())

	deleteTestDinosaur(t, h, ctx, client, foundDinosaur.Id)
}
func TestObservatorium_GetMetricsByQueryInstant(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 2)})
	})
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	seedDinosaur, _, err := client.DefaultApi.CreateDinosaur(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded dinosaur request: %s", err.Error())
	}

	foundDinosaur, err := common.WaitForDinosaurToReachStatus(ctx, test.TestServices.DBFactory, client, seedDinosaur.Id, constants2.DinosaurRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for dinosaur to be ready")
	// 200 OK
	dinosaur, resp, err := client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get dinosaur request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/dinosaurs_mgmt/v1/dinosaurs/%s", dinosaur.Id)))
	Expect(dinosaur.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(dinosaur.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(dinosaur.Name).To(Equal(mockDinosaurName))
	Expect(dinosaur.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))

	// 404 Not Found
	dinosaur, resp, _ = client.DefaultApi.GetDinosaurById(ctx, fmt.Sprintf("not-%s", seedDinosaur.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Dinosaur cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	dinosaur, _, _ = client.DefaultApi.GetDinosaurById(context, seedDinosaur.Id)
	Expect(dinosaur.Id).NotTo(BeEmpty())

	filters := public.GetMetricsByInstantQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByInstantQuery(context, dinosaur.Id, &filters)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(metrics.Items)).NotTo(Equal(0))

	firstMetric := metrics.Items[0]
	Expect(firstMetric.Timestamp).NotTo(BeNil())
	Expect(firstMetric.Value).NotTo(BeNil())

	deleteTestDinosaur(t, h, ctx, client, foundDinosaur.Id)
}
