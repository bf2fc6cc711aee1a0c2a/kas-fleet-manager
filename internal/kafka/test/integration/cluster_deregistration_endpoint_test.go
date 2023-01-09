package integration

import (
	"net/http"
	"testing"

	"github.com/antihax/optional"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"

	"github.com/onsi/gomega"
)

func TestEnterpriseClusterDeregistration(t *testing.T) {
	g := gomega.NewWithT(t)
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	adminAccount := h.NewAccountWithNameAndOrg("admin", kafkaMocks.DefaultOrganisationId)

	otherAdminAccount := h.NewAccountWithNameAndOrg("other-admin", "98765432")

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	otherAdminadminCtx := h.NewAuthenticatedContext(otherAdminAccount, claims)

	nonAuthAccount := h.NewAccount("", "", "", "")
	nonadminCtx := h.NewAuthenticatedContext(nonAuthAccount, nil)

	// setup pre-requisites to performing requests
	db := test.TestServices.DBFactory.New()

	id1 := api.NewID()
	id2 := api.NewID()
	id3 := api.NewID()
	nonExistentID := api.NewID()

	entCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: id1,
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = "standard"
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = id1
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.IdentityProviderID = "some-identity-provider"
	})

	anotherEntCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: id2,
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = "standard"
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = id2
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.IdentityProviderID = "some-identity-provider"
	})

	nonEntCluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: id3,
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = "standard"
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = id3
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.IdentityProviderID = "some-identity-provider"
	})

	kafka := kafkaMocks.BuildKafkaRequest(kafkaMocks.WithPredefinedTestValues(), func(kr *dbapi.KafkaRequest) {
		kr.Meta = api.Meta{
			ID: api.NewID(),
		}
		kr.Region = cluster.Region
		kr.CloudProvider = cluster.CloudProvider
		kr.SizeId = "x1"
		kr.InstanceType = types.STANDARD.String()
		kr.ClusterID = id1
		kr.DesiredKafkaBillingModel = "enterprise"
		kr.ActualKafkaBillingModel = "enterprise"
	})

	err := db.Create(entCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(anotherEntCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(nonEntCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(kafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	forceTrue := public.DeleteEnterpriseClusterByIdOpts{Force: optional.NewBool(true)}
	forceFalse := public.DeleteEnterpriseClusterByIdOpts{Force: optional.NewBool(false)}

	// test deregistration scenarios
	// async=false deregistration non enterprise cluster
	_, resp, e := client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, false, id1, &forceFalse)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint with async=false")
	closeRespBody(resp)

	// no org id account deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(nonadminCtx, true, id1, &forceTrue)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint with no org ID in the token")
	closeRespBody(resp)

	// not found cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, nonExistentID, &forceTrue)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint against non-existent cluster")
	closeRespBody(resp)

	// other org cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(otherAdminadminCtx, true, id1, &forceTrue)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister enterprise cluster from another org")
	closeRespBody(resp)

	// non enterprise cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id3, &forceTrue)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister non-enterprise cluster")
	closeRespBody(resp)

	// force false with kafkas on the cluster failure
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id1, &forceFalse)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister enterprise cluster with kafkas on it without force=true")
	closeRespBody(resp)

	// force false with empty cluster success
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id2, &forceFalse)
	g.Expect(e).NotTo(gomega.HaveOccurred(), "error should not be thrown when attempting to deregister enterprise cluster with no kafkas on it without force=true")
	closeRespBody(resp)

	// force true with non-empty cluster success
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id1, &forceTrue)
	g.Expect(e).NotTo(gomega.HaveOccurred(), "error should not be thrown when attempting to deregister enterprise cluster with kafkas on it with force=true")
	closeRespBody(resp)
}

func closeRespBody(resp *http.Response) {
	if resp != nil {
		resp.Body.Close()
	}
}
