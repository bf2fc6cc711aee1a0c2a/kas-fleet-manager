package integration

import (
	"net/http"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

// TestApiStatus_Success verifies the object returned by api status endpoint:
// - dinosaur maximum capacity is set to true if user is in deny list
// - dinosaur maximum capacity is set to true if user is not in deny list
// - or service maximum capacity has been reached i.e we've more than maxCapacity of dinosaurs created
// - otherwise it is set to false.
func TestServiceStatus(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// now create another user who has access to the service to perform the remaining two cases
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	serviceStatus, serviceStatusResp, err := client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Dinosaurs.MaxCapacityReached).To(Equal(false))

	// now modify the maximum capacity of dinosaurs and set it to 2 so that when we insert two dinosaurs in the database, we'll reach maximum capacity
	DinosaurConfig(h).DinosaurCapacity.MaxCapacity = 2
	db := test.TestServices.DBFactory.New()
	dinosaurRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	dinosaurCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          "some-random-owner",
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
		},
		{
			MultiAZ:       false,
			Owner:         "some-random-owner-2",
			Region:        dinosaurRegion,
			CloudProvider: dinosaurCloudProvider,
			Name:          "dummy-dinosaur-2",
			Status:        constants2.DinosaurRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}
	serviceStatus, serviceStatusResp, err = client.DefaultApi.GetServiceStatus(ctx)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceStatusResp.StatusCode).To(Equal(http.StatusOK))
	Expect(serviceStatus.Dinosaurs.MaxCapacityReached).To(Equal(true))
}
