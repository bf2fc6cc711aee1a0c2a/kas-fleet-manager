package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
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

	dynamicCapacityInfoString := "{\"standard\":{\"max_nodes\":1,\"max_units\":3,\"remaining_units\":3}}"

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
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)
	if resp != nil {
		g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp.Body.Close()
	}

	clusterWithAddonParameters, resp2, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(noReqClaimsCtx, entCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterWithAddonParametersRespValues(clusterWithAddonParameters, g)
	if resp2 != nil {
		g.Expect(resp2.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp2.Body.Close()
	}

	// non-admin user of the same org trying to access addon params
	clusterWithAddonParameters, resp3, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(nonAdminCtx, entCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterWithAddonParametersRespValues(clusterWithAddonParameters, g)
	if resp3 != nil {
		g.Expect(resp3.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp3.Body.Close()
	}

	// non-existent cluster
	cluster, resp4, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, validNonExistentClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)
	if resp4 != nil {
		g.Expect(resp4.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp4.Body.Close()
	}

	clusterWithAddonParameters, resp5, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(adminCtx, validNonExistentClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterWithAddonParametersRespValues(clusterWithAddonParameters, g)
	if resp5 != nil {
		g.Expect(resp5.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp5.Body.Close()
	}

	// enterprise cluster from different org
	cluster, resp6, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, otherOrgCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)
	if resp6 != nil {
		g.Expect(resp6.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp6.Body.Close()
	}

	clusterWithAddonParameters, resp7, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(adminCtx, otherOrgCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterWithAddonParametersRespValues(clusterWithAddonParameters, g)
	if resp7 != nil {
		g.Expect(resp7.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp7.Body.Close()
	}

	// non-enterprise cluster from same org
	cluster, resp8, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, nonEntCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterRespValues(cluster, g)
	if resp8 != nil {
		g.Expect(resp8.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp8.Body.Close()
	}

	clusterWithAddonParameters, resp9, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(adminCtx, nonEntCluster.ClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	checkEmptyClusterWithAddonParametersRespValues(clusterWithAddonParameters, g)
	if resp9 != nil {
		g.Expect(resp9.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp9.Body.Close()
	}

	// success
	cluster, resp10, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterById(nonAdminCtx, entCluster.ClusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cluster.ClusterId).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(strings.Contains(cluster.Href, entCluster.ClusterID)).To(gomega.BeTrue())
	g.Expect(cluster.Id).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(cluster.Kind).To(gomega.Equal(objRefMocks.KindCluster))
	g.Expect(cluster.Status).To(gomega.Equal(entCluster.Status.String()))
	g.Expect(cluster.AccessKafkasViaPrivateNetwork).To((gomega.Equal(entCluster.AccessKafkasViaPrivateNetwork)))
	if resp10 != nil {
		g.Expect(resp10.StatusCode).To(gomega.Equal(http.StatusOK))
		defer resp10.Body.Close()
	}

	clusterWithAddonParameters, resp11, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseClusterWithAddonParameters(adminCtx, entCluster.ClusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(clusterWithAddonParameters.ClusterId).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(strings.Contains(clusterWithAddonParameters.Href, entCluster.ClusterID)).To(gomega.BeTrue())
	g.Expect(clusterWithAddonParameters.Id).To(gomega.Equal(entCluster.ClusterID))
	g.Expect(clusterWithAddonParameters.Kind).To(gomega.Equal(objRefMocks.KindCluster))
	g.Expect(clusterWithAddonParameters.Status).To(gomega.Equal(entCluster.Status.String()))
	g.Expect(clusterWithAddonParameters.AccessKafkasViaPrivateNetwork).To((gomega.Equal(entCluster.AccessKafkasViaPrivateNetwork)))
	g.Expect(len(clusterWithAddonParameters.FleetshardParameters)).To(gomega.Equal(7))
	for _, param := range clusterWithAddonParameters.FleetshardParameters {
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

func checkEmptyClusterWithAddonParametersRespValues(cluster public.EnterpriseClusterWithAddonParameters, g gomega.Gomega) {
	g.Expect(cluster.ClusterId).To(gomega.BeEmpty())
	g.Expect(cluster.Href).To(gomega.BeEmpty())
	g.Expect(cluster.Id).To(gomega.BeEmpty())
	g.Expect(cluster.Kind).To(gomega.BeEmpty())
	g.Expect(cluster.Status).To(gomega.BeEmpty())
}
