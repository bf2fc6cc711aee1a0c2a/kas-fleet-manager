package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

func TestClusterPlacementStrategy_ManualType(t *testing.T) {

	// Start with no cluster config and manual scaling.
	configtHook := func(h *test.Helper) {
		clusterConfig := h.Env.Config.OSDClusterConfig
		clusterConfig.DataPlaneClusterScalingType = config.ManualScaling
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegrationWithHooks(t, ocmServer, configtHook)
	defer teardown()

	if h.Env.Config.OCM.MockMode != config.MockModeEmulateServer {
		t.SkipNow()
	}

	// load existing cluster and assign kafka to it so that it is not deleted

	db := h.DBFactory.New()
	clusterWithKafkaID := "cluster-id-that-should-not-be-deleted"

	kafka := api.KafkaRequest{
		ClusterID:     clusterWithKafkaID,
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Name:          "dummy-kafka",
		Status:        constants.KafkaRequestStatusReady.String(),
	}

	if err := db.Save(&kafka).Error; err != nil {
		t.Error("failed to create a dummy kafka request")
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
	h.Env.Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{
			ClusterId:          "test03",
			KafkaInstanceLimit: 1,
			Region:             clusterCriteria.Region,
			MultiAZ:            clusterCriteria.MultiAZ,
			CloudProvider:      clusterCriteria.Provider,
			Schedulable:        true,
		},
		// this is a dummy cluster which will be auto created and should not be deleted because it has kafka in it
		config.ManualCluster{
			ClusterId:          clusterWithKafkaID,
			KafkaInstanceLimit: 1,
			Region:             clusterCriteria.Region,
			MultiAZ:            clusterCriteria.MultiAZ,
			CloudProvider:      clusterCriteria.Provider,
			Schedulable:        true,
			ProviderType:       api.ClusterProviderStandalone,
			ClusterDNS:         clusterDns,
			Status:             api.ClusterProvisioning, // standalone cluster needs to start at provisioning state.
		},
	})

	var clusterService services.ClusterService
	h.Env.MustResolveAll(&clusterService)

	// Ensure both clusters in the config file have been created
	pollErr := common.WaitForClustersMatchCriteriaToBeGivenCount(h.DBFactory, &clusterService, &clusterCriteria, 2)
	Expect(pollErr).NotTo(HaveOccurred())

	// Ensure that cluster dns is populated with given value
	cluster, err := clusterService.FindClusterByID(clusterWithKafkaID)
	Expect(err).NotTo(HaveOccurred())
	Expect(cluster.ClusterDNS).To(Equal(clusterDns))
	Expect(cluster.ProviderType).To(Equal(api.ClusterProviderStandalone))

	//*********************************************************************
	//data plane cluster config - with new clusters
	//*********************************************************************
	h.Env.Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test02", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
	})

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(h.DBFactory, &clusterService, &clusterCriteria, 4)
	Expect(pollErr).NotTo(HaveOccurred())

	// Now delete the kafka from the original cluster to check for placement strategy and wait for cluster deletion
	if err := db.Delete(&kafka).Error; err != nil {
		t.Fatal("failed to delete a dummy kafka request")
	}

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(h.DBFactory, &clusterService, &clusterCriteria, 3)
	Expect(pollErr).NotTo(HaveOccurred())

	//*********************************************************************
	//Test kafka instance creation and OSD cluster placement
	//*********************************************************************
	// Need to mark the clusters to be ready so that placement can actually happen
	updateErr := clusterService.UpdateMultiClusterStatus([]string{"test01", "test02", "test03"}, api.ClusterReady)
	Expect(updateErr).NotTo(HaveOccurred())
	kafkas := []*api.KafkaRequest{
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser1",
			Name:          "dummy-kafka-1",
			Status:        constants.KafkaRequestStatusAccepted.String(),
		},
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser2",
			Name:          "dummy-kafka-2",
			Status:        constants.KafkaRequestStatusAccepted.String(),
		},
	}

	var kafkaSrv services.KafkaService
	h.Env.MustResolveAll(&kafkaSrv)

	errK := kafkaSrv.RegisterKafkaJob(kafkas[0])
	if errK != nil {
		Expect(errK).NotTo(HaveOccurred())
		return
	}

	dbFactory := h.DBFactory
	kafkaFound, kafkaErr := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-1")
	Expect(kafkaErr).NotTo(HaveOccurred())
	Expect(kafkaFound.ClusterID).To(Equal("test03"))

	errK2 := kafkaSrv.RegisterKafkaJob(kafkas[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	kafkaFound2, kafkaErr2 := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-2")
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(kafkaFound2.ClusterID).To(Equal("test02"))
}
