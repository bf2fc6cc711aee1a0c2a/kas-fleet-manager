package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const gcp = "gcp"
const aws = "aws"
const azure = "azure"

const afEast1Region = "af-east-1"
const usEast1Region = "us-east-1"

var limit = int(1)

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

var mockSupportedInstanceTypes = &config.KafkaSupportedInstanceTypesConfig{
	Configuration: config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			{
				Id:          "standard",
				DisplayName: "Standard",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x1",
						DisplayName:                 "1",
						IngressThroughputPerSec:     "30Mi",
						EgressThroughputPerSec:      "30Mi",
						TotalMaxConnections:         1000,
						MaxDataRetentionSize:        "100Gi",
						MaxPartitions:               1000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 100,
						MaxMessageSize:              "1Mi",
						MinInSyncReplicas:           2,
						ReplicationFactor:           3,
						SupportedAZModes: []string{
							"multi",
						},
						QuotaConsumed:    1,
						QuotaType:        "rhosak",
						CapacityConsumed: 1,
						MaturityStatus:   config.MaturityStatusStable,
					},
				},
			},
			{
				Id:          "developer",
				DisplayName: "Trial",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x1",
						DisplayName:                 "1",
						IngressThroughputPerSec:     "30Mi",
						EgressThroughputPerSec:      "30Mi",
						TotalMaxConnections:         1000,
						MaxDataRetentionSize:        "100Gi",
						MaxPartitions:               1000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 100,
						MaxMessageSize:              "1Mi",
						MinInSyncReplicas:           1,
						ReplicationFactor:           1,
						SupportedAZModes: []string{
							"single",
						},
						LifespanSeconds:  &[]int{172800}[0],
						QuotaConsumed:    1,
						QuotaType:        "rhosak",
						CapacityConsumed: 1,
						MaturityStatus:   config.MaturityStatusTechPreview,
					},
				},
			},
		},
	},
}

func setupOcmServerWithMockRegionsResp() (*httptest.Server, error) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	usEast1 := clustersmgmtv1.NewCloudRegion().
		ID("us-east-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1").
		DisplayName("us east 1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	afSouth1 := clustersmgmtv1.NewCloudRegion().
		ID("af-south-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/af-south-1").
		DisplayName("af-south-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	euWest2 := clustersmgmtv1.NewCloudRegion().
		ID("eu-west-2").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/eu-west-2").
		DisplayName("eu-west-2").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	euCentral1 := clustersmgmtv1.NewCloudRegion().
		ID("eu-central-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/eu-central-1").
		DisplayName("eu-central-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	apSouth1 := clustersmgmtv1.NewCloudRegion().
		ID("ap-south-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/ap-south-1").
		DisplayName("ap-south-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	regions, err := clustersmgmtv1.NewCloudRegionList().Items(usEast1, afSouth1, euWest2, euCentral1, apSouth1).Build()
	if err != nil {
		return nil, err
	}
	ocmServerBuilder.SetCloudRegionsGetResponse(regions, nil)

	awsProvider := mocks.GetMockCloudProviderBuilder(nil)
	gcpProvider := mocks.GetMockCloudProviderBuilder(func(builder *clustersmgmtv1.CloudProviderBuilder) {
		builder.ID("gcp").
			HREF("/api/clusters_mgmt/v1/cloud_providers/gcp").
			Name("gcp").
			DisplayName("gcp")
	})

	providers, err := clustersmgmtv1.NewCloudProviderList().Items(awsProvider, gcpProvider).Build()
	if err != nil {
		return nil, err
	}

	ocmServerBuilder.SetCloudProvidersGetResponse(providers, nil)
	ocmServer := ocmServerBuilder.Build()
	return ocmServer, nil
}

func TestCloudProviderRegions(t *testing.T) {
	g := gomega.NewWithT(t)

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
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			g.Expect(len(regions.RegionList.Items)).NotTo(gomega.Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			g.Expect(regions.ID).NotTo(gomega.Equal(nil))
			g.Expect(id).NotTo(gomega.Equal(nil))
			g.Expect(name).NotTo(gomega.Equal(nil))
			g.Expect(multiAz).NotTo(gomega.Equal(nil))
		}
	}

}

func TestCachedCloudProviderRegions(t *testing.T) {
	g := gomega.NewWithT(t)

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
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			g.Expect(len(regions.RegionList.Items)).NotTo(gomega.Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			g.Expect(regions.ID).NotTo(gomega.Equal(nil))
			g.Expect(id).NotTo(gomega.Equal(nil))
			g.Expect(name).NotTo(gomega.Equal(nil))
			g.Expect(multiAz).NotTo(gomega.Equal(nil))
		}
	}

}

func TestListCloudProviders(t *testing.T) {
	g := gomega.NewWithT(t)

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
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list cloud providers: %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(cloudProviderList.Items).NotTo(gomega.BeEmpty(), "g.Expected cloud providers list")

	// verify that the cloud providers list should contain atleast "gcp" which comes from standalone provider type
	hasGcp := false
	for _, cloudProvider := range cloudProviderList.Items {
		if cloudProvider.Id == gcp {
			hasGcp = true
			break
		}
	}
	g.Expect(hasGcp).To(gomega.BeTrue())
}

func TestListCloudProviderRegions(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer, err := setupOcmServerWithMockRegionsResp()
	if err != nil {
		t.Errorf("Failed to set mock ocm region list response")
	}
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(pc *config.ProviderConfig, kc *config.KafkaConfig, dc *config.DataplaneClusterConfig) {
		pc.ProvidersConfig.SupportedProviders = config.ProviderList{
			{
				Name:    "aws",
				Default: true,
				Regions: config.RegionList{
					{
						Name:    "us-east-1",
						Default: true,
						SupportedInstanceTypes: config.InstanceTypeMap{
							"developer": {
								Limit: &limit,
							},
							"standard": {
								Limit: &limit,
							},
						},
					},
					{
						Name: "af-south-1",
						SupportedInstanceTypes: config.InstanceTypeMap{
							"standard": {},
						},
					},
					{
						Name:                   "eu-central-1",
						SupportedInstanceTypes: config.InstanceTypeMap{},
					},
				},
			},
		}

		kc.SupportedInstanceTypes = mockSupportedInstanceTypes
		dc.DataPlaneClusterScalingType = config.ManualScaling
		dc.ClusterConfig = config.NewClusterConfig(config.ClusterList{
			{
				Name:                  "dummyCluster1",
				ClusterId:             api.NewID(),
				MultiAZ:               true,
				Region:                afEast1Region,
				CloudProvider:         gcp,
				Status:                api.ClusterReady,
				ProviderType:          api.ClusterProviderOCM,
				SupportedInstanceType: "standard,developer",
				KafkaInstanceLimit:    1,
				Schedulable:           true,
			},
			{
				Name:                  "dummyCluster2",
				ClusterId:             api.NewID(),
				MultiAZ:               true,
				Region:                usEast1Region,
				CloudProvider:         aws,
				Status:                api.ClusterReady,
				ProviderType:          api.ClusterProviderOCM,
				SupportedInstanceType: "standard,developer",
				KafkaInstanceLimit:    2,
				Schedulable:           true,
			},
		})

	})
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// all regions available for the 'aws' cloud provider should be returned
	cloudProviderRegionsList, resp1, err := client.DefaultApi.GetCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	if resp1 != nil {
		resp1.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	g.Expect(resp1.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(cloudProviderRegionsList.Items).NotTo(gomega.BeEmpty(), "g.Expected aws cloud provider regions to return a non-empty list")

	// enabled should only be set to true for regions that support at least one instance type in the providers config
	for _, cpr := range cloudProviderRegionsList.Items {
		if cpr.Id == "us-east-1" {
			g.Expect(cpr.Enabled).To(gomega.BeTrue())
			g.Expect(cpr.Capacity).To(gomega.HaveLen(2))

			for _, c := range cpr.Capacity {
				g.Expect(c.AvailableSizes).To(gomega.HaveLen(1))
				g.Expect(c.AvailableSizes[0]).To(gomega.Equal("x1"))
			}
		} else if cpr.Id == "af-south-1" {
			g.Expect(cpr.Enabled).To(gomega.BeTrue())
			g.Expect(cpr.Capacity).To(gomega.HaveLen(1))                   // only supports standard
			g.Expect(cpr.Capacity[0].AvailableSizes).To(gomega.HaveLen(0)) // no cluster has been created in this region so capacity should be empty
		} else {
			g.Expect(cpr.Enabled).To(gomega.BeFalse())
			g.Expect(cpr.Capacity).To(gomega.HaveLen(0))
		}
	}

	// Create a kafka in us-east-1 to use up capacity
	err = test.TestServices.DBFactory.New().Create(&dbapi.KafkaRequest{
		Name:          "dummyKafka",
		Region:        usEast1Region,
		CloudProvider: aws,
		InstanceType:  "standard",
		SizeId:        "x1",
	}).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create dummy kafka")

	// ensure capacity has changed for us-east-1
	cloudProviderRegionsList, resp1, err = client.DefaultApi.GetCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	if resp1 != nil {
		resp1.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	g.Expect(resp1.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(cloudProviderRegionsList.Items).NotTo(gomega.BeEmpty(), "g.Expected aws cloud provider regions to return a non-empty list")

	for _, cpr := range cloudProviderRegionsList.Items {
		if cpr.Id == "us-east-1" {
			g.Expect(cpr.Capacity).To(gomega.HaveLen(2))

			for _, c := range cpr.Capacity {
				if c.InstanceType == "standard" {
					g.Expect(c.AvailableSizes).To(gomega.HaveLen(0)) // dummy Kafka created above has used up all capacity for standard Kafka instances in this region
				} else {
					g.Expect(c.AvailableSizes).To(gomega.HaveLen(1)) // other Kafka instance types should not have been affected
				}
			}
			break
		}
	}

	// all regions available for the 'gcp' cloud provider should be returned
	gcpCloudProviderRegions, gcpResp, gcpErr := client.DefaultApi.GetCloudProviderRegions(ctx, gcp, nil)
	if gcpResp != nil {
		gcpResp.Body.Close()
	}

	g.Expect(gcpErr).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list gcp cloud providers regions:  %v", gcpErr)
	g.Expect(gcpResp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(gcpCloudProviderRegions.Items).NotTo(gomega.BeEmpty(), "g.Expected gcp cloud provider regions to return a non-empty list")

	// all gcp regions returned should have enabled set to false as they do not support any instance types as specified in the providers config
	for _, cpr := range gcpCloudProviderRegions.Items {
		g.Expect(cpr.Enabled).To(gomega.BeFalse())
	}

	//test with wrong provider id
	wrongCloudProviderList, respFromWrongID, errFromWrongId := client.DefaultApi.GetCloudProviderRegions(ctx, "wrong_provider_id", nil)
	if respFromWrongID != nil {
		respFromWrongID.Body.Close()
	}
	g.Expect(errFromWrongId).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", errFromWrongId)
	g.Expect(respFromWrongID.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(wrongCloudProviderList.Items).To(gomega.BeEmpty(), "g.Expected cloud providers regions list empty")
}
