package integration

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	mockclusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

// delay reconciliation so that the kafkas in deleting state are not removed before we can perform test assertions
const longReconcilerInterval = 10 * time.Minute

var cluster = mockclusters.BuildCluster(func(cluster *api.Cluster) {
	cluster.Meta = api.Meta{
		ID: api.NewID(),
	}
	cluster.ProviderType = api.ClusterProviderStandalone // ensures no errors will occur due to it not being available on ocm
	cluster.SupportedInstanceType = api.AllInstanceTypeSupport.String()
	cluster.ClientID = "some-client-id"
	cluster.ClientSecret = "some-client-secret"
	cluster.ClusterID = "test-cluster"
	cluster.CloudProvider = mocks.MockCloudProviderID
	cluster.Region = mocks.MockCloudRegionID
	cluster.Status = api.ClusterReady
	cluster.ProviderSpec = api.JSON{}
	cluster.ClusterSpec = api.JSON{}
	cluster.AvailableStrimziVersions = api.JSON{}
	cluster.DynamicCapacityInfo = api.JSON{}
})

var readyKafka = kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues(), func(kr *dbapi.KafkaRequest) {
	kr.Meta = api.Meta{
		ID: api.NewID(),
	}
	kr.Region = cluster.Region
	kr.CloudProvider = cluster.CloudProvider
	kr.SizeId = "x1"
	kr.InstanceType = types.STANDARD.String()
	kr.ClusterID = cluster.ClusterID
	kr.Status = constants.KafkaRequestStatusReady.String()
})

var deletingKafka = kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues(), func(kr *dbapi.KafkaRequest) {
	kr.Meta = api.Meta{
		ID: api.NewID(),
	}
	kr.Region = cluster.Region
	kr.CloudProvider = cluster.CloudProvider
	kr.SizeId = "x1"
	kr.InstanceType = types.STANDARD.String()
	kr.ClusterID = cluster.ClusterID
	kr.Status = constants.KafkaRequestStatusDeleting.String()
})

func TestClusterService_FindKafkaInstanceCount(t *testing.T) {
	g := gomega.NewWithT(t)

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(r *workers.ReconcilerConfig, d *config.DataplaneClusterConfig) {
		d.DataPlaneClusterScalingType = config.NoScaling
		d.EnableKafkaSreIdentityProviderConfiguration = false

		// delay reconciliation so that the kafkas in deleting state are not removed before we can perform test assertions
		r.ReconcilerRepeatInterval = 10 * time.Minute
	})

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()

	t.Cleanup(func() {
		kasfFleetshardSync.Stop()
		teardown()
		ocmServer.Close()
	})

	db := h.DBFactory().New()
	err := db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create data plane cluster")

	// create one deleting kafka and another Kafka in a different status (ready is choosen here)

	err = db.Create(readyKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed create a ready kafka in the database")

	err = db.Create(deletingKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create a kafka in deleting status in the database")

	// retrieves kafka count for the cluster and verify that the count is one i.e deleting kafkas should not be included
	kafkaCount, err := test.TestServices.ClusterService.FindKafkaInstanceCount([]string{cluster.ClusterID})
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to retrieve kafka counts database")

	g.Expect(kafkaCount).To(gomega.HaveLen(1)) // count should contain the cluster under test

	clusterKafkaCount := kafkaCount[0]
	g.Expect(clusterKafkaCount.Clusterid).To(gomega.Equal(cluster.ClusterID)) // cluster id should equal to the cluster id of the cluster under test
	g.Expect(clusterKafkaCount.Count).To(gomega.Equal(1))                     // count should be one since we include only statuses that do not consume resources
}

func TestClusterService_FindStreamingUnitCountByClusterAndInstanceType(t *testing.T) {
	g := gomega.NewWithT(t)

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(r *workers.ReconcilerConfig, d *config.DataplaneClusterConfig) {
		d.DataPlaneClusterScalingType = config.NoScaling
		d.EnableKafkaSreIdentityProviderConfiguration = false
		r.ReconcilerRepeatInterval = longReconcilerInterval
	})

	kasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	kasfFleetshardSync := kasFleetshardSyncBuilder.Build()
	kasfFleetshardSync.Start()

	t.Cleanup(func() {
		kasfFleetshardSync.Stop()
		teardown()
		ocmServer.Close()
	})

	db := h.DBFactory().New()
	err := db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create data plane cluster")

	// create one deleting kafka  and another Kafka in a different status (ready is choosen here)
	err = db.Create(readyKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed create a ready kafka in the database")

	err = db.Create(deletingKafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create a kafka in deleting status in the database")

	// retrieves streaming unit count for the cluster and verify that the count is one i.e deleting and failed kafkas should not be included
	streamingUnitCounts, err := test.TestServices.ClusterService.FindStreamingUnitCountByClusterAndInstanceType()
	g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to retrieve kafka counts database")

	g.Expect(streamingUnitCounts).To(gomega.HaveLen(2)) // count will have two elements, one for each supported instance type of the cluster under test

	for _, instanceTypeCount := range streamingUnitCounts {
		g.Expect(instanceTypeCount.ClusterId).To(gomega.Equal(cluster.ClusterID)) // cluster id should equal to the cluster id of the cluster under test
		g.Expect(instanceTypeCount.InstanceType).NotTo(gomega.BeEmpty())          // instance type should be set
		g.Expect(instanceTypeCount.Status).NotTo(gomega.BeEmpty())                // status should be set
		g.Expect(instanceTypeCount.Region).To(gomega.Equal(cluster.Region))
		g.Expect(instanceTypeCount.CloudProvider).To(gomega.Equal(cluster.CloudProvider))

		if instanceTypeCount.InstanceType == api.DeveloperTypeSupport.String() {
			g.Expect(instanceTypeCount.Count).To(gomega.Equal(int32(0))) // no developer instance was assigned to this cluster
		} else if instanceTypeCount.InstanceType == api.StandardTypeSupport.String() {
			g.Expect(instanceTypeCount.Count).To(gomega.Equal(int32(1))) // only one streaming unit instance was counted as kafka in deleting state was ignored
		}
	}
}
