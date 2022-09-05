package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"

	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
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

func TestOCMClusterResourceQuotaMetrics(t *testing.T) {
	g := gomega.NewWithT(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()

	mockClusterRelatedResourceBuilder := amsv1.NewRelatedResource().Product(clusters.AMSQuotaOSDProductName).ResourceType(clusters.AMSQuotaClusterResourceType)
	mockClusterQuotaCostBuilder := amsv1.NewQuotaCost().
		QuotaID(mocks.MockQuotaId).
		Allowed(mocks.MockQuotaMaxAllowed).
		Consumed(mocks.MockQuotaConsumed).
		RelatedResources(mockClusterRelatedResourceBuilder)

	// create a mock quota that will not be exposed as a metric as it should be ignored by the get quota cost filter
	mockIgnoredRelatedResourceBuilder := amsv1.NewRelatedResource().ResourceName("resource-to-ignore")
	mockIgnoredQuotaCostBuilder := amsv1.NewQuotaCost().
		QuotaID("ignored-quota").
		Allowed(mocks.MockQuotaMaxAllowed).
		Consumed(mocks.MockQuotaConsumed).
		RelatedResources(mockIgnoredRelatedResourceBuilder)

	quotaCostList, err := amsv1.NewQuotaCostList().Items(mockClusterQuotaCostBuilder, mockIgnoredQuotaCostBuilder).Build()
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock quota list")
	ocmServerBuilder.SetGetOrganizationQuotaCost(quotaCostList, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(reconcilerConfig *workers.ReconcilerConfig) {
		// set the interval to 1 second to have sufficient time for the metric to be propagate
		// so that metrics checks does not timeout after 10s
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
	})
	defer teardown()

	// skip the test if not running on ocm mock mode. We cannot test using an actual ocm env as the values for the quota costs
	// cannot be pre-determined for the assertions as they may change overtime.
	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	checkMetricsError := common.WaitForMetricToBePresent(h, t, metrics.ClusterProviderResourceQuotaConsumed, fmt.Sprintf("%d", mocks.MockQuotaConsumed), `cluster_provider="ocm"`, fmt.Sprintf(`quota_id="%s"`, mocks.MockQuotaId))
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())

	checkMetricsError = common.WaitForMetricToBePresent(h, t, metrics.ClusterProviderResourceQuotaMaxAllowed, fmt.Sprintf("%d", mocks.MockQuotaMaxAllowed), `cluster_provider="ocm"`, fmt.Sprintf(`quota_id="%s"`, mocks.MockQuotaId))
	g.Expect(checkMetricsError).NotTo(gomega.HaveOccurred())

	// ensure any other quota that we are not interested in are not exposed (quotas are filtered to only include addons, cluster and compute node quotas)
	ignoredQuotaMetricExposed := common.IsMetricExposedWithValue(t, metrics.ClusterProviderResourceQuotaConsumed, `quota_id="ignored-quota"`)
	g.Expect(ignoredQuotaMetricExposed).To(gomega.Equal(false))
}

func TestOCMClusterResourceQuotaMetricsNoQuotaAvailable(t *testing.T) {
	g := gomega.NewWithT(t)

	// create a mock ocm api server with GET /{orgId}/quota_cost set to respond with no quota cost items
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()

	quotaCostList, err := amsv1.NewQuotaCostList().Items().Build()
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock quota list")
	ocmServerBuilder.SetGetOrganizationQuotaCost(quotaCostList, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(reconcilerConfig *workers.ReconcilerConfig) {
		// set the interval to 1 second to have sufficient time for the metric to be propagate
		// so that metrics checks does not timeout after 10s
		reconcilerConfig.ReconcilerRepeatInterval = 1 * time.Second
	})
	defer teardown()

	// skip the test if not running on ocm mock mode. We cannot test using an actual ocm env as the values for the quota costs
	// cannot be pre-determined for the assertions as they may change overtime.
	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// there should be no exposed cluster resource quota metric as mock ams returns no quota cost items
	ocmClusterResourceQuotaConsumedMetric := common.IsMetricExposedWithValue(t, metrics.ClusterProviderResourceQuotaConsumed, `cluster_provder="ocm"`)
	g.Expect(ocmClusterResourceQuotaConsumedMetric).To(gomega.Equal(false))

	ocmClusterResourceQuotaMaxAllowedMetric := common.IsMetricExposedWithValue(t, metrics.ClusterProviderResourceQuotaMaxAllowed, `cluster_provder="ocm"`)
	g.Expect(ocmClusterResourceQuotaMaxAllowedMetric).To(gomega.Equal(false))
}
