package integration

import (
	"fmt"
	"strconv"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	mockclusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
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
	var dataplaneConfig *config.DataplaneClusterConfig
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
		d.DynamicScalingConfig.EnableDynamicScaleUpManagerScaleUpTrigger = true
		dataplaneConfig = d
	})

	// setup required services
	ocmClient := test.TestServices.OCMClient

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()

	defer func() {
		kasfFleetshardSync.Stop()
		teardown()

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
	dynamicCapacityInfo := cluster.RetrieveDynamicCapacityInfo()
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

	// check that the default machine pool is created with correct min/max node and machine type.
	// Only do the check for when running against real OCM environment
	ocmConfig := test.TestServices.OCMConfig
	machineTypeConfig, err := dataplaneConfig.DefaultComputeMachineType(cloudproviders.AWS)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if ocmConfig.MockMode != ocm.MockModeEmulateServer {
		nodes := ocmCluster.Nodes()
		g.Expect(nodes.ComputeMachineType().ID()).To(gomega.Equal(machineTypeConfig.ClusterWideWorkloadMachineType))
		g.Expect(nodes.AutoscaleCompute().MinReplicas()).To(gomega.Equal(3))
		g.Expect(nodes.AutoscaleCompute().MaxReplicas()).To(gomega.Equal(18))
	}

	// check the state of the managed kafka addon on ocm to ensure it was installed successfully
	strimziOperatorAddonInstallation, err := ocmClient.GetAddon(cluster.ClusterID, test.TestServices.OCMConfig.StrimziOperatorAddonID)
	if err != nil {
		t.Fatalf("failed to get the strimzi operator addon for cluster %s", cluster.ClusterID)
	}
	g.Expect(strimziOperatorAddonInstallation.State()).To(gomega.Equal(clustersmgmtv1.AddOnInstallationStateReady))

	// check that the kafka-standard machine pool is created in OCM
	standardKafkaMachinePoolID := "kafka-standard"
	if ocmConfig.MockMode != ocm.MockModeEmulateServer {
		standardKafkaMachinePool, machinePoolErr := ocmClient.GetMachinePool(cluster.ClusterID, standardKafkaMachinePoolID)
		// check that the machinepool is present
		g.Expect(machinePoolErr).NotTo(gomega.HaveOccurred())
		g.Expect(standardKafkaMachinePool).NotTo(gomega.BeNil())
		g.Expect(standardKafkaMachinePool.InstanceType()).To(gomega.Equal(machineTypeConfig.KafkaWorkloadMachineType))

		// check min and max nodes configuration
		autoscaling := standardKafkaMachinePool.Autoscaling()
		config, _ := dataplaneConfig.DynamicScalingConfig.GetConfigForInstanceType(api.StandardTypeSupport.String())
		g.Expect(autoscaling.MinReplicas()).To(gomega.Equal(3))
		g.Expect(autoscaling.MaxReplicas()).To(gomega.Equal(config.ComputeNodesConfig.MaxComputeNodes))

		// check that the machinepool labels match what's expected
		bf2InstanceProfileTypeKey := "bf2.org/kafkaInstanceProfileType"
		labels := standardKafkaMachinePool.Labels()
		labelValue, ok := labels[bf2InstanceProfileTypeKey]
		g.Expect(ok).To(gomega.BeTrue())
		g.Expect(labelValue).To(gomega.Equal(api.StandardTypeSupport.String()))

		// check that the machinepool taints match what's expected
		taints := standardKafkaMachinePool.Taints()
		taintFound := false
		for _, taint := range taints {
			if taint.Key() == bf2InstanceProfileTypeKey {
				taintFound = true
				g.Expect(taint.Effect()).To(gomega.Equal("NoExecute"))
				g.Expect(taint.Value()).To(gomega.Equal(api.StandardTypeSupport.String()))
				break
			}
		}

		g.Expect(taintFound).To(gomega.BeTrue())
	}

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
								Limit:                                   nil,
								MinAvailableCapacitySlackStreamingUnits: 1,
							},
						},
					},
				},
			},
		}
		// enable scale down trigger
		d.DynamicScalingConfig.EnableDynamicScaleDownManagerScaleDownTrigger = true
		d.DynamicScalingConfig.EnableDynamicScaleUpManagerScaleUpTrigger = false
		d.DataPlaneClusterScalingType = config.AutoScaling
	})

	defer teardown()

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()
	defer kasfFleetshardSync.Stop()

	// Get a 'ready' osd cluster
	cluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = types.STANDARD.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "some-cluster-id"
		cluster.CloudProvider = mocks.MockCluster.CloudProvider().ID()
		cluster.Region = mocks.MockCluster.Region().ID()
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.Status = api.ClusterProvisioning
	})

	db := test.TestServices.DBFactory.New()
	err := db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	_, err = common.WaitForClusterStatus(h.DBFactory(), &test.TestServices.ClusterService, cluster.ClusterID, api.ClusterReady)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "data plane cluster failed to reach status ready")

	// create dummy kafkas and assign it to current cluster to make it not empty
	kafka := kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues(), func(kr *dbapi.KafkaRequest) {
		kr.Meta = api.Meta{
			ID: api.NewID(),
		}
		kr.Region = cluster.Region
		kr.CloudProvider = cluster.CloudProvider
		kr.SizeId = "x1"
		kr.InstanceType = types.STANDARD.String()
		kr.ClusterID = cluster.ClusterID
		kr.Name = "dummy-kafka"
		kr.Status = constants2.KafkaRequestStatusReady.String()
	})

	if err := db.Save(&kafka).Error; err != nil {
		t.Error("failed to create a dummy kafka request")
		return
	}

	// Now create an OSD cluster with same characteristics and mark it as ready.
	// This cluster is empty so it will be deleted after some time
	dummyCluster := mockclusters.BuildCluster(func(cluster *api.Cluster) {
		dynamicCapacityInfoString := fmt.Sprintf("{\"standard\":{\"max_nodes\":1,\"max_units\":%[1]d,\"remaining_units\": 0}}", kasfleetshardsync.StandardCapacityInfo.MaxUnits)
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone // ensures no errors will occur due to it not being available on ocm
		cluster.SupportedInstanceType = api.StandardTypeSupport.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "test-cluster-to-be-deleted"
		cluster.CloudProvider = mocks.MockCloudProviderID
		cluster.Region = mocks.MockCloudRegionID
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.AvailableStrimziVersions = api.JSON(mockclusters.AvailableStrimziVersions)
		cluster.DynamicCapacityInfo = api.JSON([]byte(dynamicCapacityInfoString))
	})

	if err := db.Save(&dummyCluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// checking that cluster has been deleted
	err = common.WaitForClusterToBeDeleted(test.TestServices.DBFactory, &test.TestServices.ClusterService, dummyCluster.ClusterID)

	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error waiting for cluster deletion: %v", err)
}
