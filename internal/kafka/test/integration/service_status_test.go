package integration

import (
	"net/http"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

// TestApiStatus_Success verifies the object returned by api status endpoint:
// - kafka maximum capacity is set to true if user is in deny list
// - kafka maximum capacity is set to true if user is not in deny list
// - or service maximum capacity has been reached i.e we've more than maxCapacity of kafkas created
// - otherwise it is set to false.
func TestServiceStatus(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// now create another user who has access to the service to perform the remaining two cases
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	serviceStatus, serviceStatusResp, err := client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Dinosaurs.MaxCapacityReached).To(Equal(false))

	// now modify the maximum capacity of kafkas and set it to 2 so that when we insert two kafkas in the database, we'll reach maximum capacity
	KafkaConfig(h).KafkaCapacity.MaxCapacity = 2
	db := test.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	kafkas := []*dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          "some-random-owner",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:       false,
			Owner:         "some-random-owner-2",
			Region:        kafkaRegion,
			CloudProvider: kafkaCloudProvider,
			Name:          "dummy-kafka-2",
			Status:        constants2.KafkaRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}
	serviceStatus, serviceStatusResp, err = client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Dinosaurs.MaxCapacityReached).To(Equal(true))
}
