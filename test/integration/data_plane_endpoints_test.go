package integration

import (
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/private/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"gopkg.in/resty.v1"
	"net/http"
	"testing"
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
	token := h.CreateJWTStringWithAdditionalClaims(account, claims)
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
	token := h.CreateJWTStringWithAdditionalClaims(account, claims)
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
	token := h.CreateJWTStringWithAdditionalClaims(account, claims)
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
