package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/dgrijalva/jwt-go"

	"github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

func TestServiceAccounts_Success(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	//verify list
	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	currTime := time.Now().Format(time.RFC3339)
	createdAt, _ := time.Parse(time.RFC3339, currTime)
	//verify create
	r := public.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Owner).Should(Equal(account.Username()))
	Expect(sa.Id).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	// verify get by id
	id := sa.Id
	sa, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.Owner).NotTo(BeEmpty())
	Expect(sa.Owner).Should(Equal(account.Username()))
	Expect(sa.Id).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	//verify reset
	oldSecret := sa.ClientSecret
	sa, _, err = client.SecurityApi.ResetServiceAccountCreds(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(Equal(oldSecret))
	Expect(sa.Owner).Should(Equal(account.Username()))
	Expect(sa.Owner).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())

	// verify deletion of non-existent service account throws http status code 404
	_, resp, _ = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	f := false
	accounts, _, _ := client.SecurityApi.GetServiceAccounts(ctx)
	for _, a := range accounts.Items {
		if a.Id == id {
			f = true
		}
	}
	Expect(f).To(BeFalse())
}

func TestServiceAccounts_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
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

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestServiceAccounts_CorrectOCMIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test.TestServices.OCMConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func TestServiceAccounts_InputValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	//length check
	r := public.ServiceAccountRequest{
		Name:        "length-more-than-50-is-not-allowed-managed-service-integration-test",
		Description: "created by the managed service integration",
	}
	_, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "<script>alert(\"TEST\");</script>",
		Description: "created by the managed service integration",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//description length can not be more than 255
	r = public.ServiceAccountRequest{
		Name:        "test-svc-1",
		Description: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//min length required for name
	r = public.ServiceAccountRequest{
		Name:        "",
		Description: "test",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//min length required is not required for desc
	r = public.ServiceAccountRequest{
		Name:        "test",
		Description: "",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())

	// certain characters are allowed in the description
	r = public.ServiceAccountRequest{
		Name:        "test",
		Description: "Created by the managed-services integration tests.,",
	}
	sa, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "service-account-1",
		Description: "created by the managed service integration #$@#$#@$#@$$#",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	// verify malformed  id
	id := faker.ID
	_, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	Expect(err).Should(HaveOccurred())
}

func TestServiceAccount_CreationLimits(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	r := public.ServiceAccountRequest{
		Name:        "test-account-acc-1",
		Description: "created by the managed service integration tests",
	}

	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Owner).Should(Equal(account.Username()))
	Expect(sa.Id).NotTo(BeEmpty())

	r = public.ServiceAccountRequest{
		Name:        "test-account-acc-2",
		Description: "created by the managed service integration tests",
	}
	sa2, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa2.ClientId).NotTo(BeEmpty())
	Expect(sa2.ClientSecret).NotTo(BeEmpty())
	Expect(sa2.Owner).Should(Equal(account.Username()))
	Expect(sa2.Id).NotTo(BeEmpty())

	// limit has reached for 2 service accounts
	r = public.ServiceAccountRequest{
		Name:        "test-account-acc-3",
		Description: "created by the managed service integration tests",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	//cleanup
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())

	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa2.Id)
	Expect(err).ShouldNot(HaveOccurred())
}
