package integration

import (
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

func TestClusterPlacementStrategy_ManualType(t *testing.T) {

	// Start with no cluster config and manual scaling.
	configHook := func(clusterConfig *config.DataplaneClusterConfig) {
		clusterConfig.DataPlaneClusterScalingType = config.ManualScaling
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, configHook)
	defer teardown()

	ocmConfig := test.TestServices.OCMConfig

	if ocmConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// load existing cluster and assign dinosaur to it so that it is not deleted

	db := test.TestServices.DBFactory.New()
	clusterWithDinosaurID := "cluster-id-that-should-not-be-deleted"

	dinosaur := dbapi.DinosaurRequest{
		ClusterID:     clusterWithDinosaurID,
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Name:          "dummy-dinosaur",
		Status:        constants2.DinosaurRequestStatusReady.String(),
	}

	if err := db.Save(&dinosaur).Error; err != nil {
		t.Error("failed to create a dummy dinosaur request")
		return
	}

	clusterCriteria := services.FindClusterCriteria{
		Provider: "aws",
		Region:   "us-east-1",
		MultiAZ:  true,
	}

	//*********************************************************************
	// pre-create clusters
	//*********************************************************************
	clusterDns := "apps.example.com"

	dataplaneClusterConfig := DataplaneClusterConfig(h)

	dataplaneClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{
			ClusterId:             "test03",
			DinosaurInstanceLimit: 1,
			Region:                clusterCriteria.Region,
			MultiAZ:               clusterCriteria.MultiAZ,
			CloudProvider:         clusterCriteria.Provider,
			Schedulable:           true,
		},
		// this is a dummy cluster which will be auto created and should not be deleted because it has dinosaur in it
		config.ManualCluster{
			ClusterId:             clusterWithDinosaurID,
			DinosaurInstanceLimit: 1,
			Region:                clusterCriteria.Region,
			MultiAZ:               clusterCriteria.MultiAZ,
			CloudProvider:         clusterCriteria.Provider,
			Schedulable:           true,
			ProviderType:          api.ClusterProviderStandalone,
			ClusterDNS:            clusterDns,
			Status:                api.ClusterProvisioning, // standalone cluster needs to start at provisioning state.
		},
	})

	// Ensure both clusters in the config file have been created
	pollErr := common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 2)
	Expect(pollErr).NotTo(HaveOccurred())

	// Ensure that cluster dns is populated with given value
	cluster, err := test.TestServices.ClusterService.FindClusterByID(clusterWithDinosaurID)
	Expect(err).NotTo(HaveOccurred())
	Expect(cluster.ClusterDNS).To(Equal(clusterDns))
	Expect(cluster.ProviderType).To(Equal(api.ClusterProviderStandalone))

	//*********************************************************************
	//data plane cluster config - with new clusters
	//*********************************************************************
	dataplaneClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", DinosaurInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test01", DinosaurInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test02", DinosaurInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
	})

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 4)
	Expect(pollErr).NotTo(HaveOccurred())

	// Now delete the dinosaur from the original cluster to check for placement strategy and wait for cluster deletion
	if err := db.Delete(&dinosaur).Error; err != nil {
		t.Fatal("failed to delete a dummy dinosaur request")
	}

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 3)
	Expect(pollErr).NotTo(HaveOccurred())

	//*********************************************************************
	//Test dinosaur instance creation and OSD cluster placement
	//*********************************************************************
	// Need to mark the clusters to be ready so that placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{"test01", "test02", "test03"}, api.ClusterReady)
	Expect(updateErr).NotTo(HaveOccurred())
	dinosaurs := []*dbapi.DinosaurRequest{
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser1",
			Name:          "dummy-dinosaur-1",
			Status:        constants2.DinosaurRequestStatusAccepted.String(),
		},
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser2",
			Name:          "dummy-dinosaur-2",
			Status:        constants2.DinosaurRequestStatusAccepted.String(),
		},
	}

	errK := test.TestServices.DinosaurService.RegisterDinosaurJob(dinosaurs[0])
	if errK != nil {
		Expect(errK).NotTo(HaveOccurred())
		return
	}

	dbFactory := test.TestServices.DBFactory
	dinosaurFound, dinosaurErr := common.WaitForDinosaurClusterIDToBeAssigned(dbFactory, "dummy-dinosaur-1")
	Expect(dinosaurErr).NotTo(HaveOccurred())
	Expect(dinosaurFound.ClusterID).To(Equal("test03"))

	errK2 := test.TestServices.DinosaurService.RegisterDinosaurJob(dinosaurs[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	dinosaurFound2, dinosaurErr2 := common.WaitForDinosaurClusterIDToBeAssigned(dbFactory, "dummy-dinosaur-2")
	Expect(dinosaurErr2).NotTo(HaveOccurred())
	Expect(dinosaurFound2.ClusterID).To(Equal("test02"))
}
