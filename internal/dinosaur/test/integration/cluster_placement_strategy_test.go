package integration

import (
	"encoding/json"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/fleetshardsync"

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

	fleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	fleetshardSync := fleetshardSyncBuilder.Build()
	fleetshardSync.Start()
	defer fleetshardSync.Stop()

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
		InstanceType:  types.STANDARD.String(),
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

	standaloneCluster := &api.Cluster{
		ClusterID:             clusterWithDinosaurID,
		Region:                clusterCriteria.Region,
		MultiAZ:               clusterCriteria.MultiAZ,
		CloudProvider:         clusterCriteria.Provider,
		ProviderType:          api.ClusterProviderStandalone,
		IdentityProviderID:    "some-identity-provider-id",
		ClusterDNS:            clusterDns,
		Status:                api.ClusterProvisioning,
		SupportedInstanceType: api.EvalTypeSupport.String(),
	}

	if err := db.Save(standaloneCluster).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	dataplaneClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{
			ClusterId:             "test03",
			DinosaurInstanceLimit:    1,
			Region:                clusterCriteria.Region,
			MultiAZ:               clusterCriteria.MultiAZ,
			CloudProvider:         clusterCriteria.Provider,
			Schedulable:           true,
			SupportedInstanceType: api.AllInstanceTypeSupport.String(),
		},
		// this is a dummy cluster which will be auto created and should not be deleted because it has dinosaur in it
		config.ManualCluster{
			ClusterId:             standaloneCluster.ClusterID,
			DinosaurInstanceLimit:    1,
			Region:                standaloneCluster.Region,
			MultiAZ:               standaloneCluster.MultiAZ,
			CloudProvider:         standaloneCluster.CloudProvider,
			Schedulable:           true,
			ProviderType:          standaloneCluster.ProviderType,
			ClusterDNS:            standaloneCluster.ClusterDNS,
			Status:                standaloneCluster.Status, // standalone cluster needs to start at provisioning state.
			SupportedInstanceType: api.EvalTypeSupport.String(),
		},
	})

	// Ensure both clusters in the config file have been created
	pollErr := common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 2)
	// add identity provider id to avoid configuring the Dinosaur_SRE IDP
	err := db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	Expect(pollErr).NotTo(HaveOccurred())
	Expect(err).NotTo(HaveOccurred())

	// Ensure that cluster dns is populated with given value
	cluster, err := test.TestServices.ClusterService.FindClusterByID(clusterWithDinosaurID)
	Expect(err).NotTo(HaveOccurred())
	Expect(cluster.ClusterDNS).To(Equal(clusterDns))
	Expect(cluster.ProviderType).To(Equal(api.ClusterProviderStandalone))
	Expect(cluster.SupportedInstanceType).To(Equal(api.EvalTypeSupport.String()))

	// Ensure supported instance type is populated
	cluster, err = test.TestServices.ClusterService.FindClusterByID("test03")
	Expect(err).NotTo(HaveOccurred())
	Expect(cluster.SupportedInstanceType).To(Equal(api.AllInstanceTypeSupport.String()))

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
	// add identity provider id for all of the clusters to avoid configuring the DINOSAUR_SRE IDP as it is not needed
	err = db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	Expect(err).NotTo(HaveOccurred())

	// Need to mark the clusters to ClusterWaitingForFleetShardOperator so that fleetshardsync and placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{"test01", "test02", "test03"}, api.ClusterWaitingForFleetShardOperator)
	Expect(updateErr).NotTo(HaveOccurred())
	dinosaurs := []*dbapi.DinosaurRequest{{
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser1",
		Name:          "dummy-dinosaur-1",
		Status:        constants2.DinosaurRequestStatusAccepted.String(),
		InstanceType:  types.STANDARD.String(),
	}, {
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser2",
		Name:          "dummy-dinosaur-2",
		Status:        constants2.DinosaurRequestStatusAccepted.String(),
		InstanceType:  types.EVAL.String(),
	}}

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

func TestClusterPlacementStrategy_CheckInstanceTypePlacement(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// load existing cluster and assign dinosaur to it so that it is not deleted

	db := test.TestServices.DBFactory.New()

	cloudProvider := "aws"
	region := "us-east-1"
	multiAz := true

	//*********************************************************************
	// pre-create clusters
	//*********************************************************************
	clusterDns := "apps.example.com"
	standardInstanceCluster := "cluster-that-supports-standard-instance"
	evalInstanceCluster := "cluster-that-supports-eval-instance"

	availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v0.23.0-0",
			Ready:   true,
			DinosaurVersions: []api.DinosaurVersion{
				api.DinosaurVersion{Version: "2.7.0"},
			},
			DinosaurIBPVersions: []api.DinosaurIBPVersion{
				api.DinosaurIBPVersion{Version: "2.7"},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())

	standaloneClusters := []*api.Cluster{
		{
			ClusterID:                evalInstanceCluster,
			Region:                   region,
			MultiAZ:                  multiAz,
			CloudProvider:            cloudProvider,
			ProviderType:             api.ClusterProviderStandalone,
			IdentityProviderID:       "some-identity-provider-id",
			ClusterDNS:               clusterDns,
			Status:                   api.ClusterReady,
			SupportedInstanceType:    api.EvalTypeSupport.String(),
			AvailableStrimziVersions: availableStrimziVersions,
		},
		{
			ClusterID:                standardInstanceCluster,
			Region:                   region,
			MultiAZ:                  multiAz,
			CloudProvider:            cloudProvider,
			ProviderType:             api.ClusterProviderStandalone,
			IdentityProviderID:       "some-identity-provider-id",
			ClusterDNS:               clusterDns,
			Status:                   api.ClusterReady,
			SupportedInstanceType:    api.StandardTypeSupport.String(),
			AvailableStrimziVersions: availableStrimziVersions,
		},
	}

	if err := db.Create(standaloneClusters).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	// Need to mark the clusters to be ready so that placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{standardInstanceCluster, evalInstanceCluster}, api.ClusterReady)
	Expect(updateErr).NotTo(HaveOccurred())
	dinosaurs := []*dbapi.DinosaurRequest{{
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser1",
		Name:          "dummy-dinosaur-1",
		Status:        constants2.DinosaurRequestStatusAccepted.String(),
		InstanceType:  types.STANDARD.String(),
	}, {
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser2",
		Name:          "dummy-dinosaur-2",
		Status:        constants2.DinosaurRequestStatusAccepted.String(),
		InstanceType:  types.EVAL.String(),
	}}

	if err := db.Create(dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	dbFactory := test.TestServices.DBFactory

	// verify that a standard instance is assigned to a data plane cluster that supports only standard instances
	dinosaurFound, dinosaurErr := common.WaitForDinosaurClusterIDToBeAssigned(dbFactory, "dummy-dinosaur-1")
	Expect(dinosaurErr).NotTo(HaveOccurred())
	Expect(dinosaurFound.ClusterID).To(Equal(standardInstanceCluster))

	// verify that an eval instance is assigned to a data plane cluster that supports only eval instances
	dinosaurFound2, dinosaurErr2 := common.WaitForDinosaurClusterIDToBeAssigned(dbFactory, "dummy-dinosaur-2")
	Expect(dinosaurErr2).NotTo(HaveOccurred())
	Expect(dinosaurFound2.ClusterID).To(Equal(evalInstanceCluster))
}
