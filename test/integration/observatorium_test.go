package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks/kasfleetshardsync"
	. "github.com/onsi/gomega"
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

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
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

	service := services.NewObservatoriumService(h.Env().Clients.Observatorium, h.Env().Services.Kafka)
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
	Expect(err).NotTo(HaveOccurred(), "Error getting kafka metrics:  %v", err)
	Expect(len(*metricsList)).NotTo(Equal(0), "Should return length greater then zero")

	// Delete created kafkas
	deleteTestKafka(t, h, ctx, client, seedKafka.Id)
}

func TestObservatorium_GetMetricsByQueryRange(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
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

	var foundKafka openapi.KafkaRequest
	_ = utils.NewPollerBuilder().
		OutputFunction(t.Logf).
		IntervalAndTimeout(kafkaCheckInterval, kafkaReadyTimeout).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
			if err != nil {
				return true, err
			}
			return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
		}).
		RetryLogMessage(fmt.Sprintf("Waiting for kafka (%s) to be ready", seedKafka.Id)).
		Build().Poll()

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	kafka, _, _ = client.DefaultApi.GetKafkaById(context, seedKafka.Id)
	Expect(kafka.Id).NotTo(BeEmpty())
	Expect(err).NotTo(HaveOccurred(), "Error occurred when loading clients: %v", err)
	filters := openapi.GetMetricsByRangeQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByRangeQuery(context, kafka.Id, 5, 30, &filters)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(metrics.Items)).NotTo(Equal(0))
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}
func TestObservatorium_GetMetricsByQueryInstant(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
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

	var foundKafka openapi.KafkaRequest

	_ = utils.NewPollerBuilder().
		OutputFunction(t.Logf).
		IntervalAndTimeout(kafkaCheckInterval, kafkaReadyTimeout).
		OnRetry(func(attempt int, maxRetries int) (bool, error) {
			foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
			if err != nil {
				return true, err
			}
			return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
		}).
		RetryLogMessage(fmt.Sprintf("Waiting for kafka (%s) to be ready", seedKafka.Id)).
		Build().Poll()

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	acc := h.NewRandAccount()
	context := h.NewAuthenticatedContext(acc, nil)
	kafka, _, _ = client.DefaultApi.GetKafkaById(context, seedKafka.Id)
	Expect(kafka.Id).NotTo(BeEmpty())

	filters := openapi.GetMetricsByInstantQueryOpts{}
	metrics, resp, err := client.DefaultApi.GetMetricsByInstantQuery(context, kafka.Id, &filters)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get metrics data:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(metrics.Items)).NotTo(Equal(0))
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}
