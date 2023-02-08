package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	kafkatest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	mockclusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"

	"github.com/onsi/gomega"
)

// Test cluster registration endpoint with invalid body
func TestClusterRegistration_BadRequest(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := kafkatest.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// setup pre-requisites for performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	testCases := []struct {
		name   string
		assert func()
	}{
		{
			name: "should return bad request when cluster id is not valid",
			assert: func() {
				payload := public.EnterpriseOsdClusterPayload{
					ClusterId:                     "invalid",
					ClusterIngressDnsName:         "apps.example.com",
					KafkaMachinePoolNodeCount:     3,
					AccessKafkasViaPrivateNetwork: false,
				}
				_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
				if resp != nil {
					resp.Body.Close()
				}
				g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
			},
		},
		{
			name: "should return bad request when cluster ingress dns name is not valid",
			assert: func() {
				payload := public.EnterpriseOsdClusterPayload{
					ClusterId:                     "1234abcd1234abcd1234abcd1234abcd",
					ClusterIngressDnsName:         "appsexamplecom",
					KafkaMachinePoolNodeCount:     3,
					AccessKafkasViaPrivateNetwork: true,
				}
				_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
				if resp != nil {
					resp.Body.Close()
				}
				g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
			},
		},
		{
			name: "should return bad request when kafka machine pool node count is less than 3",
			assert: func() {
				payload := public.EnterpriseOsdClusterPayload{
					ClusterId:                     "1234abcd1234abcd1234abcd1234abcd",
					ClusterIngressDnsName:         "apps.example.com",
					KafkaMachinePoolNodeCount:     2,
					AccessKafkasViaPrivateNetwork: true,
				}
				_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
				if resp != nil {
					resp.Body.Close()
				}
				g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
			},
		},
		{
			name: "should return bad request when kafka machine pool node count is not a multiple of 3",
			assert: func() {
				payload := public.EnterpriseOsdClusterPayload{
					ClusterId:                     "1234abcd1234abcd1234abcd1234abcd",
					ClusterIngressDnsName:         "apps.example.com",
					KafkaMachinePoolNodeCount:     20,
					AccessKafkasViaPrivateNetwork: true,
				}
				_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
				if resp != nil {
					resp.Body.Close()
				}
				g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
			},
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			testcase.assert()
		})
	}
}

// Test that an error is returned when an organization is not allowed to register a cluster
// TODO this has to be replaced with quota checks
func TestClusterRegistration_UnauthorizedTest(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := kafkatest.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// create an account in the organization id that is not allowed to regiser a cluster
	account := h.NewAccount("some-account", "some-name", "some@account.test", "org-id-that-is-not-allowed")
	ctx := h.NewAuthenticatedContext(account, nil)

	payload := public.EnterpriseOsdClusterPayload{
		ClusterId:                     "1234abcd1234abcd1234abcd1234abcd",
		ClusterIngressDnsName:         "apps.example.com",
		KafkaMachinePoolNodeCount:     3,
		AccessKafkasViaPrivateNetwork: false,
	}
	_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
}

// Test that the cluster id is unique
func TestClusterRegistration_ClusterIDUniquenessChecks(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := kafkatest.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// seed a cluster with the cluster id
	clusterID := "1234abcd1234abcd1234abcd1234abcd"
	existingCluster := mockclusters.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = api.AllInstanceTypeSupport.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = clusterID
		cluster.Region = afEast1Region
		cluster.AccessKafkasViaPrivateNetwork = true
		cluster.CloudProvider = gcp
		cluster.MultiAZ = true
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.IdentityProviderID = "some-identity-provider-id"
	})

	if err := test.TestServices.DBFactory.New().Create(existingCluster).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	// attempt to register a cluster with the same id should fail
	payload := public.EnterpriseOsdClusterPayload{
		ClusterId:                     clusterID,
		ClusterIngressDnsName:         "apps.example.com",
		KafkaMachinePoolNodeCount:     3,
		AccessKafkasViaPrivateNetwork: true,
	}

	_, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
	if resp != nil {
		resp.Body.Close()
	}

	g.Expect(err).To(gomega.HaveOccurred(), "error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusConflict))
}

func TestClusterRegistration_Successful(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := kafkatest.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	// only run test in integration env
	ocmConfig := test.TestServices.OCMConfig
	if ocmConfig.MockMode != ocm.MockModeEmulateServer || h.Env.Name != environments.IntegrationEnv {
		t.SkipNow()
	}

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, claims)

	payload := public.EnterpriseOsdClusterPayload{
		ClusterId:                     "1234abcd1234abcd1234abcd1234abcd",
		ClusterIngressDnsName:         "apps.example.com",
		KafkaMachinePoolNodeCount:     12,
		AccessKafkasViaPrivateNetwork: true,
	}

	enterpriseCluster, resp, err := client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
	if resp != nil {
		defer resp.Body.Close()
	}

	g.Expect(err).ToNot(gomega.HaveOccurred(), "error posting object:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(enterpriseCluster.ClusterId).To(gomega.Equal(payload.ClusterId))
	g.Expect(enterpriseCluster.Status).To(gomega.Equal(api.ClusterAccepted.String()))
	g.Expect(enterpriseCluster.AccessKafkasViaPrivateNetwork).To(gomega.BeTrue())
	g.Expect(enterpriseCluster.FleetshardParameters).To(gomega.HaveLen(7))
	for _, parameter := range enterpriseCluster.FleetshardParameters {
		g.Expect(parameter.Id).ToNot(gomega.BeEmpty())
		g.Expect(parameter.Value).ToNot(gomega.BeEmpty())
	}

	cluster, err := test.TestServices.ClusterService.FindClusterByID(enterpriseCluster.ClusterId)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "error posting object:  %v", err)
	g.Expect(payload.ClusterIngressDnsName).To(gomega.Equal(cluster.ClusterDNS))
	g.Expect(api.StandardTypeSupport.String()).To(gomega.Equal(cluster.SupportedInstanceType))
	g.Expect(api.EnterpriseDataPlaneClusterType.String()).To(gomega.Equal(cluster.ClusterType))
	g.Expect(api.ClusterProviderOCM).To(gomega.Equal(cluster.ProviderType))
	g.Expect(cluster.ExternalID).ToNot(gomega.BeEmpty())
	g.Expect(cluster.Region).ToNot(gomega.BeEmpty())
	g.Expect(cluster.CloudProvider).ToNot(gomega.BeEmpty())

	dynamicScalingInfo := cluster.RetrieveDynamicCapacityInfo()
	g.Expect(payload.KafkaMachinePoolNodeCount).To(gomega.Equal(dynamicScalingInfo[api.StandardTypeSupport.String()].MaxNodes))

	// waiting for cluster state to become `waiting_for_kas_fleetshard_operator`, so that its persisted struct can be updated after terraforming phase
	_, checkWaitingForKasFleetshardOperatorErr := common.WaitForClusterStatus(test.TestServices.DBFactory, &test.TestServices.ClusterService, cluster.ClusterID, api.ClusterWaitingForKasFleetShardOperator)
	g.Expect(checkWaitingForKasFleetshardOperatorErr).NotTo(gomega.HaveOccurred(), "Error waiting for cluster to reach waiting for fleetshard status: %s %v", cluster.ClusterID, checkWaitingForKasFleetshardOperatorErr)

	// observe that capacity info is there
	privateAPIClient := test.NewPrivateAPIClient(h)
	privateAPICtx, err := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(h, cluster.ClusterID)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	managedKafkaAgentCR, resp, err := privateAPIClient.AgentClustersApi.GetKafkaAgent(privateAPICtx, cluster.ClusterID)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	capacity, ok := managedKafkaAgentCR.Spec.Capacity[cluster.SupportedInstanceType]
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(capacity.MaxNodes).To(gomega.Equal(int32(payload.KafkaMachinePoolNodeCount)))
	g.Expect(managedKafkaAgentCR.Spec.Net.Private).To(gomega.BeTrue())

	// register another cluster with AccessKafkasViaPrivateNetwork set to false and verify that it is set to false
	payload = public.EnterpriseOsdClusterPayload{
		ClusterId:                     "1234abcd1234abcd1234abcd1234abce",
		ClusterIngressDnsName:         "apps.example2.com",
		KafkaMachinePoolNodeCount:     9,
		AccessKafkasViaPrivateNetwork: false,
	}

	enterpriseCluster, resp, err = client.EnterpriseDataplaneClustersApi.RegisterEnterpriseOsdCluster(ctx, payload)
	if resp != nil {
		defer resp.Body.Close()
	}

	g.Expect(err).ToNot(gomega.HaveOccurred(), "error posting object:  %v", err)
	g.Expect(enterpriseCluster.AccessKafkasViaPrivateNetwork).To(gomega.BeFalse())
}
