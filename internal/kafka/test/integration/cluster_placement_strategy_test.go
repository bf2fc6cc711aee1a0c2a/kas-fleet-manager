package integration

import (
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
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
	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, configHook)
	defer teardown()

	ocmConfig := test.TestServices.OCMConfig

	if ocmConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// load existing cluster and assign kafka to it so that it is not deleted

	db := test.TestServices.DBFactory.New()
	clusterWithKafkaID := "cluster-id-that-should-not-be-deleted"

	kafka := dbapi.KafkaRequest{
		ClusterID:     clusterWithKafkaID,
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Name:          "dummy-kafka",
		Status:        constants2.KafkaRequestStatusReady.String(),
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

	dataplaneClusterConfig := DataplaneClusterConfig(h)

	standaloneCluster := &api.Cluster{
		ClusterID:          clusterWithKafkaID,
		Region:             clusterCriteria.Region,
		MultiAZ:            clusterCriteria.MultiAZ,
		CloudProvider:      clusterCriteria.Provider,
		ProviderType:       api.ClusterProviderStandalone,
		IdentityProviderID: "some-identity-provider-id",
		ClusterDNS:         clusterDns,
		Status:             api.ClusterProvisioning,
	}

	if err := db.Save(standaloneCluster).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	dataplaneClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
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
			ClusterId:          standaloneCluster.ClusterID,
			KafkaInstanceLimit: 1,
			Region:             standaloneCluster.Region,
			MultiAZ:            standaloneCluster.MultiAZ,
			CloudProvider:      standaloneCluster.CloudProvider,
			Schedulable:        true,
			ProviderType:       standaloneCluster.ProviderType,
			ClusterDNS:         standaloneCluster.ClusterDNS,
			Status:             standaloneCluster.Status, // standalone cluster needs to start at provisioning state.
		},
	})

	// Ensure both clusters in the config file have been created
	pollErr := common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 2)
	// add identity provider id to avoid configuring the Kafka_SRE IDP
	err := db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	Expect(pollErr).NotTo(HaveOccurred())
	Expect(err).NotTo(HaveOccurred())

	// Ensure that cluster dns is populated with given value
	cluster, err := test.TestServices.ClusterService.FindClusterByID(clusterWithKafkaID)
	Expect(err).NotTo(HaveOccurred())
	Expect(cluster.ClusterDNS).To(Equal(clusterDns))
	Expect(cluster.ProviderType).To(Equal(api.ClusterProviderStandalone))

	//*********************************************************************
	//data plane cluster config - with new clusters
	//*********************************************************************
	dataplaneClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test02", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
	})

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 4)
	Expect(pollErr).NotTo(HaveOccurred())

	// Now delete the kafka from the original cluster to check for placement strategy and wait for cluster deletion
	if err := db.Delete(&kafka).Error; err != nil {
		t.Fatal("failed to delete a dummy kafka request")
	}

	pollErr = common.WaitForClustersMatchCriteriaToBeGivenCount(test.TestServices.DBFactory, &test.TestServices.ClusterService, &clusterCriteria, 3)
	Expect(pollErr).NotTo(HaveOccurred())

	//*********************************************************************
	//Test kafka instance creation and OSD cluster placement
	//*********************************************************************
	// add identity provider id for all of the clusters to avoid configuring the KAFKA_SRE IDP as it is not needed
	err = db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	Expect(err).NotTo(HaveOccurred())

	// Need to mark the clusters to be ready so that placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{"test01", "test02", "test03"}, api.ClusterReady)
	Expect(updateErr).NotTo(HaveOccurred())
	kafkas := []*dbapi.KafkaRequest{{
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser1",
		Name:          "dummy-kafka-1",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
	}, {
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser2",
		Name:          "dummy-kafka-2",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
	}}

	errK := test.TestServices.KafkaService.RegisterKafkaJob(kafkas[0])
	if errK != nil {
		Expect(errK).NotTo(HaveOccurred())
		return
	}

	dbFactory := test.TestServices.DBFactory
	kafkaFound, kafkaErr := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-1")
	Expect(kafkaErr).NotTo(HaveOccurred())
	Expect(kafkaFound.ClusterID).To(Equal("test03"))

	errK2 := test.TestServices.KafkaService.RegisterKafkaJob(kafkas[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	kafkaFound2, kafkaErr2 := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-2")
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(kafkaFound2.ClusterID).To(Equal("test02"))
}
