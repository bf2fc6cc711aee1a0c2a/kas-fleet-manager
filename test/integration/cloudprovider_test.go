package integration

import (
	"testing"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	cloudProviderRegions, err := h.Env().Services.CloudProviders.GetCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		Expect(regions.RegionList.Len()).NotTo(Equal(0))
		regions.RegionList.Each(func(item *clustersmgmtv1.CloudRegion) bool {
			href := item.HREF()
			id := item.ID()
			name := item.DisplayName()
			multiAz := item.SupportsMultiAZ()

			Expect(regions.ID).NotTo(Equal(nil))
			Expect(href).NotTo(Equal(nil))
			Expect(id).NotTo(Equal(nil))
			Expect(name).NotTo(Equal(nil))
			Expect(href).NotTo(Equal(nil))
			Expect(multiAz).NotTo(Equal(nil))
			return true
		})

	}

}
