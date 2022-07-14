package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/bxcodec/faker/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
	"gopkg.in/resty.v1"
)

func TestAuth_success(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(h.CreateJWTString(serviceAccount)).
		Get(h.RestURL("/"))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusOK))

}

func TestAuthSucess_publicUrls(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		Get(h.RestURL("/"))
	g.Expect(err).To(gomega.BeNil())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusOK))

	errorsList, resp, err := client.ErrorsApi.GetErrors(context.Background())
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.BeNil())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(errorsList.Items).NotTo(gomega.BeEmpty())

	errorCode := "7"
	_, notFoundErrorResp, err := client.ErrorsApi.GetErrorById(context.Background(), errorCode)

	if notFoundErrorResp != nil {
		notFoundErrorResp.Body.Close()
	}
	g.Expect(err).To(gomega.BeNil())
	g.Expect(notFoundErrorResp.StatusCode).To(gomega.Equal(http.StatusOK))
}

func TestAuthSuccess_usingSSORHToken(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"username":           nil,
		"preferred_username": serviceAccount.Username(),
	}

	token := h.CreateJWTStringWithClaim(serviceAccount, claims)

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusOK))
}

func TestAuthFailure_withoutToken(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Request doesn't contain the 'Authorization' header or the 'cs_jwt' cookie"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_invalidTokenWithInvalidTyp(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"typ": "Invalid",
	}

	token := h.CreateJWTStringWithClaim(serviceAccount, claims)

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Bearer token type 'Invalid' isn't allowed"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_ExpiredToken(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Duration(-15) * time.Minute).Unix(),
	}

	token := h.CreateJWTStringWithClaim(serviceAccount, claims)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Bearer token is expired"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_invalidTokenMissingIat(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"iat": nil,
	}

	token := h.CreateJWTStringWithClaim(serviceAccount, claims)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Bearer token doesn't contain required claim 'iat'"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_invalidTokenMissingAlgHeader(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"username":   serviceAccount.Username(),
		"first_name": serviceAccount.FirstName(),
		"last_name":  serviceAccount.LastName(),
		"typ":        "Bearer",
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Minute * time.Duration(auth.TokenExpMin)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = auth.JwkKID
	token.Header["alg"] = ""
	strToken, _ := token.SignedString(h.JWTPrivateKey)

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(strToken).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())

	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Bearer token can't be verified"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_invalidTokenUnsigned(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelper(t, ocmServer)
	serviceAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "13640203")
	defer teardown()
	claims := jwt.MapClaims{
		"username":   serviceAccount.Username(),
		"first_name": serviceAccount.FirstName(),
		"last_name":  serviceAccount.LastName(),
		"typ":        "Bearer",
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Minute * time.Duration(auth.TokenExpMin)).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = auth.JwkKID
	strToken := token.Raw

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(strToken).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).To(gomega.BeNil())
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Request doesn't contain the 'Authorization' header or the 'cs_jwt' cookie"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func TestAuthFailure_usingMasSsoTokenOnKafkasGet(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(keycloakConfig *keycloak.KeycloakConfig) {
		keycloakConfig.SelectSSOProvider = keycloak.MAS_SSO
	})
	defer teardown()

	// create a mas-sso service account token
	orgId := "13640203"
	masSsoSA := h.NewAccount("service-account-srvc-acct-1", "", "", orgId)
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)
	claims := jwt.MapClaims{
		"iss":                keycloakConfig.SSOProviderRealm().ValidIssuerURI,
		"rh-org-id":          orgId,
		"rh-user-id":         masSsoSA.ID(),
		"preferred_username": masSsoSA.Username(),
	}

	token := h.CreateJWTStringWithClaim(masSsoSA, claims)

	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token).
		Get(h.RestURL("/kafkas"))
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
	re := parseResponse(restyResp)
	g.Expect(re.Code).To(gomega.Equal(fmt.Sprintf("%s-%d", errors.ERROR_CODE_PREFIX, errors.ErrorUnauthenticated)))
	g.Expect(re.Reason).To(gomega.Equal("Account authentication could not be verified"))
	g.Expect(restyResp.StatusCode()).To(gomega.Equal(http.StatusUnauthorized))
}

func parseResponse(restyResp *resty.Response) public.Error {
	var re public.Error
	if err := json.Unmarshal(restyResp.Body(), &re); err != nil {
		panic(err)
	}
	return re
}
