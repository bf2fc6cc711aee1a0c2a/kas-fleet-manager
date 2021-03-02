package integration

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	publicOpenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
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

	var testKafkas = []api.KafkaRequest{
		{
			ClusterID: testServer.ClusterID,
			MultiAZ:   false,
			Name:      mockKafkaName1,
			Status:    constants.KafkaRequestStatusDeprovision.String(),
		},
		{
			ClusterID: testServer.ClusterID,
			MultiAZ:   false,
			Name:      mockKafkaName2,
			Status:    constants.KafkaRequestStatusDeprovision.String(),
		},
		{
			ClusterID: testServer.ClusterID,
			MultiAZ:   false,
			Name:      mockKafkaName3,
			Status:    constants.KafkaRequestStatusReady.String(),
		},
	}

	db := testServer.Helper.Env().DBFactory.New()

	// create dummy kafkas
	for i, k := range testKafkas {
		if err := db.Save(&k).Error; err != nil {
			t.Error("failed to create a dummy kafka request")
			break
		}
		testKafkas[i] = k
	}

	list, resp, err := testServer.PrivateClient.DefaultApi.GetKafkas(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(int(list.Total)).To(Equal(len(testKafkas)))

	find := func(slice []openapi.ManagedKafka, match func(kafka openapi.ManagedKafka) bool) *openapi.ManagedKafka {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}

	for _, k := range testKafkas {
		if mk := find(list.Items, func(item openapi.ManagedKafka) bool { return item.Annotation.Id == k.ID }); mk != nil {
			Expect(mk.Name).To(Equal(k.Name))
			Expect(mk.Annotation.PlacementId).To(Equal(k.PlacementId))
			Expect(mk.Annotation.Id).To(Equal(k.ID))
			Expect(mk.Spec.Deleted).To(Equal(k.Status == constants.KafkaRequestStatusDeprovision.String()))
		} else {
			t.Error("failed matching managedkafka id with kafkarequest id")
			break
		}
	}
}
