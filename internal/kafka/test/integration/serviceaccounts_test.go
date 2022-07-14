package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/antihax/optional"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	kafkatest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"

	"github.com/bxcodec/faker/v3"
	"github.com/onsi/gomega"
)

func getAuthenticatedContext(h *kafkatest.Helper, claims jwt.MapClaims) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolve(&keycloakConfig)

	// If REDHAT_SSO, is used, we need to use a token retrieved from the actual redhat sso instance
	if keycloakConfig.SelectSSOProvider == keycloak.REDHAT_SSO {
		return h.NewAuthenticatedContextForSSO(keycloakConfig)
	}

	// If MAS_SSO is used, we need to use a token with the mocked claims as
	// it contains properties in the claims like org_id to be set for authorization
	account := h.NewRandAccount()
	return h.NewAuthenticatedContext(account, claims)
}

func TestServiceAccounts_GetByClientID(t *testing.T) {
	g := gomega.NewWithT(t)
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	ctx := getAuthenticatedContext(h, nil)

	opts := public.GetServiceAccountsOpts{
		ClientId: optional.NewString("srvc-acct-12345678-1234-1234-1234-123456789012"),
	}

	list, resp, err := client.SecurityApi.GetServiceAccounts(ctx, &opts)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(list.Items).To(gomega.HaveLen(0))

	// Create one Service Account
	r := public.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))

	// Find service account by clientid
	opts.ClientId = optional.NewString(sa.ClientId)

	list, resp, err = client.SecurityApi.GetServiceAccounts(ctx, &opts)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(list.Items).To(gomega.HaveLen(1))
	g.Expect(list.Items[0].ClientId == sa.ClientId)
	g.Expect(list.Items[0].Id == sa.Id)
	g.Expect(list.Items[0].Name == sa.Name)
	_, resp, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func TestServiceAccounts_Success(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// get username from the access token to verify service account owner
	ctx := getAuthenticatedContext(h, nil)
	accessTokenValue := fmt.Sprintf("%s", ctx.Value(compat.ContextAccessToken))
	g.Expect(accessTokenValue).ToNot(gomega.BeEmpty(), "failed to get access token from context")

	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(accessTokenValue, claims)
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to parse access token")
	g.Expect(claims).ToNot(gomega.BeEmpty(), "access token does not have any claims")

	kfmClaims := auth.KFMClaims(claims)
	username, err := kfmClaims.GetUsername()
	g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to get username from claims")
	g.Expect(username).ToNot(gomega.BeEmpty(), "username is empty in claims")

	//verify list
	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	currTime := time.Now().Format(time.RFC3339)
	createdAt, _ := time.Parse(time.RFC3339, currTime)

	//verify create
	r := public.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(sa.ClientId).NotTo(gomega.BeEmpty())
	g.Expect(sa.ClientSecret).NotTo(gomega.BeEmpty())
	g.Expect(sa.Id).NotTo(gomega.BeEmpty())
	g.Expect(sa.CreatedAt).Should(gomega.BeTemporally(">=", createdAt))
	g.Expect(sa.CreatedBy).Should(gomega.Equal(username))

	// verify get by id
	id := sa.Id
	sa, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(sa.ClientId).NotTo(gomega.BeEmpty())
	g.Expect(sa.Id).NotTo(gomega.BeEmpty())
	g.Expect(sa.CreatedAt).Should(gomega.BeTemporally(">=", createdAt))
	g.Expect(sa.CreatedBy).Should(gomega.Equal(username))

	//verify reset
	oldSecret := sa.ClientSecret
	sa, resp, err = client.SecurityApi.ResetServiceAccountCreds(ctx, id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(sa.ClientSecret).NotTo(gomega.BeEmpty())
	g.Expect(sa.ClientSecret).NotTo(gomega.Equal(oldSecret))
	g.Expect(sa.CreatedAt).Should(gomega.BeTemporally(">=", createdAt))
	g.Expect(sa.CreatedBy).Should(gomega.Equal(username))

	//verify delete
	_, resp, err = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// verify deletion of non-existent service account throws http status code 404
	_, resp, err = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred())
	// skipping for now as mas sso and rhsso service returns different http codes: mas sso returns 404, rhsso returns 500.
	// g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))

	f := false
	accounts, resp, _ := client.SecurityApi.GetServiceAccounts(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	for _, a := range accounts.Items {
		if a.Id == id {
			f = true
		}
	}
	g.Expect(f).To(gomega.BeFalse())
}

func TestServiceAccounts_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      "invalidiss",
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
}

func TestServiceAccounts_CorrectTokenIssuer_AuthzSuccess(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	ctx := getAuthenticatedContext(h, nil)

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
}

func TestServiceAccounts_InputValidation(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	ctx := getAuthenticatedContext(h, nil)

	//length check
	r := public.ServiceAccountRequest{
		Name:        "length-more-than-50-is-not-allowed-managed-service-integration-test",
		Description: "created by the managed service integration",
	}
	_, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "<script>alert(\"TEST\");</script>",
		Description: "created by the managed service integration",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	//description length can not be more than 255
	r = public.ServiceAccountRequest{
		Name:        "test-svc-1",
		Description: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	//min length required for name
	r = public.ServiceAccountRequest{
		Name:        "",
		Description: "test",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolve(&keycloakConfig)

	if keycloakConfig.SelectSSOProvider == keycloak.MAS_SSO {
		// min length required is not required for desc
		// only test this for mas sso. redhat sso requires the description to not be empty
		r = public.ServiceAccountRequest{
			Name:        "test",
			Description: "",
		}
		sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
		if resp != nil {
			resp.Body.Close()
		}
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
		g.Expect(sa.ClientId).NotTo(gomega.BeEmpty())
		g.Expect(sa.ClientSecret).NotTo(gomega.BeEmpty())
		g.Expect(sa.Id).NotTo(gomega.BeEmpty())

		// verify delete
		_, resp, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
		if resp != nil {
			resp.Body.Close()
		}
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
	}

	// certain characters are allowed in the description
	r = public.ServiceAccountRequest{
		Name:        "test",
		Description: "Created by the managed-services integration tests.,",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
	g.Expect(sa.ClientId).NotTo(gomega.BeEmpty())
	g.Expect(sa.ClientSecret).NotTo(gomega.BeEmpty())
	g.Expect(sa.Id).NotTo(gomega.BeEmpty())

	//verify delete
	_, resp, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "service-account-1",
		Description: "created by the managed service integration #$@#$#@$#@$$#",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))

	// verify malformed  id
	id := faker.ID
	_, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusBadRequest))
}

func TestServiceAccounts_SsoProvider_MAS_SSO(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(c *keycloak.KeycloakConfig) {
		c.SelectSSOProvider = keycloak.MAS_SSO
	})
	defer teardown()

	ctx := h.NewAuthenticatedContext(nil, nil)

	sp, resp, err := client.SecurityApi.GetSsoProviders(ctx)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(sp.Name).To(gomega.Equal("mas_sso"))
	g.Expect(sp.BaseUrl).To(gomega.Equal(test.TestServices.KeycloakConfig.SSOProviderRealm().BaseURL))
	g.Expect(sp.TokenUrl).To(gomega.Equal(test.TestServices.KeycloakConfig.SSOProviderRealm().TokenEndpointURI))
}

func TestServiceAccounts_SsoProvider_SSO(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(c *keycloak.KeycloakConfig) {
		c.SelectSSOProvider = keycloak.REDHAT_SSO
	})
	defer teardown()

	ctx := h.NewAuthenticatedContext(nil, nil)

	sp, resp, err := client.SecurityApi.GetSsoProviders(ctx)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(sp.Name).To(gomega.Equal("redhat_sso"))
	g.Expect(sp.BaseUrl).To(gomega.Equal(test.TestServices.KeycloakConfig.SSOProviderRealm().BaseURL))
	g.Expect(sp.TokenUrl).To(gomega.Equal(test.TestServices.KeycloakConfig.SSOProviderRealm().TokenEndpointURI))
}

// Todo Temporary commenting out the test
// func TestServiceAccount_CreationLimits(t *testing.T) {
// 	g := gomega.NewWithT(t)

// 	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
// 	defer ocmServer.Close()

// 	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
// 	// ocm
// 	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
// 	defer teardown()

// 	account := h.NewRandAccount()
// 	ctx := h.NewAuthenticatedContext(account, nil)

// 	r := public.ServiceAccountRequest{
// 		Name:        "test-account-acc-1",
// 		Description: "created by the managed service integration tests",
// 	}

// 	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
// 	g.Expect(err).ShouldNot(gomega.HaveOccurred())
// 	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
// 	g.Expect(sa.ClientId).NotTo(gomega.BeEmpty())
// 	g.Expect(sa.ClientSecret).NotTo(gomega.BeEmpty())
// 	g.Expect(sa.CreatedBy).Should(gomega.Equal(account.Username()))
// 	g.Expect(sa.Id).NotTo(gomega.BeEmpty())

// 	r = public.ServiceAccountRequest{
// 		Name:        "test-account-acc-2",
// 		Description: "created by the managed service integration tests",
// 	}
// 	sa2, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
// 	g.Expect(err).ShouldNot(gomega.HaveOccurred())
// 	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
// 	g.Expect(sa2.ClientId).NotTo(gomega.BeEmpty())
// 	g.Expect(sa2.ClientSecret).NotTo(gomega.BeEmpty())
// 	g.Expect(sa2.CreatedBy).Should(gomega.Equal(account.Username()))
// 	g.Expect(sa2.Id).NotTo(gomega.BeEmpty())

// 	// limit has reached for 2 service accounts
// 	r = public.ServiceAccountRequest{
// 		Name:        "test-account-acc-3",
// 		Description: "created by the managed service integration tests",
// 	}
// 	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
// 	g.Expect(err).Should(gomega.HaveOccurred())
// 	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusForbidden))

// 	//cleanup
// 	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
// 	g.Expect(err).ShouldNot(gomega.HaveOccurred())

// 	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa2.Id)
// 	g.Expect(err).ShouldNot(gomega.HaveOccurred())
// }
