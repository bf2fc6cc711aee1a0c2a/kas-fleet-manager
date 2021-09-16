package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

const gcp = "gcp"
const aws = "aws"
const azure = "azure"

const afEast1Region = "af-east-1"
const usEast1Region = "us-east-1"

var dummyClusters = []*api.Cluster{
	{
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             afEast1Region,
		CloudProvider:      gcp,
		Status:             api.ClusterReady,
		ProviderType:       api.ClusterProviderStandalone,
		IdentityProviderID: "some-identity-provider-id",
	},
	{
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             usEast1Region,
		CloudProvider:      aws,
		Status:             api.ClusterReady,
		ProviderType:       api.ClusterProviderOCM,
		IdentityProviderID: "some-identity-provider-id",
	},
}

func TestCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	cloudProviderRegions, err := test.TestServices.CloudProvidersService.GetCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			Expect(len(regions.RegionList.Items)).NotTo(Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			Expect(regions.ID).NotTo(Equal(nil))
			Expect(id).NotTo(Equal(nil))
			Expect(name).NotTo(Equal(nil))
			Expect(multiAz).NotTo(Equal(nil))
		}
	}

}

func TestCachedCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	cloudProviderRegions, err := test.TestServices.CloudProvidersService.GetCachedCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			Expect(len(regions.RegionList.Items)).NotTo(Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			Expect(regions.ID).NotTo(Equal(nil))
			Expect(id).NotTo(Equal(nil))
			Expect(name).NotTo(Equal(nil))
			Expect(multiAz).NotTo(Equal(nil))
		}
	}

}

func TestListCloudProviders(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	cloudProviderList, resp, err := client.DefaultApi.GetCloudProviders(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).NotTo(BeEmpty(), "Expected cloud providers list")

	// verify that the cloud providers list should contain atleast "gcp" which comes from standalone provider type
	hasGcp := false
	for _, cloudProvider := range cloudProviderList.Items {
		if cloudProvider.Id == gcp {
			hasGcp = true
			break
		}
	}
	Expect(hasGcp).To(BeTrue())
}

func TestListCloudProviderRegions(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	cloudProviderRegionsList, resp1, err := client.DefaultApi.GetCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp1.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderRegionsList.Items).NotTo(BeEmpty(), "Expected cloud provider regions list")

	gcpCloudProviderRegions, gcpResp, gcpErr := client.DefaultApi.GetCloudProviderRegions(ctx, gcp, nil)
	Expect(gcpErr).NotTo(HaveOccurred(), "Error occurred when attempting to list gcp cloud providers regions:  %v", gcpErr)
	Expect(gcpResp.StatusCode).To(Equal(http.StatusOK))
	Expect(gcpCloudProviderRegions.Items).NotTo(BeEmpty(), "Expected gcp cloud provider regions list")

	// verify that region from standalone provider type is there
	gcpHasAfEast1Region := false
	for _, region := range gcpCloudProviderRegions.Items {
		if region.Id == afEast1Region {
			gcpHasAfEast1Region = true
			break
		}
	}

	Expect(gcpHasAfEast1Region).To(BeTrue())

	//test with wrong provider id
	wrongCloudProviderList, respFromWrongID, errFromWrongId := client.DefaultApi.GetCloudProviderRegions(ctx, "wrong_provider_id", nil)
	Expect(errFromWrongId).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", errFromWrongId)
	Expect(respFromWrongID.StatusCode).To(Equal(http.StatusOK))
	Expect(wrongCloudProviderList.Items).To(BeEmpty(), "Expected cloud providers regions list empty")

}
