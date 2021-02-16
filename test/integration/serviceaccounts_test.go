package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	. "github.com/onsi/gomega"
)

func TestServiceAccounts_Success(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	//verify list
	_, resp, err := client.DefaultApi.ListServiceAccounts(ctx)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	//verify create
	r := openapi.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.DefaultApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientID).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	// verify get by id
	id := sa.Id
	sa, resp, err = client.DefaultApi.GetServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientID).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	//verify reset
	oldSecret := sa.ClientSecret
	sa, _, err = client.DefaultApi.ResetServiceAccountCreds(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(Equal(oldSecret))

	//verify delete
	_, _, err = client.DefaultApi.DeleteServiceAccount(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())

	f := false
	accounts, _, _ := client.DefaultApi.ListServiceAccounts(ctx)
	for _, a := range accounts.Items {
		if a.Id == id {
			f = true
		}
	}
	Expect(f).To(BeFalse())
}
