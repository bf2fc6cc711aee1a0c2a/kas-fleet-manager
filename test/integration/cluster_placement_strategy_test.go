package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	ocm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang/glog"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestClusterPlacementStrategy_ManualType(t *testing.T) {
	var clusterConfig *config.ClusterConfig
	var originalScalingType string

	// Start with no cluster config and manual scaling.
	startHook := func(h *test.Helper) {
		clusterConfig = h.Env().Config.OSDClusterConfig.ClusterConfig
		originalScalingType = h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType
		h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = config.ManualScaling
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.OSDClusterConfig.ClusterConfig = clusterConfig
		h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = originalScalingType
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer teardown()

	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
		t.SkipNow()
	}

	// load existing cluster and assign kafka to it so that it is not deleted

	db := h.Env().DBFactory.New()
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
	h.Env().Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		// this is a dummy cluster which will be auto created and should not be deleted because it has kafka in it
		config.ManualCluster{ClusterId: clusterWithKafkaID, KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
	})

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.OSDClusterConfig)

	// Ensure both clusters in the config file have been created
	pollErr := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {
		clusters, svcErr := clusterService.FindAllClusters(clusterCriteria)
		if svcErr != nil {
			return true, svcErr
		}
		t.Logf("%d clusters found in the database: ", len(clusters))
		return len(clusters) == 2, nil
	})
	Expect(pollErr).NotTo(HaveOccurred())

	//*********************************************************************
	//data plane cluster config - with new clusters
	//*********************************************************************
	h.Env().Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 0, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
		config.ManualCluster{ClusterId: "test02", KafkaInstanceLimit: 1, Region: clusterCriteria.Region, MultiAZ: clusterCriteria.MultiAZ, CloudProvider: clusterCriteria.Provider, Schedulable: true},
	})

	pollErr = wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {
		found, svcErr := clusterService.FindAllClusters(clusterCriteria)
		if svcErr != nil {
			return true, svcErr
		}
		return len(found) == 4, nil // make sure that the original cluster is there because it has kafka
	})
	Expect(pollErr).NotTo(HaveOccurred())

	// Now delete the kafka from the original cluster to check for placement strategy and wait for cluster deletion
	if err := db.Delete(&kafka).Error; err != nil {
		t.Fatal("failed to delete a dummy kafka request")
	}

	pollErr = wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {
		found, svcErr := clusterService.FindAllClusters(clusterCriteria)
		if svcErr != nil {
			return true, svcErr
		}
		return len(found) == 3, nil // we should only have clusters in the manual config
	})
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

	keySrv := services.NewKeycloakService(h.Env().Config.Keycloak, h.Env().Config.Keycloak.KafkaRealm)
	kafkaSrv := services.NewKafkaService(h.Env().DBFactory, clusterService, keySrv, h.Env().Config.Kafka, h.Env().Config.AWS, services.NewQuotaService(ocmClient))

	errK := kafkaSrv.RegisterKafkaJob(kafkas[0])
	if errK != nil {
		Expect(errK).NotTo(HaveOccurred())
		return
	}

	dbFactory := h.Env().DBFactory
	kafkaFound, kafkaErr := collectResult(dbFactory, "dummy-kafka-1")
	Expect(kafkaErr).NotTo(HaveOccurred())
	Expect(kafkaFound.ClusterID).To(Equal("test03"))

	errK2 := kafkaSrv.RegisterKafkaJob(kafkas[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	kafkaFound2, kafkaErr2 := collectResult(dbFactory, "dummy-kafka-2")
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(kafkaFound2.ClusterID).To(Equal("test02"))
}

func collectResult(dbFactory *db.ConnectionFactory, kafkaRequestName string) (*api.KafkaRequest, error) {
	kafkaFound := &api.KafkaRequest{}
	kafkaErr := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {
		if err := dbFactory.New().Where("name = ?", kafkaRequestName).First(kafkaFound).Error; err != nil {
			return false, err
		}
		glog.Infof("got kafka instance %v", kafkaFound)
		return kafkaFound.ClusterID != "", nil
	})

	return kafkaFound, kafkaErr
}
