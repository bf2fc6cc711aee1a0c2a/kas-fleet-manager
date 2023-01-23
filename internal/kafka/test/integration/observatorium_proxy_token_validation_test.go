package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

// Returns an authenticated context to be used for calling the data plane endpoints
func contextWithValidIssuerAndClientIDclaims(h *coreTest.Helper, clientID string) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolve(&keycloakConfig)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss":      keycloakConfig.SSOProviderRealm().ValidIssuerURI,
		"clientId": clientID,
	}

	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx
}

func TestObservatoriumProxyTokenValidation(t *testing.T) {
	g := gomega.NewWithT(t)

	clusterID := "12345678abcdefgh12345678abcdefgh"
	externalClusterID := "69d631de-9b7f-4bc2-bf4f-4d3295a7b25d"
	otherClusterID := "5678abcdefgh12345678abcdefgh1234"
	otherExternalClusterID := "34a47b83-e09a-42cd-9332-08dd77b84d52"
	nonExistentExternalClusterID := "34a47b83-e09a-42cd-9332-08dd77b84d53"
	invalidExternalClusterID := "invalid"

	cluster1 := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: clusterID,
		}
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.AccessKafkasViaPrivateNetwork = false
		cluster.Status = api.ClusterReady
		cluster.MultiAZ = true
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterID = clusterID
		cluster.ClientID = clusterID
		cluster.ExternalID = externalClusterID
		cluster.DynamicCapacityInfo = api.JSON{}
	})

	cluster2 := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: otherClusterID,
		}
		cluster.ClusterID = otherClusterID
		cluster.ExternalID = otherExternalClusterID
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.OrganizationID = "99999999"
		cluster.ClientID = otherClusterID
		cluster.AccessKafkasViaPrivateNetwork = false
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.Status = api.ClusterReady
		cluster.MultiAZ = true
		cluster.DynamicCapacityInfo = api.JSON{}
	})

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// only run test in integration env
	ocmConfig := test.TestServices.OCMConfig
	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name != environments.IntegrationEnv {
		t.SkipNow()
	}

	db := test.TestServices.DBFactory.New()
	err := db.Create(cluster1).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(cluster2).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	privateClient := test.NewPrivateAPIClient(h)

	clusterIDCtx := contextWithValidIssuerAndClientIDclaims(h, cluster1.ClientID)

	// fail without valid token in the context
	resp, err := privateClient.ObservatoriumProxyApi.VerifyObservatoriumProxyRequestValid(context.Background(), externalClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	if resp != nil {
		g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
		defer resp.Body.Close()
	}
	// fail with invalid external ID
	resp2, err := privateClient.ObservatoriumProxyApi.VerifyObservatoriumProxyRequestValid(clusterIDCtx, invalidExternalClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	if resp2 != nil {
		g.Expect(resp2.StatusCode).To(gomega.Equal(http.StatusBadRequest))
		defer resp2.Body.Close()
	}

	// fail with non-existent external ID
	resp3, err := privateClient.ObservatoriumProxyApi.VerifyObservatoriumProxyRequestValid(clusterIDCtx, nonExistentExternalClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	if resp3 != nil {
		g.Expect(resp3.StatusCode).To(gomega.Equal(http.StatusNotFound))
		defer resp3.Body.Close()
	}

	// fail with non-matching external ID against clientID from the context
	resp4, err := privateClient.ObservatoriumProxyApi.VerifyObservatoriumProxyRequestValid(clusterIDCtx, otherExternalClusterID)
	g.Expect(err).To(gomega.HaveOccurred())
	if resp4 != nil {
		g.Expect(resp4.StatusCode).To(gomega.Equal(http.StatusForbidden))
		defer resp4.Body.Close()
	}

	// success
	resp5, err := privateClient.ObservatoriumProxyApi.VerifyObservatoriumProxyRequestValid(clusterIDCtx, externalClusterID)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	if resp5 != nil {
		g.Expect(resp5.StatusCode).To(gomega.Equal(http.StatusOK))
		defer resp5.Body.Close()
	}
}
