package integration

import (
	"context"
	"fmt"
	publicOpenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"gopkg.in/resty.v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestServer struct {
	OcmServer     *httptest.Server
	TearDown      func()
	ClusterID     string
	Token         string
	Client        *publicOpenapi.APIClient
	PrivateClient *openapi.APIClient
	Helper        *test.Helper
	Ctx           context.Context
}

type claimsFunc func(account *v1.Account, clusterId string, h *test.Helper) jwt.MapClaims

func setup(t *testing.T, claims claimsFunc) TestServer {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	startHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = false
	}
	h, client, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)

	clusterId, getClusterErr := utils.GetOsdClusterID(h, t, false)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}

	account := h.NewAllowedServiceAccount()
	ctx := h.NewAuthenticatedContext(account, claims(account, clusterId, h))
	token := h.CreateJWTStringWithClaim(account, claims(account, clusterId, h))

	config := openapi.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", h.AppConfig.Server.BindAddress)
	config.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	privateClient := openapi.NewAPIClient(config)

	return TestServer{
		OcmServer:     ocmServer,
		Client:        client,
		PrivateClient: privateClient,
		TearDown: func() {
			ocmServer.Close()
			tearDown()
		},
		ClusterID: clusterId,
		Token:     token,
		Helper:    h,
		Ctx:       ctx,
	}
}

func TestDataPlaneEndpoints_AuthzSuccess(t *testing.T) {
	clusterId := "test-cluster-id"
	testServer := setup(t, func(account *v1.Account, cid string, h *test.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": h.AppConfig.Keycloak.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": clusterId,
		}
	})

	defer testServer.TearDown()

	body := map[string]openapi.DataPlaneKafkaStatus{
		testServer.ClusterID: {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid
}

func TestDataPlaneEndpoints_AuthzFailWhenNoRealmRole(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *test.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss":                                h.AppConfig.Keycloak.ValidIssuerURI,
			"kas-fleetshard-operator-cluster-id": "test-cluster-id",
		}
	})

	defer testServer.TearDown()

	body := map[string]openapi.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_AuthzFailWhenClusterIdNotMatch(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *test.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": h.AppConfig.Keycloak.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": "different-cluster-id",
		}
	})
	defer testServer.TearDown()

	body := map[string]openapi.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_GetManagedKafkas(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *test.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      h.AppConfig.Keycloak.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	})
	defer testServer.TearDown()

	k := publicOpenapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName1,
		MultiAz:       true,
	}

	_, _, err := testServer.Client.DefaultApi.CreateKafka(testServer.Ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	k.Name = mockKafkaName2
	_, _, err = testServer.Client.DefaultApi.CreateKafka(testServer.Ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	k.Name = mockKafkaName3
	_, _, err = testServer.Client.DefaultApi.CreateKafka(testServer.Ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	list, resp, err := testServer.PrivateClient.DefaultApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(list.Total).To(Equal(int32(3)))
}
