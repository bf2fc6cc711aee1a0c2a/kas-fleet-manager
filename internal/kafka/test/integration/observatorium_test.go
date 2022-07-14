package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

const (
	mockResourceNamespace = "my-kafka-namespace"
	mockKafkaClusterName  = "my-cluster"
)

func TestObservatorium_ResourceStateMetric(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	service := services.NewObservatoriumService(test.TestServices.ObservatoriumClient, test.TestServices.KafkaService)
	kafkaState, err := service.GetKafkaState(mockKafkaClusterName, mockResourceNamespace)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error getting kafka state:  %v", err)
	g.Expect(kafkaState.State).NotTo(gomega.BeEmpty(), "Should return state")
}

func TestObservatorium_GetMetrics(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	seedKafka, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	service := services.NewObservatoriumService(test.TestServices.ObservatoriumClient, test.TestServices.KafkaService)
	metricsList := &observatorium.KafkaMetrics{}
	q := observatorium.MetricsReqParams{}
	q.ResultType = observatorium.RangeQuery
	q.FillDefaults()

	token, err := h.AuthHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Errorf("failed to create token: %s", err.Error())
	}
	context := auth.SetTokenInContext(context.Background(), token)
	_, err = service.GetMetricsByKafkaId(context, metricsList, seedKafka.Id, q)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error getting kafka metrics:  %v", err)
	g.Expect(len(*metricsList)).NotTo(gomega.Equal(0), "Should return length greater then zero")

	// Delete created kafkas
	deleteTestKafka(t, h, ctx, client, seedKafka.Id)
}

func TestObservatorium_GetMetricsByQueryRange(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	seedKafka, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	foundKafka, _ := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, seedKafka.Id, constants2.KafkaRequestStatusReady)

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(kafka.Href).To(gomega.Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	g.Expect(kafka.Region).To(gomega.Equal(mocks.MockCluster.Region().ID()))
	g.Expect(kafka.CloudProvider).To(gomega.Equal(mocks.MockCluster.CloudProvider().ID()))
	g.Expect(kafka.Name).To(gomega.Equal(mockKafkaName))
	g.Expect(kafka.Status).To(gomega.Equal(constants2.KafkaRequestStatusReady.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	kafka, resp, _ = client.DefaultApi.GetKafkaById(context, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when loading clients: %v", err)
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty())
	filters := public.GetMetricsByRangeQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByRangeQuery(context, kafka.Id, 5, 30, &filters)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(metrics.Items)).NotTo(gomega.Equal(0))

	firstMetric := metrics.Items[0]
	g.Expect(firstMetric.Values[0].Timestamp).NotTo(gomega.BeNil())
	g.Expect(firstMetric.Values[0].Value).NotTo(gomega.BeNil())

	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}
func TestObservatorium_GetMetricsByQueryInstant(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	seedKafka, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	foundKafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, seedKafka.Id, constants2.KafkaRequestStatusReady)
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for kafka to be ready")
	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty(), "g.Expected ID assigned on creation")
	g.Expect(kafka.Kind).To(gomega.Equal(presenters.KindKafka))
	g.Expect(kafka.Href).To(gomega.Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	g.Expect(kafka.Region).To(gomega.Equal(mocks.MockCluster.Region().ID()))
	g.Expect(kafka.CloudProvider).To(gomega.Equal(mocks.MockCluster.CloudProvider().ID()))
	g.Expect(kafka.Name).To(gomega.Equal(mockKafkaName))
	g.Expect(kafka.Status).To(gomega.Equal(constants2.KafkaRequestStatusReady.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	kafka, resp, _ = client.DefaultApi.GetKafkaById(context, seedKafka.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(kafka.Id).NotTo(gomega.BeEmpty())

	filters := public.GetMetricsByInstantQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByInstantQuery(context, kafka.Id, &filters)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(len(metrics.Items)).NotTo(gomega.Equal(0))

	firstMetric := metrics.Items[0]
	g.Expect(firstMetric.Timestamp).NotTo(gomega.BeNil())
	g.Expect(firstMetric.Value).NotTo(gomega.BeNil())

	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}
