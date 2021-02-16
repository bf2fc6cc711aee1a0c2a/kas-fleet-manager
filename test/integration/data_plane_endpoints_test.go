package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"
)

func TestDataPlaneEndpoints_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	startHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = false
	}
	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	clusterId := "test-cluster-id"

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": clusterId,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	body := map[string]openapi.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(body).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(clusterStatusUpdateRequest).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid
}

func TestDataPlaneEndpoints_AuthzFailWhenNoRealmRole(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	startHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = false
	}
	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	clusterId := "test-cluster-id"

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"kas-fleetshard-operator-cluster-id": clusterId,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	body := map[string]openapi.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(body).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(clusterStatusUpdateRequest).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_AuthzFailWhenClusterIdNotMatch(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	startHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = true
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.ClusterCreationConfig.EnableKasFleetshardOperator = false
	}
	h, _, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	clusterId := "test-cluster-id"

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": "different-cluster-id",
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	body := map[string]openapi.DataPlaneKafkaStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(body).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/kafkas/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := openapi.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		SetBody(clusterStatusUpdateRequest).
		Put(h.RestURL("/agent-clusters/" + clusterId + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}
