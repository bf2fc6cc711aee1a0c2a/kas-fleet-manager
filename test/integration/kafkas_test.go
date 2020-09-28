package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

const (
	mockKafkaName = "test"
)

// TestKafkaPost tests the POST /kafkas endpoint
// This test should act as a "golden" integration test, it should be well documented and act as a reference for future
// integration tests.
func TestKafkaPost(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	// POST responses per openapi spec: 201, 409, 500
	k := openapi.KafkaRequest{
		Region:        clusterservicetest.MockClusterRegion,
		CloudProvider: clusterservicetest.MockClusterCloudProvider,
		ClusterID:     clusterservicetest.MockClusterID,
		Name:          mockKafkaName,
	}

	// 202 Accepted
	kafka, resp, err := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, k)
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	// 400 bad request. posting junk json is one way to trigger 400.
	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/kafkas"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client := test.RegisterIntegration(t, ocmServer)
	defer h.StopServer()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	k := openapi.KafkaRequest{
		Region:        clusterservicetest.MockClusterRegion,
		CloudProvider: clusterservicetest.MockClusterCloudProvider,
		ClusterID:     clusterservicetest.MockClusterID,
		Name:          mockKafkaName,
	}

	seedKafka, _, err := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, k)
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	// 200 OK
	kafka, resp, err := client.DefaultApi.ApiManagedServicesApiV1KafkasIdGet(ctx, seedKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(clusterservicetest.MockClusterRegion))
	Expect(kafka.CloudProvider).To(Equal(clusterservicetest.MockClusterCloudProvider))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(services.KafkaRequestStatusAccepted.String()))

	// 404 Not Found
	kafka, resp, err = client.DefaultApi.ApiManagedServicesApiV1KafkasIdGet(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}
