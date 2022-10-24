package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	mockclusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"

	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

type TestEnv struct {
	helper   *coreTest.Helper
	client   *public.APIClient
	teardown func()
}

func termsRequiredSetup(termsRequired bool, t *testing.T) TestEnv {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	termsReviewResponse, err := mocks.GetMockTermsReviewBuilder(nil).TermsRequired(termsRequired).Build()
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetTermsReviewPostResponse(termsReviewResponse, nil)
	ocmServer := ocmServerBuilder.Build()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, func(serverConfig *server.ServerConfig, c *config.DataplaneClusterConfig) {
		serverConfig.EnableTermsAcceptance = true
	})

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	cluster := mockclusters.BuildCluster(func(cluster *api.Cluster) {
		cluster.Meta = api.Meta{
			ID: api.NewID(),
		}
		cluster.ProviderType = api.ClusterProviderStandalone
		cluster.SupportedInstanceType = api.AllInstanceTypeSupport.String()
		cluster.ClientID = "some-client-id"
		cluster.ClientSecret = "some-client-secret"
		cluster.ClusterID = api.NewID()
		cluster.Region = mocks.MockCluster.Region().ID()
		cluster.CloudProvider = mocks.MockCluster.CloudProvider().ID()
		cluster.MultiAZ = true
		cluster.Status = api.ClusterReady
		cluster.ProviderSpec = api.JSON{}
		cluster.ClusterSpec = api.JSON{}
		cluster.IdentityProviderID = "some-identity-provider-id"
	})

	db := h.DBFactory().New()
	err = db.Create(cluster).Error
	if err != nil {
		t.Fatalf(err.Error(), "failed to create data plane cluster")
	}

	return TestEnv{
		helper: h,
		client: client,
		teardown: func() {
			ocmServer.Close()
			tearDown()
		},
	}
}

func TestTermsRequired_CreateKafkaTermsRequired(t *testing.T) {
	g := gomega.NewWithT(t)

	env := termsRequiredSetup(true, t)
	defer env.teardown()

	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// setup pre-requisites to performing requests
	account := env.helper.NewRandAccount()
	ctx := env.helper.NewAuthenticatedContext(account, nil)

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	_, resp, err := env.client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}

	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))
}

func TestTermsRequired_CreateKafka_TermsNotRequired(t *testing.T) {
	g := gomega.NewWithT(t)

	env := termsRequiredSetup(false, t)
	defer env.teardown()

	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// setup pre-requisites to performing requests
	account := env.helper.NewRandAccount()
	ctx := env.helper.NewAuthenticatedContext(account, nil)

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
	}

	_, resp, err := env.client.DefaultApi.CreateKafka(ctx, true, k)
	if resp != nil {
		resp.Body.Close()
	}

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
}

func TestTermsRequired_ListKafkaTermsRequired(t *testing.T) {
	g := gomega.NewWithT(t)

	env := termsRequiredSetup(true, t)
	defer env.teardown()

	// setup pre-requisites to performing requests
	account := env.helper.NewRandAccount()
	ctx := env.helper.NewAuthenticatedContext(account, nil)

	_, resp, err := env.client.DefaultApi.GetKafkas(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}
