package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
)

func NewAuthenticatedContextForAdminEndpoints(h *coreTest.Helper, realmRoles []string) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": keycloakConfig.OSDClusterIDPRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": realmRoles,
		},
		"preferred_username": "integration-test-user",
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)

	return ctx
}

func TestAdminDinosaur_Get(t *testing.T) {
	sampleDinosaurID := api.NewID()
	desiredDinosaurOperatorVersion := "test"
	type args struct {
		ctx        func(h *coreTest.Helper) context.Context
		dinosaurID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result adminprivate.Dinosaur, resp *http.Response, err error)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when the role defined in the request is not any of read, write or full",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{"notallowedrole"})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredDinosaurOperatorVersion).To(Equal(desiredDinosaurOperatorVersion))
				Expect(result.AccountNumber).ToNot(BeEmpty())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminWriteRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredDinosaurOperatorVersion).To(Equal(desiredDinosaurOperatorVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredDinosaurOperatorVersion).To(Equal(desiredDinosaurOperatorVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: "should fail when the requested dinosaur does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
				},
				dinosaurID: "unexistingdinosaurID",
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when the request does not contain a valid issuer",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					account := h.NewAllowedServiceAccount()
					claims := jwt.MapClaims{
						"iss": "invalidiss",
						"realm_access": map[string][]string{
							"roles": {auth.FleetManagerAdminReadRole},
						},
					}
					token := h.CreateJWTStringWithClaim(account, claims)
					ctx := context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)
					return ctx
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
	}

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewDinosaurHelper(t, ocmServer)
	defer tearDown()
	db := test.TestServices.DBFactory.New()
	dinosaur := &dbapi.DinosaurRequest{
		MultiAZ:                        false,
		Owner:                          "test-user",
		Region:                         "test",
		CloudProvider:                  "test",
		Name:                           "test-dinosaur",
		OrganisationId:                 "13640203",
		DesiredDinosaurOperatorVersion: desiredDinosaurOperatorVersion,
		Status:                         constants.DinosaurRequestStatusReady.String(),
		Namespace:                      fmt.Sprintf("dinosaur-%s", sampleDinosaurID),
	}
	dinosaur.ID = sampleDinosaurID

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.GetDinosaurById(ctx, tt.args.dinosaurID)
			tt.verifyResponse(result, resp, err)
		})
	}
}
