package integration

import (
	"context"
	"fmt"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"net/http"
	"testing"
)

const (
	mockResourceNamespace = "my-kafka-namespace"
	mockKafkaClusterName  = "my-cluster"
)

func TestObservatorium_ResourceStateMetric(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()
	h.Env().Config.ObservabilityConfiguration.EnableMock = true
	err := h.Env().LoadClients()
	Expect(err).NotTo(HaveOccurred(), "Error occurred when loading clients: %v", err)

	defer h.Reset()
	service := services.NewObservatoriumService(h.Env().Clients.Observatorium, h.Env().Services.Kafka)
	kafkaState, err := service.GetKafkaState(mockKafkaClusterName, mockResourceNamespace)
	Expect(err).NotTo(HaveOccurred(), "Error getting kafka state:  %v", err)
	Expect(kafkaState.State).NotTo(BeEmpty(), "Should return state")
}

func TestObservatorium_GetMetrics(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	seedKafka, _, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	h.Env().Config.ObservabilityConfiguration.EnableMock = true
	ctx = auth.SetUsernameContext(context.TODO(), account.Username())
	err = h.Env().LoadClients()
	Expect(err).NotTo(HaveOccurred(), "Error occurred when loading clients: %v", err)
	service := services.NewObservatoriumService(h.Env().Clients.Observatorium, h.Env().Services.Kafka)
	metricsList := &observatorium.KafkaMetrics{}
	q := observatorium.RangeQuery{}
	q.FillDefaults()
	_, err = service.GetMetricsByKafkaId(ctx, metricsList, seedKafka.Id, q)
	Expect(err).NotTo(HaveOccurred(), "Error getting kafka metrics:  %v", err)
	Expect(len(*metricsList)).NotTo(Equal(0), "Should return length greater then zero")

}

func TestObservatorium_GetMetricsByKafkaId(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()
	h.Env().Config.ObservabilityConfiguration.EnableMock = true

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	seedKafka, _, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(constants.KafkaRequestStatusAccepted.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account)
	kafka, _, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(kafka.Id).NotTo(BeEmpty())
	h.Env().Config.ObservabilityConfiguration.EnableMock = true
	err = h.Env().LoadClients()
	Expect(err).NotTo(HaveOccurred(), "Error occurred when loading clients: %v", err)
	filters := openapi.GetMetricsByKafkaIdOpts{}

	metric, resp, err := client.DefaultApi.GetMetricsByKafkaId(ctx, kafka.Id, 5, 30, &filters)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get metric data:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(metric.Items)).NotTo(Equal(0))

}
