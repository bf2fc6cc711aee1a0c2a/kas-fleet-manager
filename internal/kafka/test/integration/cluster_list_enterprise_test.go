package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	clusterMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	kafkaMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/onsi/gomega"
)

func TestEnterpriseClustersList(t *testing.T) {
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

	account := h.NewRandAccount()
	authCtx := h.NewAuthenticatedContext(account, nil)

	nonAuthAccount := h.NewAccount("", "", "", "")
	nonAuthCtx := h.NewAuthenticatedContext(nonAuthAccount, nil)

	cluster := clusterMocks.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.OrganizationID = kafkaMocks.DefaultOrganisationId
		cluster.ClusterType = api.EnterpriseDataPlaneClusterType.String()
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.ClusterID = api.NewID()
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
	})

	db := test.TestServices.DBFactory.New()
	err := db.Create(cluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = db.Create(otherOrgCluster).Error
	g.Expect(err).NotTo(gomega.HaveOccurred())

	clusterListNonAuth, resp, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseOsdClusters(nonAuthCtx)
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(len(clusterListNonAuth.Items)).To(gomega.Equal(0))
	if resp != nil {
		defer resp.Body.Close()
	}

	// only return clusters belonging to the org of the user
	clusterList, resp2, err := client.EnterpriseDataplaneClustersApi.GetEnterpriseOsdClusters(authCtx)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(clusterList).ToNot((gomega.BeNil()))
	g.Expect(len(clusterList.Items)).To(gomega.Equal(1))
	g.Expect(clusterList.Size).To(gomega.Equal(int32(1)))
	g.Expect(clusterList.Items[0].Status).To(gomega.Equal(api.ClusterReady.String()))
	g.Expect(resp2).ToNot((gomega.BeNil()))
	if resp2 != nil {
		defer resp2.Body.Close()
	}
}
