package integration

import (
	"net/http"
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
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == "aws" || regions.ID == "azure" || regions.ID == "gcp" {
			Expect(regions.RegionList.Len()).NotTo(Equal(0))
		}
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

func TestListCloudProviders(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	cloudProviderList, resp, err := client.DefaultApi.ListCloudProviders(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).NotTo(BeEmpty(), "Expected cloud providers list")

}
func TestListCloudProviderRegions(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	cloudProviderList, resp, err := client.DefaultApi.ListCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).NotTo(BeEmpty(), "Expected cloud provider regions list")

	//test with wrong provider id
	cloudProviderList, resp, err = client.DefaultApi.ListCloudProviderRegions(ctx, "wrong_provider_id", nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).To(BeEmpty(), "Expected cloud providers regions list empty")

}
