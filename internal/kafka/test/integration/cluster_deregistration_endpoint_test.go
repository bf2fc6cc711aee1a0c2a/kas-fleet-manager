package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
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
	org := amsv1.NewOrganization().ID("some-org-id").Capabilities(amsv1.NewCapability().Name("s").Value("s"))
	orgList, err := amsv1.NewOrganizationList().Items(org).Build()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetOrganizationsGetResponse(orgList, nil)
	subscriptions, err := amsv1.NewSubscriptionList().Items(amsv1.NewSubscription()).Build()
	g.Expect(err).ToNot(gomega.HaveOccurred())
	ocmServerBuilder.SetSubscriptionSearchResponse(subscriptions, nil)
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	ocmConfig := test.TestServices.OCMConfig
	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name != environments.IntegrationEnv {
		t.SkipNow()
	}

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

	id1 := "12345678901234567890123456789012"
	id2 := "22345678901234567890123456789012"
	id3 := "32345678901234567890123456789012"
	nonExistentID := "42345678901234567890123456789012"

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
		cluster.ClusterDNS = "apps.example.com"
		cluster.ExternalID = "69d631de-9b7f-4bc2-bf4f-4d3295a7b25e"
	})

	registrationPayload := public.EnterpriseOsdClusterPayload{
		ClusterId:                     anotherEntCluster.ClusterID,
		ClusterIngressDnsName:         anotherEntCluster.ClusterDNS,
		KafkaMachinePoolNodeCount:     12,
		AccessKafkasViaPrivateNetwork: true,
	}

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

	err = db.Create(entCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(anotherEntCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(nonEntCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(kafka).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// test deregistration scenarios
	// async=false deregistration non enterprise cluster
	_, resp, e := client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, false, id1)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint with async=false")
	closeRespBody(resp)

	// no org id account deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(nonadminCtx, true, id1)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint with no org ID in the token")
	closeRespBody(resp)

	// not found cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, nonExistentID)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to call deregister endpoint against non-existent cluster")
	closeRespBody(resp)

	// other org cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(otherAdminadminCtx, true, id1)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister enterprise cluster from another org")
	closeRespBody(resp)

	// non enterprise cluster deregistration non enterprise cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id3)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister non-enterprise cluster")
	closeRespBody(resp)

	// should fail when kafkas are on the cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id1)
	g.Expect(e).To(gomega.HaveOccurred(), "error should be thrown when attempting to deregister enterprise cluster with kafkas on it")
	closeRespBody(resp)

	// should succeed with empty cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.DeleteEnterpriseClusterById(adminCtx, true, id2)
	g.Expect(e).NotTo(gomega.HaveOccurred(), "error should not be thrown when attempting to deregister enterprise cluster with no kafkas on it")
	closeRespBody(resp)

	// wait for the cluster to be removed
	e = common.WaitForClusterToBeDeleted(test.TestServices.DBFactory, &test.TestServices.ClusterService, anotherEntCluster.ClusterID)
	g.Expect(e).NotTo(gomega.HaveOccurred(), "error should not be thrown when waiting for cluster to be deleted")

	// re-registration should succeed for previously deleted cluster
	_, resp, e = client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(adminCtx, registrationPayload)
	g.Expect(e).NotTo(gomega.HaveOccurred(), "error should not be thrown when attempting to re-register a previously deleted cluster")
	closeRespBody(resp)
}

func closeRespBody(resp *http.Response) {
	if resp != nil {
		resp.Body.Close()
	}
}
