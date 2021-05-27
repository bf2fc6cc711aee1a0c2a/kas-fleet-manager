package integration

import (
	"net/http"
	"testing"

	api "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

// TestApiStatus_Success verifies the object returned by api status endpoint:
// - kafka maximum capacity is set to true if user is in deny list
// - kafka maximum capacity is set to true if user is not in deny list and is not allowed to access the service via the allow list
// - or service maximum capacity has been reached i.e we've more than maxCapacity of kafkas created
// - otherwise it is set to false.
func TestServiceStatus(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// these values are taken from config/deny-list-configuration.yaml
	deniedUser := "denied-test-user1@example.com"
	// this value is taken from config/allow-list-configuration.yaml
	orgId := "13640203"
	deniedAccount := h.NewAccount(deniedUser, deniedUser, deniedUser, orgId)
	deniedCtx := h.NewAuthenticatedContext(deniedAccount, nil)

	// since this user is in the deny list and not authorized to access the service, kafkas maximum capacity should be set to true
	serviceStatus, serviceStatusResp, err := client.DefaultApi.GetServiceStatus(deniedCtx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Kafkas.MaxCapacityReached).To(Equal(true))

	// create another user not in deny list and allow list
	// since this user is in not in the allow list and not authorized to access the service, kafkas maximum capacity should be set to true
	orgId = "some-org-id"
	notAllowedUser := "user@not-allowed.com"
	notAllowAccount := h.NewAccount(notAllowedUser, notAllowedUser, notAllowedUser, orgId)
	notAllowedCtx := h.NewAuthenticatedContext(notAllowAccount, nil)
	serviceStatus, serviceStatusResp, err = client.DefaultApi.GetServiceStatus(notAllowedCtx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Kafkas.MaxCapacityReached).To(Equal(true))

	// now create another user who has access to the service to perform the remaining two cases
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	serviceStatus, serviceStatusResp, err = client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Kafkas.MaxCapacityReached).To(Equal(false))

	// now modify the maximum capacity of kafkas and set it to 2 so that when we insert two kafkas in the database, we'll reach maximum capacity
	h.Env().Config.Kafka.KafkaCapacity.MaxCapacity = 2
	db := h.Env().DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	kafkas := []*api.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          "some-random-owner",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:       false,
			Owner:         "some-random-owner-2",
			Region:        kafkaRegion,
			CloudProvider: kafkaCloudProvider,
			Name:          "dummy-kafka-2",
			Status:        constants.KafkaRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}
	serviceStatus, serviceStatusResp, err = client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Kafkas.MaxCapacityReached).To(Equal(true))
}
