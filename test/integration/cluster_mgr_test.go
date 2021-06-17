package integration

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks/kasfleetshardsync"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Tests a successful cluster reconcile
func TestClusterManager_SuccessfulReconcile(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := NewKafkaHelperWithHooks(t, ocmServer, func(c *config.OCMConfig) {
		c.ClusterLoggingOperatorAddonID = config.ClusterLoggingOperatorAddonID
	})
	defer teardown()

	// setup required services
	ocmClient := ocm.NewClient(testServices.OCM2Client.Connection)

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()
	defer kasfFleetshardSync.Stop()

	// create a cluster - this will need to be done manually until cluster creation is implemented in the cluster manager reconcile
	clusterRegisterError := testServices.ClusterService.RegisterClusterJob(&api.Cluster{
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Region:        mocks.MockCluster.Region().ID(),
		MultiAZ:       testMultiAZ,
		Status:        api.ClusterAccepted,
	})
	if clusterRegisterError != nil {
		t.Fatalf("Failed to register cluster: %s", clusterRegisterError.Error())
	}

	clusterID, err := utils.WaitForClusterIDToBeAssigned(testServices.DBFactory, &testServices.ClusterService, &services.FindClusterCriteria{
		Region:   mocks.MockCluster.Region().ID(),
		Provider: mocks.MockCluster.CloudProvider().ID(),
	},
	)

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster id to be assigned: %v", err)

	// waiting for cluster state to become `ready`
	cluster, checkReadyErr := utils.WaitForClusterStatus(testServices.DBFactory, &testServices.ClusterService, clusterID, api.ClusterReady)
	Expect(checkReadyErr).NotTo(HaveOccurred(), "Error waiting for cluster to be ready: %s %v", cluster.ClusterID, checkReadyErr)

	// save cluster struct to be reused in subsequent tests and cleanup script
	err = common.PersistClusterStruct(*cluster)
	if err != nil {
		t.Fatalf("failed to persist cluster struct %v", err)
	}
	Expect(cluster.DeletedAt.Valid).To(Equal(false), fmt.Sprintf("Expected deleted_at property to be non valid meaning cluster not soft deleted, instead got %v", cluster.DeletedAt))
	Expect(cluster.Status).To(Equal(api.ClusterReady), fmt.Sprintf("Expected status property to be %s, instead got %s ", api.ClusterReady, cluster.Status))
	Expect(cluster.IdentityProviderID).ToNot(BeEmpty(), "Expected identity_provider_id property to be defined")

	// check the state of cluster on ocm to ensure cluster was provisioned successfully
	ocmCluster, err := ocmClient.GetCluster(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster from ocm")
	}
	Expect(ocmCluster.Status().State()).To(Equal(clustersmgmtv1.ClusterStateReady))
	// check the state of externalID in the DB to check that it has been set appropriately
	Expect(cluster.ExternalID).NotTo(Equal(""))
	Expect(cluster.ExternalID).To(Equal(ocmCluster.ExternalID()))

	// check the state of the managed kafka addon on ocm to ensure it was installed successfully
	strimziOperatorAddonInstallation, err := ocmClient.GetAddon(cluster.ClusterID, testServices.OCMConfig.StrimziOperatorAddonID)
	if err != nil {
		t.Fatalf("failed to get the strimzi operator addon for cluster %s", cluster.ClusterID)
	}
	Expect(strimziOperatorAddonInstallation.State()).To(Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// check the state of the cluster logging operator addon on ocm to ensure it was installed successfully
	clusterLoggingOperatorAddonInstallation, err := ocmClient.GetAddon(cluster.ClusterID, testServices.OCMConfig.ClusterLoggingOperatorAddonID)
	if err != nil {
		t.Fatalf("failed to get the cluster logging addon installation for cluster %s", cluster.ClusterID)
	}
	Expect(clusterLoggingOperatorAddonInstallation.State()).To(Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// The cluster DNS should have been persisted
	ocmClusterDNS, err := ocmClient.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster DNS from ocm")
	}
	Expect(cluster.ClusterDNS).To(Equal(ocmClusterDNS))

	common.CheckMetricExposed(h, t, metrics.ClusterCreateRequestDuration)
	common.CheckMetricExposed(h, t, metrics.ClusterStatusSinceCreated)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsSuccessCount, constants.ClusterOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsTotalCount, constants.ClusterOperationCreate.String()))
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s{worker_type=\"%s\"}", metrics.KasFleetManager, metrics.ReconcilerDuration, "cluster"), true)
}

func TestClusterManager_SuccessfulReconcileDeprovisionCluster(t *testing.T) {

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := NewKafkaHelper(t, ocmServer)
	defer teardown()

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()
	defer kasfFleetshardSync.Stop()

	// setup required services
	// Get a 'ready' osd cluster
	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	db := testServices.DBFactory.New()
	cluster, _ := testServices.ClusterService.FindClusterByID(clusterID)

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
	h.Env.Config.OSDClusterConfig.DataPlaneClusterScalingType = config.AutoScaling

	// checking that cluster has been deleted
	err := utils.WaitForClusterToBeDeleted(testServices.DBFactory, &testServices.ClusterService, dummyCluster.ClusterID)

	Expect(err).NotTo(HaveOccurred(), "Error waiting for cluster deletion: %v", err)
}
