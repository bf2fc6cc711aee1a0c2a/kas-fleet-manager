package integration

import (
	"fmt"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"testing"
	"time"

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
	timeout  = 2 * time.Hour
	interval = 10 * time.Second
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
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS)

	// create a cluster - this will need to be done manually until cluster creation is implemented in the cluster manager reconcile
	newCluster, provisionErr := clusterService.Create(&api.Cluster{
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Region:        mocks.MockCluster.Region().ID(),
		MultiAZ:       testMultiAZ,
	})
	if provisionErr != nil {
		t.Fatalf("Failed to create a new cluster: %s", provisionErr.Error())
	}

	Expect(newCluster.Version().ID()).To(Equal(ocm.OpenshiftVersion))

	// Wait until cluster status has been updated to ready
	var cluster api.Cluster
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		foundCluster, findClusterErr := clusterService.FindClusterByID(newCluster.ID())
		if findClusterErr != nil {
			return true, fmt.Errorf("failed to find cluster with id %s: %s", newCluster.ID(), err)
		}
		cluster = foundCluster
		return cluster.Status == api.ClusterReady, nil
	})

	// ensure cluster is provisioned and terraformed successfully
	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster to be ready: %v", cluster.ID, err)
	Expect(cluster.ID).To(Equal(cluster.ClusterID))
	Expect(cluster.DeletedAt).To(BeNil(), fmt.Sprintf("Expected deleted_at property to be empty, instead got %s", cluster.DeletedAt))

	// check the state of cluster on ocm to ensure cluster was provisioned successfully
	ocmClusterStatus, err := ocmClient.GetClusterStatus(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster status from ocm")
	}
	Expect(ocmClusterStatus.State()).To(Equal(clustersmgmtv1.ClusterStateReady))

	// check the state of the managed kafka addon on ocm to ensure it was installed successfully
	addonInstallation, err := ocmClient.GetManagedKafkaAddon(newCluster.ID())
	if err != nil {
		t.Fatalf("failed to get addonInstallation for cluster %s", newCluster.ID())
	}
	Expect(addonInstallation.State()).To(Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// save cluster struct to be reused in subsequent tests
	err = utils.PersistClusterStruct(cluster)
	if err != nil {
		t.Log(err)
	}
	common.CheckMetricExposed(h, t, metrics.ClusterCreateRequestDuration)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.ClusterOperationsSuccessCount, constants.ClusterOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.ClusterOperationsTotalCount, constants.ClusterOperationCreate.String()))
}
