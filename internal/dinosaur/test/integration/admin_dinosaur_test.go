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
	"github.com/dgrijalva/jwt-go"
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
	desiredStrimziVersion := "test"
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
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
				Expect(result.Namespace).ToNot(BeEmpty())
			},
		},
		{
			name: "should fail when the requested dinosaur does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
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
							"roles": {auth.KasFleetManagerAdminReadRole},
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
		MultiAZ:               false,
		Owner:                 "test-user",
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-dinosaur",
		OrganisationId:        "13640203",
		DesiredStrimziVersion: desiredStrimziVersion,
		Status:                constants.DinosaurRequestStatusReady.String(),
		Namespace:             fmt.Sprintf("dinosaur-%s", sampleDinosaurID),
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

func TestAdminDinosaur_Delete(t *testing.T) {
	type args struct {
		ctx func(h *coreTest.Helper) context.Context
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should fail when the role defined in the request is not %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
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

	dinosaurId := api.NewID()
	db := test.TestServices.DBFactory.New()
	dinosaur := &dbapi.DinosaurRequest{
		MultiAZ:        false,
		Owner:          "test-user",
		Region:         "test",
		CloudProvider:  "test",
		Name:           "test-dinosaur",
		OrganisationId: "13640203",
		Status:         constants.DinosaurRequestStatusReady.String(),
	}
	dinosaur.ID = dinosaurId

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			_, resp, err := client.DefaultApi.DeleteDinosaurById(ctx, dinosaurId, true)
			tt.verifyResponse(resp, err)
		})
	}
}

func TestAdminDinosaur_List(t *testing.T) {
	type args struct {
		ctx              func(h *coreTest.Helper) context.Context
		dinosaurListSize int
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error, dinosaurList adminprivate.DinosaurList, expectedSize int)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				dinosaurListSize: 0,
			},
			verifyResponse: func(resp *http.Response, err error, dinosaurList adminprivate.DinosaurList, expectedSize int) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				Expect(len(dinosaurList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, dinosaurList adminprivate.DinosaurList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(dinosaurList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				dinosaurListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, dinosaurList adminprivate.DinosaurList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(dinosaurList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				dinosaurListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, dinosaurList adminprivate.DinosaurList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(dinosaurList.Items)).To(Equal(expectedSize))
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
	dinosaurs := []dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          "test-user1",
			Region:         "test",
			CloudProvider:  "test",
			Name:           "test-dinosaur1",
			OrganisationId: "13640203",
			Status:         constants.DinosaurRequestStatusReady.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "test-user2",
			Region:         "test",
			CloudProvider:  "test",
			Name:           "test-dinosaur2",
			OrganisationId: "12147054",
			Status:         constants.DinosaurRequestStatusReady.String(),
		},
	}

	for _, dinosaur := range dinosaurs {
		if err := db.Create(&dinosaur).Error; err != nil {
			t.Errorf("failed to create Dinosaur db record due to error: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			dinosaurList, resp, err := client.DefaultApi.GetDinosaurs(ctx, nil)
			tt.verifyResponse(resp, err, dinosaurList, tt.args.dinosaurListSize)
		})
	}
}

func TestAdminDinosaur_Update(t *testing.T) {
	sampleDinosaurID := api.NewID()
	fullyPopulatedDinosaurUpdateRequest := adminprivate.DinosaurUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		DinosaurVersion: "2.7.0",
	}
	emptyDinosaurUpdateRequest := adminprivate.DinosaurUpdateRequest{}
	updateRequestWithDinosaurVersion := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "2.7.0",
	}
	updateRequestWithStrimziVersion := adminprivate.DinosaurUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.23.0-0",
	}
	unsupportedStrimziVersionUpdateRequest := adminprivate.DinosaurUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.999.0-0",
	}
	type args struct {
		ctx                   func(h *coreTest.Helper) context.Context
		dinosaurID            string
		dinosaurUpdateRequest adminprivate.DinosaurUpdateRequest
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
			name: fmt.Sprintf("should fail when the role defined in the request is %s", auth.KasFleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
			},
		},
		{
			name: "should succeed when updating dinosaur version only",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: updateRequestWithDinosaurVersion,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
			},
		},
		{
			name: "should succeed when updating strimzi version only",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: updateRequestWithStrimziVersion,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID))
			},
		},
		{
			name: "should fail when setting unsupported strimzi version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: unsupportedStrimziVersionUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when dinosaurUpdateRequest is empty",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: emptyDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when the requested dinosaur does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				dinosaurID:            "nonexistentdinosaurID",
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
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
							"roles": {auth.KasFleetManagerAdminReadRole},
						},
					}
					token := h.CreateJWTStringWithClaim(account, claims)
					ctx := context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)
					return ctx
				},
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
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
	// create a dummy cluster and assign a dinosaur to it
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	err2 := cluster.SetAvailableStrimziVersions([]api.StrimziVersion{{
		Version: updateRequestWithStrimziVersion.StrimziVersion,
		Ready:   true,
	}, {
		Ready:   true,
		Version: fullyPopulatedDinosaurUpdateRequest.StrimziVersion,
	}})

	if err2 != nil {
		t.Error("failed to set available strimzi versions")
		return
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a dinosaur that will be updated
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-dinosaur",
		OrganisationId:         "13640203",
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:  "2.6.0",
		DesiredDinosaurVersion: "2.6.0",
		ActualStrimziVersion:   "v.23.0",
		DesiredStrimziVersion:  "v0.23.0",
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.UpdateDinosaurById(ctx, tt.args.dinosaurID, tt.args.dinosaurUpdateRequest)
			tt.verifyResponse(result, resp, err)
		})
	}
}
