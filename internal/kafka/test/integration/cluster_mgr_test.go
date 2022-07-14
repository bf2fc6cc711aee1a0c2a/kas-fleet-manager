package integration

import (
	"fmt"
	"strconv"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Tests a successful cluster reconcile
func TestClusterManager_SuccessfulReconcile(t *testing.T) {
	g := gomega.NewWithT(t)

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	zeroDeveloperLimit := 0
	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(d *config.DataplaneClusterConfig, p *config.ProviderConfig) {
		p.ProvidersConfig.SupportedProviders = config.ProviderList{
			config.Provider{
				Default: true,
				Name:    mocks.MockCluster.CloudProvider().ID(),
				Regions: config.RegionList{
					config.Region{
						Default: true,
						Name:    mocks.MockCluster.Region().ID(),
						SupportedInstanceTypes: config.InstanceTypeMap{
							api.StandardTypeSupport.String(): config.InstanceTypeConfig{
								Limit: nil,
							},
							api.DeveloperTypeSupport.String(): config.InstanceTypeConfig{
								Limit: &zeroDeveloperLimit,
							},
						},
					},
				},
			},
		}

		// turn auto scaling on to enable cluster auto creation
		d.DataPlaneClusterScalingType = config.AutoScaling
	})

	// setup required services
	ocmClient := test.TestServices.OCMClient

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()

	defer func() {
		teardown()
		kasfFleetshardSync.Stop()

		list, _ := test.TestServices.ClusterService.FindAllClusters(services.FindClusterCriteria{
			Region:   mocks.MockCluster.Region().ID(),
			Provider: mocks.MockCluster.CloudProvider().ID(),
		})

		for _, clusters := range list {
			if err := getAndDeleteServiceAccounts(clusters.ClientID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with client id: %v", clusters.ClientID)
			}
			if err := getAndDeleteServiceAccounts(clusters.ID, h.Env); err != nil {
				t.Fatalf("Failed to delete service account with cluster id: %v", clusters.ID)
			}
		}
	}()

	// checks that there is a cluster created
	clusterID, err := common.WaitForClusterIDToBeAssigned(test.TestServices.DBFactory, &test.TestServices.ClusterService, &services.FindClusterCriteria{
		Region:                mocks.MockCluster.Region().ID(),
		Provider:              mocks.MockCluster.CloudProvider().ID(),
		SupportedInstanceType: api.StandardTypeSupport.String(),
		MultiAZ:               true,
	})

	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for cluster id to be assigned: %v", err)

	// waiting for cluster state to become `cluster_provisioning`, so that its struct can be persisted
	cluster, checkProvisioningErr := common.WaitForClusterStatus(test.TestServices.DBFactory, &test.TestServices.ClusterService, clusterID, api.ClusterProvisioning)
	g.Expect(checkProvisioningErr).NotTo(gomega.HaveOccurred(), "Error waiting for cluster to start provisioning: %s %v", cluster.ClusterID, checkProvisioningErr)

	// save cluster struct to be reused in by the cleanup script if the cluster won't become ready before the timeout
	err = common.PersistClusterStruct(*cluster, api.ClusterProvisioning)
	if err != nil {
		t.Fatalf("failed to persist cluster struct %v", err)
	}

	// waiting for cluster state to become `ready`
	cluster, checkReadyErr := common.WaitForClusterStatus(test.TestServices.DBFactory, &test.TestServices.ClusterService, clusterID, api.ClusterReady)
	g.Expect(checkReadyErr).NotTo(gomega.HaveOccurred(), "Error waiting for cluster to be ready: %s %v", cluster.ClusterID, checkReadyErr)

	// check that the cluster has dynamic capacity info persisted
	dynamicCapacityInfo, err := cluster.RetrieveDynamicCapacityInfo()
	g.Expect(err).ToNot(gomega.HaveOccurred(), "Error retrieving dynamic capacity info")
	standardCapacity, ok := dynamicCapacityInfo[api.StandardTypeSupport.String()]
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(standardCapacity.MaxUnits).To(gomega.Equal(kasfleetshardsync.StandardCapacityInfo.MaxUnits))
	g.Expect(standardCapacity.RemainingUnits).To(gomega.Equal(kasfleetshardsync.StandardCapacityInfo.RemainingUnits))

	// save cluster struct to be reused in subsequent tests and cleanup script
	err = common.PersistClusterStruct(*cluster, api.ClusterProvisioned)
	if err != nil {
		t.Fatalf("failed to persist cluster struct %v", err)
	}
	g.Expect(cluster.DeletedAt.Valid).To(gomega.Equal(false), fmt.Sprintf("g.Expected deleted_at property to be non valid meaning cluster not soft deleted, instead got %v", cluster.DeletedAt))
	g.Expect(cluster.Status).To(gomega.Equal(api.ClusterReady), fmt.Sprintf("g.Expected status property to be %s, instead got %s ", api.ClusterReady, cluster.Status))
	g.Expect(cluster.IdentityProviderID).ToNot(gomega.BeEmpty(), "g.Expected identity_provider_id property to be defined")

	// check the state of cluster on ocm to ensure cluster was provisioned successfully
	ocmCluster, err := ocmClient.GetCluster(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster from ocm")
	}
	g.Expect(ocmCluster.Status().State()).To(gomega.Equal(clustersmgmtv1.ClusterStateReady))
	// check the state of externalID in the DB to check that it has been set appropriately
	g.Expect(cluster.ExternalID).NotTo(gomega.Equal(""))
	g.Expect(cluster.ExternalID).To(gomega.Equal(ocmCluster.ExternalID()))

	// check the state of the managed kafka addon on ocm to ensure it was installed successfully
	strimziOperatorAddonInstallation, err := ocmClient.GetAddon(cluster.ClusterID, test.TestServices.OCMConfig.StrimziOperatorAddonID)
	if err != nil {
		t.Fatalf("failed to get the strimzi operator addon for cluster %s", cluster.ClusterID)
	}
	g.Expect(strimziOperatorAddonInstallation.State()).To(gomega.Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// The cluster DNS should have been persisted
	ocmClusterDNS, err := ocmClient.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		t.Fatalf("failed to get cluster DNS from ocm")
	}
	g.Expect(cluster.ClusterDNS).To(gomega.Equal(ocmClusterDNS))

	common.CheckMetricExposed(h, t, metrics.ClusterCreateRequestDuration)
	common.CheckMetricExposed(h, t, metrics.ClusterStatusSinceCreated)

	// check that no capacity is used
	checkMetricsError := common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityUsed, "0", api.StandardTypeSupport.String(), cluster.ClusterID, cluster.Region, cluster.CloudProvider)
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())

	// available and max capacity metrics should have a value of standard max capacity
	metricValue := strconv.FormatInt(int64(standardCapacity.MaxUnits), 10)
	checkMetricsError = common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityMax, metricValue, api.StandardTypeSupport.String(), cluster.ClusterID, cluster.Region, cluster.CloudProvider)
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())
	checkMetricsError = common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityAvailable, metricValue, api.StandardTypeSupport.String(), cluster.ClusterID, cluster.Region, cluster.CloudProvider)
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsSuccessCount, constants2.ClusterOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.ClusterOperationsTotalCount, constants2.ClusterOperationCreate.String()))
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s{worker_type=\"%s\"}", metrics.KasFleetManager, metrics.ReconcilerDuration, "cluster"), true)
}

func TestClusterManager_SuccessfulReconcileDeprovisionCluster(t *testing.T) {
	g := gomega.NewWithT(t)

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
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

	db := test.TestServices.DBFactory.New()
	cluster, _ := test.TestServices.ClusterService.FindClusterByID(clusterID)

	// create dummy kafkas and assign it to current cluster to make it not empty
	kafka := dbapi.KafkaRequest{
		ClusterID:     cluster.ClusterID,
		MultiAZ:       false,
		Region:        cluster.Region,
		CloudProvider: cluster.CloudProvider,
		Name:          "dummy-kafka",
		Status:        constants2.KafkaRequestStatusReady.String(),
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
	DataplaneClusterConfig(h).DataPlaneClusterScalingType = config.AutoScaling

	// checking that cluster has been deleted
	err := common.WaitForClusterToBeDeleted(test.TestServices.DBFactory, &test.TestServices.ClusterService, dummyCluster.ClusterID)

	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for cluster deletion: %v", err)
}
