package integration

import (
	"fmt"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"testing"
	"time"
)

const (
	mockKafkaName  = "test"
	mockKafkaOwner = "owner"
)

// TestKafkaCreate_Success validates the happy path of the kafka post endpoint:
// - kafka successfully registered with database
// - kafka worker picks up on creation job
// - cluster is found for kafka
// - kafka is assigned cluster
// - kafka becomes ready once syncset is created
func TestKafkaCreate_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setup cluster if one doesn't already exist for the tested region
	foundCluster, svcErr := h.Env().Services.Cluster.FindCluster(services.FindClusterCriteria{
		Region:   mocks.MockCluster.Region().Name(),
		Provider: mocks.MockCluster.CloudProvider().Name(),
	})
	if svcErr != nil {
		t.Fatal(svcErr)
	}
	if foundCluster == nil {
		_, svcErr = h.Env().Services.Cluster.Create(&api.Cluster{
			CloudProvider: mocks.MockCloudProvider.ID(),
			ClusterID:     mocks.MockCluster.ID(),
			ExternalID:    mocks.MockCluster.ExternalID(),
			Region:        mocks.MockCluster.Region().ID(),
		})
		if svcErr != nil {
			t.Fatal(svcErr)
		}
	}

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	// POST responses per openapi spec: 201, 409, 500
	k := openapi.KafkaRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	// kafka successfully registered with database
	kafka, resp, err := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, true, k)
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	var foundKafka openapi.KafkaRequest
	err = wait.PollImmediate(time.Second*30, time.Minute*5, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.ApiManagedServicesApiV1KafkasIdGet(ctx, kafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == services.KafkaRequestStatusComplete.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become complete: %v", err)
	// final check on the status
	Expect(foundKafka.Status).To(Equal(services.KafkaRequestStatusComplete.String()))
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
}

// TestKafkaPost_Validations tests the API validations performed by the kafka creation endpoint
//
// these could also be unit tests
func TestKafkaPost_Validations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	tests := []struct {
		name     string
		body     openapi.KafkaRequest
		wantCode int
	}{
		{
			name: "HTTP 400 when region not provided",
			body: openapi.KafkaRequest{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not provided",
			body: openapi.KafkaRequest{
				MultiAz: mocks.MockCluster.MultiAZ(),
				Region:  mocks.MockCluster.Region().ID(),
				Owner:   mockKafkaOwner,
				Name:    mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: openapi.KafkaRequest{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when id is provided",
			body: openapi.KafkaRequest{
				Id:            mocks.MockCluster.ID(),
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			_, resp, _ := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, true, tt.body)
			Expect(resp.StatusCode).To(Equal(tt.wantCode))
		})
	}
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
	k := openapi.KafkaRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	seedKafka, _, err := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, true, k)
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
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(services.KafkaRequestStatusAccepted.String()))

	// 404 Not Found
	kafka, resp, err = client.DefaultApi.ApiManagedServicesApiV1KafkasIdGet(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}
