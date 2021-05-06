package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	ocm "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	clusterIDAssignTimeout     = 2 * time.Minute
	timeout                    = 3 * time.Hour
	interval                   = 10 * time.Second
	observatoriumReadyWaitTime = 30 * time.Minute
	clusterDeletionTimeout     = 5 * time.Minute
	clusterReadyInterval       = 10 * time.Minute
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
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.OSDClusterConfig)

	// create a cluster - this will need to be done manually until cluster creation is implemented in the cluster manager reconcile
	clusterRegisterError := clusterService.RegisterClusterJob(&api.Cluster{
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Region:        mocks.MockCluster.Region().ID(),
		MultiAZ:       testMultiAZ,
		Status:        api.ClusterAccepted,
	})
	if clusterRegisterError != nil {
		t.Fatalf("Failed to register cluster: %s", clusterRegisterError.Error())
	}

	var clusterID string
	// checking for cluster_id to be assigned to the new cluster
	err := common.Poll(interval, clusterIDAssignTimeout, func(attempt int, maxAttempts int) (done bool, err error) {
		t.Logf("%d/%d - Waiting for cluster id to be assigned to the new cluster", attempt, maxAttempts)
		foundCluster, svcErr := clusterService.FindCluster(services.FindClusterCriteria{
			Region:   mocks.MockCluster.Region().ID(),
			Provider: mocks.MockCluster.CloudProvider().ID(),
		})

		if svcErr != nil || foundCluster == nil {
			return true, fmt.Errorf("failed to find OSD cluster %s", svcErr)
		}
		clusterID = foundCluster.ClusterID
		return foundCluster.ClusterID != "", nil
	}, nil, func(attempt int, maxAttempts int, err error) error {
		t.Logf("%d/%d - Finished waiting for cluster id to be assigned to the new cluster", attempt, maxAttempts)
		return err
	})

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster id to be assigned: %v", err)

	// Wait for cluster to be ready
	checkClusterReadyErr := common.WaitForAndUpdateOSDClusterStatusToReady(h, t, clusterID, clusterReadyInterval, timeout)
	Expect(checkClusterReadyErr).NotTo(HaveOccurred(), "Error waiting for cluster '%s' to be ready: %v", clusterID, checkClusterReadyErr)

	// Get latest cluster details from the database
	cluster, findClusterByIdErr := clusterService.FindClusterByID(clusterID)
	if findClusterByIdErr != nil {
		t.Fatalf("Failed to find cluster '%s': %v", clusterID, err)
	}

	// save cluster struct to be reused in subsequent tests and cleanup script
	err = common.PersistClusterStruct(*cluster)
	if err != nil {
		t.Fatalf("failed to persist cluster struct %v", err)
	}
	Expect(cluster.DeletedAt.Valid).To(Equal(false), fmt.Sprintf("Expected deleted_at property to be non valid meaning cluster not soft deleted, instead got %v", cluster.DeletedAt))
	Expect(cluster.Status).To(Equal(api.ClusterReady), fmt.Sprintf("Expected status property to be %s, instead got %s ", api.ClusterReady, cluster.Status))
	Expect(cluster.IdentityProviderID).ToNot(BeEmpty(), "Expected identity_provider_id property to be defined")

	// check the state of cluster on ocm to ensure cluster was provisioned successfully
	ocmClusterStatus, err := ocmClient.GetClusterStatus(clusterID)
	if err != nil {
		t.Fatalf("failed to get cluster status from ocm")
	}
	Expect(ocmClusterStatus.State()).To(Equal(clustersmgmtv1.ClusterStateReady))

	// observatorium needs to get ready and until we change the way kafka
	// statuses are obtained, integration tests will fail without this wait time
	// as their status may not be correctly scraped jut after the OSD cluster is created
	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
		time.Sleep(observatoriumReadyWaitTime)
	}

	common.CheckMetricExposed(h, t, metrics.ClusterStatusSinceCreated)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsSuccessCount, constants.ClusterOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsTotalCount, constants.ClusterOperationCreate.String()))
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s{worker_type=\"%s\"}", metrics.KasFleetManager, metrics.ReconcilerDuration, "cluster"), true)
}

func TestClusterManager_SuccessfulReconcileDeprovisionCluster(t *testing.T) {
	var originalScalingType *string = new(string)
	startHook := func(h *test.Helper) {
		*originalScalingType = h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = *originalScalingType
	}

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer teardown()

	// setup required services
	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	clusterService := services.NewClusterService(h.Env().DBFactory, ocmClient, h.Env().Config.AWS, h.Env().Config.OSDClusterConfig)

	// Get a 'ready' osd cluster
	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	db := h.Env().DBFactory.New()
	cluster, _ := clusterService.FindClusterByID(clusterID)

	// create dummy kafkas and assign it to current cluster to make it not empty
	kafka := api.KafkaRequest{
		ClusterID:     cluster.ClusterID,
		MultiAZ:       false,
		Region:        cluster.Region,
		CloudProvider: cluster.CloudProvider,
		Name:          "dummy-kafka",
		Status:        constants.KafkaRequestStatusReady.String(),
	}

	if err := db.Save(&kafka).Error; err != nil {
		t.Error("failed to create a dummy kafka request")
		return
	}

	// Now create an OSD cluster with same characteristics and mark it as ready.
	// This cluster is empty so it will be deleted after some time
	dummyCluster := api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:     api.NewID(),
		MultiAZ:       cluster.MultiAZ,
		Region:        cluster.Region,
		CloudProvider: cluster.CloudProvider,
		Status:        api.ClusterReady,
	}

	if err := db.Save(&dummyCluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// We enable Dynamic Scaling at this point and not in the startHook due to
	// we want to ensure the pre-existing OSD cluster entry is stored in the DB
	// before enabling the dynamic scaling logic
	h.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = config.AutoScaling

	// checking that cluster has been deleted
	err := wait.PollImmediate(interval, clusterDeletionTimeout, func() (done bool, err error) {
		clusterFromDb, findClusterByIdErr := clusterService.FindClusterByID(dummyCluster.ClusterID)
		if findClusterByIdErr != nil {
			return false, findClusterByIdErr
		}
		return clusterFromDb == nil, nil // cluster has been deleted
	})

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster deletion: %v", err)
}
