package integration

import (
	"fmt"
	"testing"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/common"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	clusterIDAssignTimeout = 5 * time.Minute
	timeout                = 2 * time.Hour
	interval               = 10 * time.Second
	readyWaitTime          = 30 * time.Minute
)

// Tests a successful cluster reconcile
func TestClusterManager_SuccessfulReconcile(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setup required services
	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.ClusterCreationConfig)

	// create a cluster - this will need to be done manually until cluster creation is implemented in the cluster manager reconcile
	clusterRegisterError := clusterService.RegisterClusterJob(&api.Cluster{
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Region:        mocks.MockCluster.Region().ID(),
		MultiAZ:       testMultiAZ,
	})
	if clusterRegisterError != nil {
		t.Fatalf("Failed to register cluster: %s", clusterRegisterError.Error())
	}

	var cluster api.Cluster

	// checking for cluster_id to be assigned to new cluster
	err := wait.PollImmediate(interval, clusterIDAssignTimeout, func() (done bool, err error) {

		foundCluster, svcErr := clusterService.FindCluster(services.FindClusterCriteria{
			Region:   mocks.MockCluster.Region().ID(),
			Provider: mocks.MockCluster.CloudProvider().ID(),
		})

		if svcErr != nil || foundCluster == nil {
			return true, fmt.Errorf("failed to find OSD cluster %s", svcErr)
		}
		cluster = *foundCluster
		return foundCluster.ClusterID != "", nil
	})

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster id to be assigned: %v", err)

	// waiting for cluster state to become `ready`
	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		foundCluster, findClusterErr := clusterService.FindClusterByID(cluster.ClusterID)
		if findClusterErr != nil {
			return true, fmt.Errorf("failed to find cluster with id %s: %s", cluster.ClusterID, err)
		}
		if foundCluster == nil {
			return false, nil
		}
		cluster = *foundCluster
		return cluster.Status == api.ClusterReady, nil
	})

	// ensure cluster is provisioned and terraformed successfully
	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster to be ready: %s %v", cluster.ID, err)
	Expect(cluster.DeletedAt).To(BeNil(), fmt.Sprintf("Expected deleted_at property to be empty, instead got %s", cluster.DeletedAt))

	// save cluster struct to be reused in subsequent tests
	err = utils.PersistClusterStruct(cluster)
	if err != nil {
		t.Fatalf("failed to persist cluster struct %v", err)
	}

	// check the state of cluster on ocm to ensure cluster was provisioned successfully
	ocmClusterStatus, err := ocmClient.GetClusterStatus(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster status from ocm")
	}
	Expect(ocmClusterStatus.State()).To(Equal(clustersmgmtv1.ClusterStateReady))

	// check the state of the managed kafka addon on ocm to ensure it was installed successfully
	addonInstallation, err := ocmClient.GetAddon(cluster.ClusterID, api.ManagedKafkaAddonID)
	if err != nil {
		t.Fatalf("failed to get addonInstallation for cluster %s", cluster.ClusterID)
	}
	Expect(addonInstallation.State()).To(Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// observatorium needs to get ready and until we change the way kafka
	// statuses are obtained, integration tests will fail without this wait time
	// as their status may not be correctly scraped jut after the OSD cluster is created
	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
		time.Sleep(readyWaitTime)
	}

	common.CheckMetricExposed(h, t, metrics.ClusterCreateRequestDuration)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.ClusterOperationsSuccessCount, constants.ClusterOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.ClusterOperationsTotalCount, constants.ClusterOperationCreate.String()))
}
