package integration

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	ocm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang/glog"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
)

func TestClusterPlacementStrategy_ManualType(t *testing.T) {
	var oriFlag string
	startHook := func(h *test.Helper) {
		oriFlag = h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType
		h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = "manual"
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = oriFlag
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer teardown()

	//*********************************************************************
	//pre-create an cluster
	//*********************************************************************
	h.Env().Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: "us-east-1", MultiAZ: true, CloudProvider: "aws"},
	})

	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.OSDClusterConfig)

	currentClusterId := "test03"
	err1 := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {

		if foundCluster, svcErr := clusterService.FindClusterByID(currentClusterId); svcErr != nil {
			return true, fmt.Errorf("failed to find OSD cluster %s", svcErr)
		} else {
			if svcErr != nil {
				return true, svcErr
			}

			if foundCluster == nil {
				return false, nil
			}
			cluster := *foundCluster
			glog.Infof("Cluster found: %s %s %s", cluster.ClusterID, cluster.Status.String(), cluster.ID)

			return foundCluster.ClusterID != "", nil
		}
	})
	Expect(err1).NotTo(HaveOccurred())

	//*********************************************************************
	//data plane cluster config - with new clusters
	//*********************************************************************
	h.Env().Config.OSDClusterConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		config.ManualCluster{ClusterId: "test03", KafkaInstanceLimit: 1, Region: "us-east-1", MultiAZ: true, CloudProvider: "aws"},
		config.ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 0, Region: "us-east-1", MultiAZ: true, CloudProvider: "aws"},
		config.ManualCluster{ClusterId: "test02", KafkaInstanceLimit: 1, Region: "us-east-1", MultiAZ: true, CloudProvider: "aws"},
	})
	err3 := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {

		found, svcErr := clusterService.FindAllClusters(services.FindClusterCriteria{
			Provider: "aws",
			Region:   "us-east-1",
			MultiAZ:  true,
		})

		if svcErr != nil {
			return true, fmt.Errorf("failed to find OSD cluster with wanted status %s", svcErr)
		}
		if found == nil {
			return false, nil
		}
		return len(found) == 3, nil
	})
	Expect(err3).NotTo(HaveOccurred())

	//*********************************************************************
	//Test kafka instance creation and OSD cluster placement
	//*********************************************************************
	kafka := []*api.KafkaRequest{
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser1",
			Name:          "dummy-kafka",
			Status:        constants.KafkaRequestStatusAccepted.String(),
		},
		{MultiAZ: true,
			Region:        "us-east-1",
			CloudProvider: "aws",
			Owner:         "dummyuser2",
			Name:          "dummy-kafka",
			Status:        constants.KafkaRequestStatusAccepted.String(),
		},
	}

	keySrv := services.NewKeycloakService(h.Env().Config.Keycloak, h.Env().Config.Keycloak.KafkaRealm)
	kafkaSrv := services.NewKafkaService(h.Env().DBFactory, services.NewSyncsetService(ocmClient), clusterService, keySrv, h.Env().Config.Kafka, h.Env().Config.AWS, services.NewQuotaService(ocmClient))

	errK := kafkaSrv.RegisterKafkaJob(kafka[0])
	if errK != nil {
		Expect(errK).NotTo(HaveOccurred())
		return
	}

	kafkaFound, kafkaErr := collectResult(kafkaSrv, 1)
	Expect(kafkaErr).NotTo(HaveOccurred())
	Expect(kafkaFound[0].ClusterID).To(Equal("test03"))

	errK2 := kafkaSrv.RegisterKafkaJob(kafka[1])
	if errK2 != nil {
		Expect(errK2).NotTo(HaveOccurred())
		return
	}

	kafkaFound2, kafkaErr2 := collectResult(kafkaSrv, 2)
	Expect(kafkaErr2).NotTo(HaveOccurred())
	Expect(len(kafkaFound2)).To(Equal(2))
	for _, v := range kafkaFound2 {
		if v.Owner == "dummyuser2" {
			Expect(v.ClusterID).To(Equal("test02"))
		} else if v.Owner == "dummyuser1" {
			Expect(v.ClusterID).To(Equal("test03"))
		}
	}
}

func collectResult(kafkaSrv services.KafkaService, recordCnt int) ([]*api.KafkaRequest, error) {

	var kafkaFound []*api.KafkaRequest
	kafkaErr := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {
		found, er := kafkaSrv.ListByStatus(constants.KafkaRequestStatusReady)

		if er != nil {
			return true, fmt.Errorf("failed to find the Kafka clusters with ready status %s", er)
		}
		if found == nil {
			return false, nil
		}
		//for _, v := range found {
		//	glog.Infof("Kafka found ->>: %v ", v)
		//}
		kafkaFound = found
		return len(kafkaFound) == recordCnt, nil
	})

	return kafkaFound, kafkaErr
}
