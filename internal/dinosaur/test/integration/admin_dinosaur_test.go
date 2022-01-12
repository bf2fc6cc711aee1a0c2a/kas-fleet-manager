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
		ctx     func(h *coreTest.Helper) context.Context
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
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
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
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
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
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
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
			name: fmt.Sprintf("should fail when the role defined in the request is not %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminWriteRole})
				},
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
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
		ctx           func(h *coreTest.Helper) context.Context
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
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
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
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminWriteRole})
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
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.FleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
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
	sampleDinosaurID1 := api.NewID()
	sampleDinosaurID2 := api.NewID()
	sampleDinosaurID3 := api.NewID()
	sampleDinosaurID4 := api.NewID()
	fullyPopulatedDinosaurUpdateRequest := adminprivate.DinosaurUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		DinosaurVersion:    "2.8.2",
		DinosaurIbpVersion: "2.8.1",
	}
	emptyDinosaurUpdateRequest := adminprivate.DinosaurUpdateRequest{}
	ibpDowngrade := adminprivate.DinosaurUpdateRequest{
		DinosaurIbpVersion: "2.6.0",
	}
	ibpUpgrade := adminprivate.DinosaurUpdateRequest{
		DinosaurIbpVersion: "2.8.0",
	}
	ibpUpgradeHigherThanDinosaur := adminprivate.DinosaurUpdateRequest{
		DinosaurIbpVersion: "2.8.2",
	}
	dinosaurVersionMajorDowngrade := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "1.8.0",
	}
	dinosaurVersionMinorDowngrade := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "2.7.0",
	}
	dinosaurVersionPatchDowngrade := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "2.8.0",
	}
	dinosaurVersionPatchUpgradeInStatus := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "2.9.0",
	}
	dinosaurVersionPatchUpgradeNotInStatus := adminprivate.DinosaurUpdateRequest{
		DinosaurVersion: "2.10.0",
	}
	strimziUpgrade := adminprivate.DinosaurUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.25.0-0",
	}
	nonExistentStrimziUpgrade := adminprivate.DinosaurUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.29.0-0",
	}
	notReadyStrimziUpgrade := adminprivate.DinosaurUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.25.1-0",
	}
	allVersionsUpgrade := adminprivate.DinosaurUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.25.0-0",
		DinosaurIbpVersion: "2.8.1",
		DinosaurVersion:    "2.8.2",
	}
	versionsNotInStatusForValidStrimziVersion := adminprivate.DinosaurUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.25.0-0",
		DinosaurIbpVersion: "2.8.3",
		DinosaurVersion:    "2.8.3",
	}
	type args struct {
		ctx                func(h *coreTest.Helper) context.Context
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
				dinosaurID: sampleDinosaurID1,
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
				dinosaurID: sampleDinosaurID1,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should fail when the role defined in the request is %s", auth.FleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
				},
				dinosaurID: sampleDinosaurID1,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when dinosaurUpdateRequest is empty",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: emptyDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: ibpDowngrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version to version higher than dinosaur version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: ibpUpgradeHigherThanDinosaur,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID2,
				dinosaurUpdateRequest: ibpUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower minor dinosaur version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: dinosaurVersionMinorDowngrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower patch dinosaur version smaller than ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID3,
				dinosaurUpdateRequest: dinosaurVersionPatchDowngrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower major dinosaur version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID4,
				dinosaurUpdateRequest: dinosaurVersionMajorDowngrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading to higher minor dinosaur version when not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: dinosaurVersionPatchUpgradeNotInStatus,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading dinosaur version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID2,
				dinosaurUpdateRequest: dinosaurVersionPatchUpgradeInStatus,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID2,
				dinosaurUpdateRequest: strimziUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version not in the status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: nonExistentStrimziUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading simultaneously dinosaur, ibp and strimzi version when any of those not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: versionsNotInStatusForValidStrimziVersion,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version that is not ready",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID2,
				dinosaurUpdateRequest: notReadyStrimziUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when the requested dinosaur does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminReadRole})
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
							"roles": {auth.FleetManagerAdminReadRole},
						},
					}
					token := h.CreateJWTStringWithClaim(account, claims)
					ctx := context.WithValue(context.Background(), adminprivate.ContextAccessToken, token)
					return ctx
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.FleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminWriteRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.FleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: fullyPopulatedDinosaurUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
			},
		},
		{
			name: "should succeed when upgrading ibp version to lower than dinosaur version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: ibpUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
				Expect(result.DesiredDinosaurIbpVersion).To(Equal(ibpUpgrade.DinosaurIbpVersion))
			},
		},
		{
			name: "should succeed when downgrading to lower patch dinosaur version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: dinosaurVersionPatchDowngrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
				Expect(result.DesiredDinosaurVersion).To(Equal(dinosaurVersionPatchDowngrade.DinosaurVersion))
			},
		},
		{
			name: "should succeed when upgrading to higher minor dinosaur version when in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: dinosaurVersionPatchUpgradeInStatus,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
				Expect(result.DesiredDinosaurVersion).To(Equal(dinosaurVersionPatchUpgradeInStatus.DinosaurVersion))
			},
		},
		{
			name: "should succeed when upgrading simultaneously dinosaur, ibp and strimzi version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.FleetManagerAdminFullRole})
				},
				dinosaurID:            sampleDinosaurID1,
				dinosaurUpdateRequest: allVersionsUpgrade,
			},
			verifyResponse: func(result adminprivate.Dinosaur, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleDinosaurID1))
				Expect(result.DesiredDinosaurVersion).To(Equal(allVersionsUpgrade.DinosaurVersion))
				Expect(result.DesiredDinosaurIbpVersion).To(Equal(allVersionsUpgrade.DinosaurIbpVersion))
				Expect(result.DesiredStrimziVersion).To(Equal(allVersionsUpgrade.StrimziVersion))
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

	err2 := cluster.SetAvailableStrimziVersions(
		[]api.StrimziVersion{
			{
				Version: "strimzi-cluster-operator.v0.20.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "1.8.0"},
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "1.8.0"},
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.22.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.23.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.24.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
					{Version: "2.8.2"},
					{Version: "2.9.0"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
					{Version: "2.8.2"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.25.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
					{Version: "2.8.2"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.25.1-0",
				Ready:   false,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
					{Version: "2.8.2"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.6.0"},
					{Version: "2.7.0"},
					{Version: "2.8.0"},
					{Version: "2.8.1"},
				},
			},
			{
				Version: "strimzi-cluster-operator.v0.26.0-0",
				Ready:   true,
				DinosaurVersions: []api.DinosaurVersion{
					{Version: "2.8.2"},
				},
				DinosaurIBPVersions: []api.DinosaurIBPVersion{
					{Version: "2.8.2"},
				},
			},
		},
	)

	if err2 != nil {
		t.Error("failed to set available strimzi versions")
		return
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create dinosaurs that will be upgraded in the tests
	dinosaur1 := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID1,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-dinosaur1",
		OrganisationId:         "13640203",
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:     "2.8.1",
		DesiredDinosaurVersion:    "2.8.1",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualDinosaurIBPVersion:  "2.7.0",
		DesiredDinosaurIBPVersion: "2.7.0",
	}

	dinosaur2 := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID2,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-dinosaur2",
		OrganisationId:         "13640203",
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:     "2.8.0",
		DesiredDinosaurVersion:    "2.8.0",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualDinosaurIBPVersion:  "2.7.0",
		DesiredDinosaurIBPVersion: "2.7.0",
		DinosaurIBPUpgrading:      true,
		DinosaurUpgrading:         true,
		StrimziUpgrading:       true,
	}

	dinosaur3 := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID3,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-dinosaur3",
		OrganisationId:         "13640203",
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:     "2.8.2",
		DesiredDinosaurVersion:    "2.8.2",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualDinosaurIBPVersion:  "2.8.1",
		DesiredDinosaurIBPVersion: "2.8.1",
		DinosaurIBPUpgrading:      true,
	}

	dinosaur4 := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID4,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-kafk4",
		OrganisationId:         "13640203",
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:     "2.6.0",
		DesiredDinosaurVersion:    "2.6.0",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.20.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.20.0-0",
		ActualDinosaurIBPVersion:  "1.8.0",
		DesiredDinosaurIBPVersion: "1.8.0",
		DinosaurIBPUpgrading:      true,
		DinosaurUpgrading:         true,
		StrimziUpgrading:       true,
	}

	if err := db.Create(dinosaur1).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	if err := db.Create(dinosaur2).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	if err := db.Create(dinosaur3).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	if err := db.Create(dinosaur4).Error; err != nil {
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
