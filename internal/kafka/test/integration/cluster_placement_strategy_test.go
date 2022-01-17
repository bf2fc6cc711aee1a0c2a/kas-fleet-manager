package integration

import (
	"encoding/json"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

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

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()
	defer kasfFleetshardSync.Stop()

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
		InstanceType:  types.STANDARD.String(),
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
		ClusterID:             clusterWithKafkaID,
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
			KafkaInstanceLimit:    1,
			Region:                clusterCriteria.Region,
			MultiAZ:               clusterCriteria.MultiAZ,
			CloudProvider:         clusterCriteria.Provider,
			Schedulable:           true,
			SupportedInstanceType: api.AllInstanceTypeSupport.String(),
		},
		// this is a dummy cluster which will be auto created and should not be deleted because it has kafka in it
		config.ManualCluster{
			ClusterId:             standaloneCluster.ClusterID,
			KafkaInstanceLimit:    1,
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
	// add identity provider id to avoid configuring the Kafka_SRE IDP
	err := db.Exec("UPDATE clusters set identity_provider_id = 'some-identity-provider-id'").Error
	Expect(pollErr).NotTo(HaveOccurred())
	Expect(err).NotTo(HaveOccurred())

	// Ensure that cluster dns is populated with given value
	cluster, err := test.TestServices.ClusterService.FindClusterByID(clusterWithKafkaID)
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
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "eval,standard"},
		config.ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "eval,standard"},
		config.ManualCluster{ClusterId: "test02", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true, SupportedInstanceType: "eval,standard"},
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

	// Need to mark the clusters to ClusterWaitingForKasFleetShardOperator so that fleetshardsync and placement can actually happen
	updateErr := test.TestServices.ClusterService.UpdateMultiClusterStatus([]string{"test01", "test02", "test03"}, api.ClusterWaitingForKasFleetShardOperator)
	Expect(updateErr).NotTo(HaveOccurred())
	kafkas := []*dbapi.KafkaRequest{{
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		// the Owner has to be email address from  config/quota-management-list-configuration.yaml
		// in order to create STANDARD kafka instance. Otherwise it will default to EVAL
		Owner:        "testuser3@example.com",
		Name:         "dummy-kafka-1",
		Status:       constants2.KafkaRequestStatusAccepted.String(),
		InstanceType: types.STANDARD.String(),
	}, {
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser2",
		Name:          "dummy-kafka-2",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
		InstanceType:  types.EVAL.String(),
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
	Expect(kafkaFound.InstanceType).To(Equal(types.STANDARD.String()))

	errK2 := test.TestServices.KafkaService.RegisterKafkaJob(kafkas[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	kafkaFound2, kafkaErr2 := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-2")
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(kafkaFound2.ClusterID).To(Equal("test02"))
	Expect(kafkaFound2.InstanceType).To(Equal(types.EVAL.String()))
}

func TestClusterPlacementStrategy_CheckInstanceTypePlacement(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// load existing cluster and assign kafka to it so that it is not deleted

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
			KafkaVersions: []api.KafkaVersion{
				api.KafkaVersion{Version: "2.7.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				api.KafkaIBPVersion{Version: "2.7"},
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
	kafkas := []*dbapi.KafkaRequest{{
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser1",
		Name:          "dummy-kafka-1",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
		InstanceType:  types.STANDARD.String(),
	}, {
		MultiAZ:       true,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "dummyuser2",
		Name:          "dummy-kafka-2",
		Status:        constants2.KafkaRequestStatusAccepted.String(),
		InstanceType:  types.EVAL.String(),
	}}

	if err := db.Create(kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	dbFactory := test.TestServices.DBFactory

	// verify that a standard instance is assigned to a data plane cluster that supports only standard instances
	kafkaFound, kafkaErr := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-1")
	Expect(kafkaErr).NotTo(HaveOccurred())
	Expect(kafkaFound.ClusterID).To(Equal(standardInstanceCluster))

	// verify that an eval instance is assigned to a data plane cluster that supports only eval instances
	kafkaFound2, kafkaErr2 := common.WaitForKafkaClusterIDToBeAssigned(dbFactory, "dummy-kafka-2")
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(kafkaFound2.ClusterID).To(Equal(evalInstanceCluster))
}
