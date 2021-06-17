package integration

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"net/http"
	"testing"

	api "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

func TestCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	var cloudProviders services.CloudProvidersService
	h.Env.MustResolveAll(&cloudProviders)

	cloudProviderRegions, err := cloudProviders.GetCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == "aws" || regions.ID == "azure" || regions.ID == "gcp" {
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
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	var cloudProviders services.CloudProvidersService
	h.Env.MustResolveAll(&cloudProviders)

	cloudProviderRegions, err := cloudProviders.GetCachedCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == "aws" || regions.ID == "azure" || regions.ID == "gcp" {
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	gcp := "gcp"
	// Create two clusters each with different provider type
	dummyClusters := []*api.Cluster{
		{
			ClusterID:     api.NewID(),
			MultiAZ:       true,
			Region:        "af-east-1",
			CloudProvider: gcp,
			Status:        api.ClusterReady,
			ProviderType:  api.ClusterProviderStandalone,
		},
		{
			ClusterID:     api.NewID(),
			MultiAZ:       true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Status:        api.ClusterReady,
			ProviderType:  api.ClusterProviderOCM,
		},
	}

	if err := h.DBFactory.New().Create(dummyClusters).Error; err != nil {
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	cloudProviderList, resp, err := client.DefaultApi.GetCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).NotTo(BeEmpty(), "Expected cloud provider regions list")

	//test with wrong provider id
	cloudProviderList, resp, err = client.DefaultApi.GetCloudProviderRegions(ctx, "wrong_provider_id", nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).To(BeEmpty(), "Expected cloud providers regions list empty")

}
