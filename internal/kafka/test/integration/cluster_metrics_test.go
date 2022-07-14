package integration

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"

	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

func TestClusterCapacityUsedMetric(t *testing.T) {
	g := gomega.NewWithT(t)
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(reconcilerConfig *workers.ReconcilerConfig) {
		// set the interval to 1 second to have sufficient time for the metric to be propagate
		// so that metrics checks does not timeout after 10s
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
	})
	defer teardown()

	// setup pre-requisites to performing requests
	db := test.TestServices.DBFactory.New()

	cluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = "standard"
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = "some-cluster-id"
	})

	kafka := kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues(), func(kr *dbapi.KafkaRequest) {
		kr.Meta = api.Meta{
			ID: api.NewID(),
		}
		kr.Region = cluster.Region
		kr.CloudProvider = cluster.CloudProvider
		kr.SizeId = "x1"
		kr.InstanceType = types.STANDARD.String()
		kr.ClusterID = cluster.ClusterID
	})

	err := db.Create(kafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// check that there is "1" streaming unit capacity used
	metricValue := "1"
	checkMetricsError := common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityUsed, metricValue, kafka.InstanceType, kafka.ClusterID, kafka.Region, kafka.CloudProvider)
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())

	// now delete the kafka and make verify that the capacity used goes down from "1" to "0"
	err = db.Unscoped().Delete(kafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// now verify that the metric value has gone down to "0" after the kafka deletion
	metricValue = "0"
	checkMetricsError = common.WaitForMetricToBePresent(h, t, metrics.ClusterStatusCapacityUsed, metricValue, kafka.InstanceType, kafka.ClusterID, kafka.Region, kafka.CloudProvider)
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())
}
