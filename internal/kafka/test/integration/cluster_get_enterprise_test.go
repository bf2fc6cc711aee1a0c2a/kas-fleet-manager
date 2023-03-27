package integration

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	objRefMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/object_reference"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"

	"github.com/onsi/gomega"
)

func TestEnterpriseClusterGet(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// only run test in integration env
	ocmConfig := test.TestServices.OCMConfig
	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name != environments.IntegrationEnv {
		t.SkipNow()
	}

	validNonExistentClusterID := "1234567890abcdef1234567890abcdef"

	dynamicCapacityInfoString := "{\"standard\":{\"max_nodes\":9,\"max_units\":3,\"remaining_units\":3}}"

	entCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterID = api.NewID()
		cluster.AccessKafkasViaPrivateNetwork = true
		cluster.DynamicCapacityInfo = api.JSON([]byte(dynamicCapacityInfoString))
	})

	nonEntCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.ClusterType = api.ManagedDataPlaneClusterType.String()
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterID = api.NewID()
		cluster.DynamicCapacityInfo = api.JSON([]byte(dynamicCapacityInfoString))
	})

	otherOrgCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ClusterID = api.NewID()
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.OrganizationID = "99999999"
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.Status = api.ClusterReady
		cluster.AccessKafkasViaPrivateNetwork = true
		cluster.DynamicCapacityInfo = api.JSON{}
	})

	db := test.TestServices.DBFactory.New()

	err := db.Create(entCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(nonEntCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(otherOrgCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	adminAccount := h.NewAccountWithNameAndOrg("admin", kafkaMocks.DefaultOrganisationId)

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	noReqClaimsAccount := h.NewAccount("", "", "", "")
	noReqClaimsCtx := h.NewAuthenticatedContext(noReqClaimsAccount, nil)

	nonAdminAccount := h.NewAccountWithNameAndOrg("non-admin", kafkaMocks.DefaultOrganisationId)
	nonAdminCtx := h.NewAuthenticatedContext(nonAdminAccount, nil)

	// no org within claims
	cluster, resp, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(noReqClaimsCtx, entCluster.ClusterID)
	if resp != nil {
		g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)

	clusterAddonParameters, resp2, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(noReqClaimsCtx, entCluster.ClusterID)
	if resp2 != nil {
		g.Expect(resp2.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp2.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterAddonParametersRespValues(clusterAddonParameters, g)

	// non-admin user of the same org trying to access addon params
	clusterAddonParameters, resp3, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(nonAdminCtx, entCluster.ClusterID)
	if resp3 != nil {
		g.Expect(resp3.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp3.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterAddonParametersRespValues(clusterAddonParameters, g)

	// non-existent cluster
	cluster, resp4, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, validNonExistentClusterID)
	if resp4 != nil {
		g.Expect(resp4.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp4.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)

	clusterAddonParameters, resp5, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(adminCtx, validNonExistentClusterID)
	if resp5 != nil {
		g.Expect(resp5.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp5.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterAddonParametersRespValues(clusterAddonParameters, g)

	// enterprise cluster from different org
	cluster, resp6, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, otherOrgCluster.ClusterID)
	if resp6 != nil {
		g.Expect(resp6.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp6.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)

	clusterAddonParameters, resp7, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(adminCtx, otherOrgCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterAddonParametersRespValues(clusterAddonParameters, g)
	if resp7 != nil {
		g.Expect(resp7.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp7.Body.Close()
	}

	// non-enterprise cluster from same org
	cluster, resp8, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, nonEntCluster.ClusterID)
	if resp8 != nil {
		g.Expect(resp8.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp8.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)

	clusterAddonParameters, resp9, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(adminCtx, nonEntCluster.ClusterID)
	if resp9 != nil {
		g.Expect(resp9.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp9.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterAddonParametersRespValues(clusterAddonParameters, g)

	// success
	cluster, resp10, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, entCluster.ClusterID)
	if resp10 != nil {
		g.Expect(resp10.StatusCode).To(gomega.Equal(http.StatusOK))
		defer resp10.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster.ClusterId).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(strings.Contains(cluster.Href, entCluster.ClusterID)).To(gomega.BeTrue())
	g.Expect(cluster.Id).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(cluster.Kind).To(gomega.Equal(objRefMocks.KindCluster))
	g.Expect(cluster.Status).To(gomega.Equal(entCluster.Status.String()))
	g.Expect(cluster.AccessKafkasViaPrivateNetwork).To((gomega.Equal(entCluster.AccessKafkasViaPrivateNetwork)))
	g.Expect(cluster.Region).To(gomega.Equal(entCluster.Region))
	g.Expect(cluster.CloudProvider).To(gomega.Equal(entCluster.CloudProvider))
	g.Expect(cluster.MultiAz).To(gomega.BeTrue())
	g.Expect(cluster.CapacityInformation).To(gomega.Equal(public.EnterpriseClusterAllOfCapacityInformation{
		KafkaMachinePoolNodeCount:    9,
		MaximumKafkaStreamingUnits:   3,
		RemainingKafkaStreamingUnits: 3,
		ConsumedKafkaStreamingUnits:  0,
	}))
	g.Expect(cluster.SupportedInstanceTypes.InstanceTypes).To(gomega.HaveLen(1))
	standardInstanceType := cluster.SupportedInstanceTypes.InstanceTypes[0]
	g.Expect(standardInstanceType.Id).To(gomega.Equal(types.STANDARD.String()))
	g.Expect(standardInstanceType.SupportedBillingModels).To(gomega.HaveLen(1))
	g.Expect(standardInstanceType.SupportedBillingModels[0].Id).To(gomega.Equal(constants.BillingModelEnterprise.String()))
	g.Expect(standardInstanceType.Sizes).To(gomega.HaveLen(2))
	g.Expect(standardInstanceType.Sizes[0].Id).To(gomega.Equal("x1"))
	g.Expect(standardInstanceType.Sizes[1].Id).To(gomega.Equal("x2"))

	// assign a Kafka onto the cluster and verify that the capacity information and sizes are updated as such
	enterpriseKafka := mockkafka.BuildKafkaRequest(
		mockkafka.WithPredefinedTestValues(),
		mockkafka.WithMultiAZ(true),
		mockkafka.With(mockkafka.NAME, "dummy-kafka-to-remain-1"),
		mockkafka.With(mockkafka.STATUS, constants.KafkaRequestStatusAccepted.String()),
		mockkafka.With(mockkafka.BOOTSTRAP_SERVER_HOST, ""),
		mockkafka.With(mockkafka.SIZE_ID, "x2"),
		mockkafka.With(mockkafka.INSTANCE_TYPE, types.STANDARD.String()),
		mockkafka.With(mockkafka.CLUSTER_ID, cluster.Id),
		mockkafka.WithExpiresAt(sql.NullTime{}),
	)

	err = db.Create(enterpriseKafka).Error
	g.Expect(err).ToNot(gomega.HaveOccurred())

	cluster, resp10, err = client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, entCluster.ClusterID)
	if resp10 != nil {
		g.Expect(resp10.StatusCode).To(gomega.Equal(http.StatusOK))
		defer resp10.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster.CapacityInformation).To(gomega.Equal(public.EnterpriseClusterAllOfCapacityInformation{
		KafkaMachinePoolNodeCount:    9,
		MaximumKafkaStreamingUnits:   3,
		RemainingKafkaStreamingUnits: 1,
		ConsumedKafkaStreamingUnits:  2,
	}))

	g.Expect(cluster.SupportedInstanceTypes.InstanceTypes).To(gomega.HaveLen(1))
	standardInstanceType = cluster.SupportedInstanceTypes.InstanceTypes[0]
	g.Expect(standardInstanceType.Id).To(gomega.Equal(types.STANDARD.String()))
	g.Expect(standardInstanceType.SupportedBillingModels).To(gomega.HaveLen(1))
	g.Expect(standardInstanceType.SupportedBillingModels[0].Id).To(gomega.Equal(constants.BillingModelEnterprise.String()))
	g.Expect(standardInstanceType.Sizes).To(gomega.HaveLen(2))
	g.Expect(standardInstanceType.Sizes[0].Id).To(gomega.Equal("x1"))
	g.Expect(standardInstanceType.Sizes[1].Id).To(gomega.Equal("x2"))

	// get cluster addons parameters
	clusterAddonParameters, resp11, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterAddonParameters(adminCtx, entCluster.ClusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(strings.Contains(clusterAddonParameters.Href, entCluster.ClusterID)).To(gomega.BeTrue())
	g.Expect(strings.Contains(clusterAddonParameters.Href, "addon_parameters")).To(gomega.BeTrue())
	g.Expect(clusterAddonParameters.Id).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(clusterAddonParameters.Kind).To(gomega.Equal(objRefMocks.KindClusterAddonParameters))

	g.Expect(len(clusterAddonParameters.FleetshardParameters)).To(gomega.Equal(7))
	for _, param := range clusterAddonParameters.FleetshardParameters {
		g.Expect(param.Id).ToNot(gomega.BeEmpty())
		g.Expect(param.Value).ToNot(gomega.BeEmpty())
	}
	if resp11 != nil {
		g.Expect(resp11.StatusCode).To(gomega.Equal(http.StatusOK))
		defer resp11.Body.Close()
	}
}

func checkEmptyClusterRespValues(cluster public.EnterpriseCluster, g gomega.Gomega) {
	g.Expect(cluster.ClusterId).To(gomega.BeEmpty())
	g.Expect(cluster.Href).To(gomega.BeEmpty())
	g.Expect(cluster.Id).To(gomega.BeEmpty())
	g.Expect(cluster.Kind).To(gomega.BeEmpty())
	g.Expect(cluster.Status).To(gomega.BeEmpty())
}

func checkEmptyClusterAddonParametersRespValues(cluster public.EnterpriseClusterAddonParameters, g gomega.Gomega) {
	g.Expect(cluster.Href).To(gomega.BeEmpty())
	g.Expect(cluster.Id).To(gomega.BeEmpty())
	g.Expect(cluster.Kind).To(gomega.BeEmpty())
}
