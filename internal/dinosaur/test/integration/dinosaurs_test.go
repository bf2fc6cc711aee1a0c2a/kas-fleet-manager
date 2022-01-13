package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/quota_management"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/fleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm/clusterservicetest"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
)

const (
	mockDinosaurName      = "test-dinosaur1"
	dinosaurReadyTimeout  = time.Minute * 10
	dinosaurCheckInterval = time.Second * 1
	testMultiAZ           = true
	invalidDinosaurName   = "Test_Cluster9"
	longDinosaurName      = "thisisaninvaliddinosaurclusternamethatexceedsthenamesizelimit"
)

// TestDinosaurCreate_Success validates the happy path of the dinosaur post endpoint:
// - dinosaur successfully registered with database
// - dinosaur worker picks up on creation job
// - cluster is found for dinosaur
// - dinosaur is assigned cluster
func TestDinosaurCreate_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// POST responses per openapi spec: 201, 409, 500
	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	dinosaur, resp, err := common.WaitForDinosaurCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)

	// dinosaur successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/dinosaurs_mgmt/v1/dinosaurs/%s", dinosaur.Id)))
	Expect(dinosaur.InstanceType).To(Equal(types.STANDARD.String()))
	Expect(dinosaur.ReauthenticationEnabled).To(BeTrue())

	// wait until the dinosaur goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundDinosaur, err := common.WaitForDinosaurToReachStatus(ctx, test.TestServices.DBFactory, client, dinosaur.Id, constants2.DinosaurRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for dinosaur request to become ready: %v", err)
	// check the owner is set correctly
	Expect(foundDinosaur.Owner).To(Equal(account.Username()))
	Expect(foundDinosaur.BootstrapServerHost).To(Not(BeEmpty()))
	Expect(foundDinosaur.Version).To(Equal(fleetshardsync.GetDefaultReportedDinosaurVersion()))
	Expect(foundDinosaur.Owner).To(Equal(dinosaur.Owner))
	// checking dinosaur_request bootstrap server port number being present
	dinosaur, _, err = client.DefaultApi.GetDinosaurById(ctx, foundDinosaur.Id)
	Expect(err).NotTo(HaveOccurred(), "Error getting created dinosaur_request:  %v", err)
	Expect(strings.HasSuffix(dinosaur.BootstrapServerHost, ":443")).To(Equal(true))
	Expect(dinosaur.Version).To(Equal(fleetshardsync.GetDefaultReportedDinosaurVersion()))

	db := test.TestServices.DBFactory.New()
	var dinosaurRequest dbapi.DinosaurRequest
	if err := db.Unscoped().Where("id = ?", dinosaur.Id).First(&dinosaurRequest).Error; err != nil {
		t.Error("failed to find dinosaur request")
	}

	Expect(dinosaurRequest.SsoClientID).To(BeEmpty())
	Expect(dinosaurRequest.SsoClientSecret).To(BeEmpty())
	Expect(dinosaurRequest.QuotaType).To(Equal(DinosaurConfig(h).Quota.Type))
	Expect(dinosaurRequest.PlacementId).To(Not(BeEmpty()))
	Expect(dinosaurRequest.Owner).To(Not(BeEmpty()))
	Expect(dinosaurRequest.Namespace).To(Equal(fmt.Sprintf("dinosaur-%s", strings.ToLower(dinosaurRequest.ID))))
	// this is set by the mockFleetshardSync
	Expect(dinosaurRequest.DesiredStrimziVersion).To(Equal("strimzi-cluster-operator.v0.23.0-0"))

	common.CheckMetricExposed(h, t, metrics.DinosaurCreateRequestDuration)
	common.CheckMetricExposed(h, t, metrics.ClusterStatusCapacityUsed)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationCreate.String()))

	// delete test dinosaur to free up resources on an OSD cluster
	deleteTestDinosaur(t, h, ctx, client, foundDinosaur.Id)
}

func TestDinosaur_Update(t *testing.T) {
	owner := "test-user"
	anotherOwner := "test_dinosaur_service"
	emptyOwner := ""
	sampleDinosaurID := api.NewID()
	reauthenticationEnabled := true
	reauthenticationEnabled2 := false
	emptyOwnerDinosaurUpdateReq := public.DinosaurUpdateRequest{Owner: &emptyOwner, ReauthenticationEnabled: &reauthenticationEnabled}
	sameOwnerDinosaurUpdateReq := public.DinosaurUpdateRequest{Owner: &owner}
	newOwnerDinosaurUpdateReq := public.DinosaurUpdateRequest{Owner: &anotherOwner}

	emptyDinosaurUpdate := public.DinosaurUpdateRequest{}

	reauthenticationUpdateToFalse := public.DinosaurUpdateRequest{ReauthenticationEnabled: &reauthenticationEnabled2}
	reauthenticationUpdateToTrue := public.DinosaurUpdateRequest{ReauthenticationEnabled: &reauthenticationEnabled}

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

	orgId := "13640203"
	ownerAccout := h.NewAccount(owner, owner, "some-email@dinosaur.com", orgId)
	nonOwnerAccount := h.NewAccount("other-user", "some-other-user", "some-other@dinosaur.com", orgId)

	ctx := h.NewAuthenticatedContext(ownerAccout, nil)
	nonOwnerCtx := h.NewAuthenticatedContext(nonOwnerAccount, nil)

	adminAccount := h.NewRandAccount()
	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	type args struct {
		ctx                   context.Context
		dinosaurID            string
		dinosaurUpdateRequest public.DinosaurUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result public.DinosaurRequest, resp *http.Response, err error)
	}{
		{
			name: "update with an empty body should not fail",
			args: args{
				ctx:                   ctx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: emptyDinosaurUpdate,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			},
		},
		{
			name: "should fail if owner is empty",
			args: args{
				ctx:                   ctx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: emptyOwnerDinosaurUpdateReq,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail if trying to update owner as not an owner/ org_admin",
			args: args{
				ctx:                   nonOwnerCtx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: newOwnerDinosaurUpdateReq,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail if trying to update with an empty body as a non owner or org admin",
			args: args{
				ctx:                   nonOwnerCtx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: emptyDinosaurUpdate,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when trying to update non-existent dinosaur",
			args: args{
				ctx:                   ctx,
				dinosaurID:            "non-existent-id",
				dinosaurUpdateRequest: sameOwnerDinosaurUpdateReq,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should succeed changing reauthentication enabled value to true",
			args: args{
				ctx:                   ctx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: reauthenticationUpdateToTrue,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.ReauthenticationEnabled).To(BeTrue())
			},
		},
		{
			name: "should succeed changing reauthentication enabled value to false",
			args: args{
				ctx:                   ctx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: reauthenticationUpdateToFalse,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.ReauthenticationEnabled).To(BeFalse())
			},
		},
		{
			name: "should succeed when passing a valid new username as an owner",
			args: args{
				ctx:                   adminCtx,
				dinosaurID:            sampleDinosaurID,
				dinosaurUpdateRequest: newOwnerDinosaurUpdateReq,
			},
			verifyResponse: func(result public.DinosaurRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Owner).To(Equal(*newOwnerDinosaurUpdateReq.Owner))
			},
		},
	}

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

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a dinosaur that will be updated
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID,
		},
		MultiAZ:                 false,
		Owner:                   owner,
		Region:                  "test",
		CloudProvider:           "test",
		Name:                    "test-dinosaur",
		OrganisationId:          orgId,
		Status:                  constants.DinosaurRequestStatusReady.String(),
		ClusterID:               cluster.ClusterID,
		ActualDinosaurVersion:   "2.6.0",
		DesiredDinosaurVersion:  "2.6.0",
		ActualStrimziVersion:    "v.23.0",
		DesiredStrimziVersion:   "v0.23.0",
		InstanceType:            types.STANDARD.String(),
		ReauthenticationEnabled: false,
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			result, resp, err := client.DefaultApi.UpdateDinosaurById(tt.args.ctx, tt.args.dinosaurID, tt.args.dinosaurUpdateRequest)
			tt.verifyResponse(result, resp, err)
		})
	}
}

func TestDinosaurCreate_TooManyDinosaurs(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurCloudProvider := "dummy"
	// this value is taken from config/quota-management-list-configuration.yaml
	orgId := "13640203"

	// create dummy dinosaurs
	db := test.TestServices.DBFactory.New()
	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-2",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := client.DefaultApi.CreateDinosaur(ctx, true, k)

	Expect(err).To(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

// TestDinosaurPost_Validations tests the API validations performed by the dinosaur creation endpoint
//
// these could also be unit tests
func TestDinosaurPost_Validations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name     string
		body     public.DinosaurRequestPayload
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: public.DinosaurRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        "us-east-3",
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: public.DinosaurRequestPayload{
				MultiAz:       mocks.MockCluster.MultiAZ(),
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when MultiAZ false provided",
			body: public.DinosaurRequestPayload{
				MultiAz:       false,
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: public.DinosaurRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is not valid",
			body: public.DinosaurRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          invalidDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is too long",
			body: public.DinosaurRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          longDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			_, resp, _ := client.DefaultApi.CreateDinosaur(ctx, true, tt.body)
			Expect(resp.StatusCode).To(Equal(tt.wantCode))
		})
	}
}

// TestDinosaurPost_NameUniquenessValidations tests dinosaur cluster name uniqueness verification during its creation by the API
func TestDinosaurPost_NameUniquenessValidations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 3)})
	})
	defer tearDown()

	// create two random accounts in same organisation
	account1 := h.NewRandAccount()
	ctx1 := h.NewAuthenticatedContext(account1, nil)

	account2 := h.NewRandAccount()
	ctx2 := h.NewAuthenticatedContext(account2, nil)

	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"

	// create a third account in a different organisation
	accountFromAnotherOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx3 := h.NewAuthenticatedContext(accountFromAnotherOrg, nil)

	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	// create the first dinosaur
	_, resp1, _ := client.DefaultApi.CreateDinosaur(ctx1, true, k)

	// attempt to create the second dinosaur with same name
	_, resp2, _ := client.DefaultApi.CreateDinosaur(ctx2, true, k)

	// create another dinosaur with same name for different org
	_, resp3, _ := client.DefaultApi.CreateDinosaur(ctx3, true, k)

	// verify that the first and third requests were accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp3.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request resulted into an error accepted
	Expect(resp2.StatusCode).To(Equal(http.StatusConflict))

}

// TestDinosaurDenyList_UnauthorizedValidation tests the deny list API access validations is performed when enabled
func TestDinosaurDenyList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// this value is taken from config/deny-list-configuration.yaml
	username := "denied-test-user1@example.com"
	account := h.NewAccount(username, username, username, "13640203")
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name      string
		operation func() *http.Response
	}{
		{
			name: "HTTP 403 when listing dinosaurs",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new dinosaur request",
			operation: func() *http.Response {
				body := public.DinosaurRequestPayload{
					CloudProvider: mocks.MockCluster.CloudProvider().ID(),
					MultiAz:       mocks.MockCluster.MultiAZ(),
					Region:        "us-east-3",
					Name:          mockDinosaurName,
				}

				_, resp, _ := client.DefaultApi.CreateDinosaur(ctx, true, body)
				return resp
			},
		},
		{
			name: "HTTP 403 when deleting new dinosaur request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.DeleteDinosaurById(ctx, "dinosaur-id", true)
				return resp
			},
		},
		{
			name: "HTTP 403 when getting a new dinosaur request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetDinosaurById(ctx, "dinosaur-id")
				return resp
			},
		},
	}

	// verify that the user does not have access to the service
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			resp := tt.operation()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
		})
	}
}

// TestDinosaurDenyList_RemovingDinosaurForDeniedOwners tests that all dinosaurs of denied owners are removed
func TestDinosaurDenyList_RemovingDinosaurForDeniedOwners(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, err := common.GetRunningOsdClusterID(h, t)
	if err != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", err)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// these values are taken from config/deny-list-configuration.yaml
	username1 := "denied-test-user1@example.com"
	username2 := "denied-test-user2@example.com"

	// this value is taken from config/quota-management-list-configuration.yaml
	orgId := "13640203"

	// create dummy dinosaurs and assign it to user, at the end we'll verify that the dinosaur has been deleted
	db := test.TestServices.DBFactory.New()
	dinosaurRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	dinosaurCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          username1,
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-2",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			ClusterID:      clusterID,
			Name:           "dummy-dinosaur-3",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusPreparing.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			MultiAZ:             false,
			Owner:               username2,
			Region:              dinosaurRegion,
			CloudProvider:       dinosaurCloudProvider,
			ClusterID:           clusterID,
			BootstrapServerHost: "dummy-bootstrap-server-host",
			Name:                "dummy-dinosaur-to-deprovision",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			OrganisationId:      orgId,
			Status:              constants2.DinosaurRequestStatusProvisioning.String(),
			InstanceType:        types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "some-other-user",
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "this-dinosaur-will-remain",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// Verify all Dinosaurs that needs to be deleted are removed
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurDeletionErr := common.WaitForNumberOfDinosaurToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	Expect(dinosaurDeletionErr).NotTo(HaveOccurred(), "Error waiting for first dinosaur deletion: %v", dinosaurDeletionErr)
}

// TestDinosaurQuotaManagementList_MaxAllowedInstances tests the quota management list max allowed instances creation validations is performed when enabled
// The max allowed instances limit is set to "1" set for one organisation is not depassed.
// At the same time, users of a different organisation should be able to create instances.
func TestDinosaurQuotaManagementList_MaxAllowedInstances(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1000)})
		acl.EnableInstanceLimitControl = true
	})
	defer teardown()

	// this value is taken from config/quota-management-list-configuration.yaml
	orgIdWithLimitOfOne := "12147054"
	internalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx := h.NewAuthenticatedContext(internalUserAccount, nil)

	dinosaur1 := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	dinosaur2 := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-dinosaur-2",
		MultiAz:       testMultiAZ,
	}

	dinosaur3 := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-dinosaur-3",
		MultiAz:       testMultiAZ,
	}

	// create the first dinosaur
	resp1Body, resp1, _ := client.DefaultApi.CreateDinosaur(internalUserCtx, true, dinosaur1)

	// create the second dinosaur
	_, resp2, _ := client.DefaultApi.CreateDinosaur(internalUserCtx, true, dinosaur2)

	// verify that the request errored with 403 forbidden for the account in same organisation
	Expect(resp2.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp2.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that the first request was accepted as standard type
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp1Body.InstanceType).To(Equal("standard"))

	// verify that user of the same org cannot create a new dinosaur since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx = h.NewAuthenticatedContext(accountInSameOrg, nil)

	_, sameOrgUserResp, _ := client.DefaultApi.CreateDinosaur(internalUserCtx, true, dinosaur2)
	Expect(sameOrgUserResp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(sameOrgUserResp.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of a different organisation can still create dinosaur instances
	accountInDifferentOrg := h.NewRandAccount()
	internalUserCtx = h.NewAuthenticatedContext(accountInDifferentOrg, nil)

	// attempt to create dinosaur for this user account
	_, resp4, _ := client.DefaultApi.CreateDinosaur(internalUserCtx, true, dinosaur1)

	// verify that the request was accepted
	Expect(resp4.StatusCode).To(Equal(http.StatusAccepted))

	// verify that an external user not listed in the quota list can create the default maximum allowed dinosaur instances
	externalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserCtx := h.NewAuthenticatedContext(externalUserAccount, nil)

	resp5Body, resp5, _ := client.DefaultApi.CreateDinosaur(externalUserCtx, true, dinosaur1)
	_, resp6, _ := client.DefaultApi.CreateDinosaur(externalUserCtx, true, dinosaur2)

	// verify that the first request was accepted for an eval type
	Expect(resp5.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp5Body.InstanceType).To(Equal("eval"))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp6.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp6.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that another external user in the same org can also create the default maximum allowed dinosaur instances
	externalUserSameOrgAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserSameOrgCtx := h.NewAuthenticatedContext(externalUserSameOrgAccount, nil)

	_, resp7, _ := client.DefaultApi.CreateDinosaur(externalUserSameOrgCtx, true, dinosaur3)
	_, resp8, _ := client.DefaultApi.CreateDinosaur(externalUserSameOrgCtx, true, dinosaur2)

	// verify that the first request was accepted
	Expect(resp7.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp8.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp8.Header.Get("Content-Type")).To(Equal("application/json"))
}

// TestDinosaurGet tests getting dinosaurs via the API endpoint
func TestDinosaurGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	reauthenticationEnabled := false
	k := public.DinosaurRequestPayload{
		Region:                  mocks.MockCluster.Region().ID(),
		CloudProvider:           mocks.MockCluster.CloudProvider().ID(),
		Name:                    mockDinosaurName,
		MultiAz:                 testMultiAZ,
		ReauthenticationEnabled: &reauthenticationEnabled,
	}

	seedDinosaur, _, err := client.DefaultApi.CreateDinosaur(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded dinosaur request: %s", err.Error())
	}

	// 200 OK
	dinosaur, resp, err := client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get dinosaur request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/dinosaurs_mgmt/v1/dinosaurs/%s", dinosaur.Id)))
	Expect(dinosaur.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(dinosaur.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(dinosaur.Name).To(Equal(mockDinosaurName))
	Expect(dinosaur.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))
	Expect(dinosaur.ReauthenticationEnabled).To(BeFalse())
	// When dinosaur is in 'Accepted' state it means that it still has not been
	// allocated to a cluster, which means that fleetshard-sync has not reported
	// yet any status, so the version attribute (actual version) at this point
	// should still be empty
	Expect(dinosaur.Version).To(Equal(""))

	// 404 Not Found
	dinosaur, resp, _ = client.DefaultApi.GetDinosaurById(ctx, fmt.Sprintf("not-%s", seedDinosaur.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// A different account than the one used to create the Dinosaur instance, within
	// the same organization than the used account used to create the Dinosaur instance,
	// should be able to read the Dinosaur cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	dinosaur, _, _ = client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(dinosaur.Id).NotTo(BeEmpty())

	// An account in a different organization than the one used to create the
	// Dinosaur instance should get a 404 Not Found.
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"
	account = h.NewAccountWithNameAndOrg(faker.Name(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// An allowed serviceaccount in config/quota-management-list-configuration.yaml
	// without an organization ID should get a 401 Unauthorized
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetDinosaurById(ctx, seedDinosaur.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestDinosaur_Delete(t *testing.T) {
	owner := "test-user"

	orgId := "13640203"

	sampleDinosaurID := api.NewID()

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

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgId)

	userCtx := h.NewAuthenticatedContext(userAccount, nil)

	type args struct {
		ctx        context.Context
		dinosaurID string
		async      bool
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting dinosaur without async set to true",
			args: args{
				ctx:        userCtx,
				dinosaurID: sampleDinosaurID,
				async:      false,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when deleting dinosaur with empty id",
			args: args{
				ctx:        userCtx,
				dinosaurID: "",
				async:      true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should succeed when deleting dinosaur with valid id and context",
			args: args{
				ctx:        userCtx,
				dinosaurID: sampleDinosaurID,
				async:      true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

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

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a dinosaur that will be updated
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID,
		},
		MultiAZ:                true,
		Owner:                  owner,
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-dinosaur",
		OrganisationId:         orgId,
		Status:                 constants.DinosaurRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualDinosaurVersion:  "2.6.0",
		DesiredDinosaurVersion: "2.6.0",
		ActualStrimziVersion:   "v.23.0",
		DesiredStrimziVersion:  "v0.23.0",
		InstanceType:           types.EVAL.String(),
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteDinosaurById(tt.args.ctx, tt.args.dinosaurID, tt.args.async)
			tt.verifyResponse(resp, err)
		})
	}
}

// TestDinosaurDelete_DeleteDuringCreation - test deleting dinosaur instance during creation
func TestDinosaurDelete_DeleteDuringCreation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(ocmConfig *ocm.OCMConfig, c *config.DataplaneClusterConfig) {
		if ocmConfig.MockMode == ocm.MockModeEmulateServer {
			// increase repeat interval to allow time to delete the dinosaur instance before moving onto the next state
			// no need to reset this on teardown as it is always set at the start of each test within the registerIntegrationWithHooks setup for emulated servers.
			workers.RepeatInterval = 10 * time.Second
		}
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 10)})
	})
	defer teardown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	// custom update dinosaur status implementation to ensure that dinosaurs created within this test never gets updated to a 'ready' state.
	mockFleetshardSyncBuilder.SetUpdateDinosaurStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error {
		dataplaneCluster, findDataplaneClusterErr := test.TestServices.ClusterService.FindCluster(services.FindClusterCriteria{
			Status: api.ClusterReady,
		})
		if findDataplaneClusterErr != nil {
			return fmt.Errorf("Unable to retrieve a ready data plane cluster: %v", findDataplaneClusterErr)
		}

		// only update dinosaur status if a data plane cluster is available
		if dataplaneCluster != nil {
			ctx := fleetshardsync.NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)
			dinosaurList, _, err := privateClient.AgentClustersApi.GetDinosaurs(ctx, dataplaneCluster.ClusterID)
			if err != nil {
				return err
			}
			dinosaurStatusList := make(map[string]private.DataPlaneDinosaurStatus)
			for _, dinosaur := range dinosaurList.Items {
				id := dinosaur.Metadata.Annotations.MasId
				if dinosaur.Spec.Deleted {
					dinosaurStatusList[id] = fleetshardsync.GetDeletedDinosaurStatusResponse()
				}
			}

			if _, err = privateClient.AgentClustersApi.UpdateDinosaurClusterStatus(ctx, dataplaneCluster.ClusterID, dinosaurStatusList); err != nil {
				return err
			}
		}
		return nil
	})
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.DinosaurRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	// Test deletion of a dinosaur in an 'accepted' state
	dinosaur, resp, err := common.WaitForDinosaurCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted dinosaur:  %v", err)

	_, _, err = client.DefaultApi.DeleteDinosaurById(ctx, dinosaur.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete dinosaur request: %v", err)

	_ = common.WaitForDinosaurToBeDeleted(ctx, test.TestServices.DBFactory, client, dinosaur.Id)
	dinosaurList, _, err := client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list dinosaur request: %v", err)
	Expect(dinosaurList.Total).Should(BeZero(), " Dinosaur list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDelete.String()))

	// Test deletion of a dinosaur in a 'preparing' state
	dinosaur, resp, err = common.WaitForDinosaurCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted dinosaur:  %v", err)

	dinosaur, err = common.WaitForDinosaurToReachStatus(ctx, test.TestServices.DBFactory, client, dinosaur.Id, constants2.DinosaurRequestStatusPreparing)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for dinosaur request to be preparing: %v", err)

	_, _, err = client.DefaultApi.DeleteDinosaurById(ctx, dinosaur.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete dinosaur request: %v", err)

	_ = common.WaitForDinosaurToBeDeleted(ctx, test.TestServices.DBFactory, client, dinosaur.Id)
	dinosaurList, _, err = client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list dinosaur request: %v", err)
	Expect(dinosaurList.Total).Should(BeZero(), " Dinosaur list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDelete.String()))

	// Test deletion of a dinosaur in a 'provisioning' state
	dinosaur, resp, err = common.WaitForDinosaurCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted dinosaur:  %v", err)

	dinosaur, err = common.WaitForDinosaurToReachStatus(ctx, test.TestServices.DBFactory, client, dinosaur.Id, constants2.DinosaurRequestStatusProvisioning)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for dinosaur request to be provisioning: %v", err)

	_, _, err = client.DefaultApi.DeleteDinosaurById(ctx, dinosaur.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete dinosaur request: %v", err)

	_ = common.WaitForDinosaurToBeDeleted(ctx, test.TestServices.DBFactory, client, dinosaur.Id)
	dinosaurList, _, err = client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list dinosaur request: %v", err)
	Expect(dinosaurList.Total).Should(BeZero(), " Dinosaur list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount, constants2.DinosaurOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.FleetManager, metrics.DinosaurOperationsTotalCount, constants2.DinosaurOperationDelete.String()))
}

// TestDinosaurDelete - tests fail dinosaur delete
func TestDinosaurDelete_Fail(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	dinosaur := public.DinosaurRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
		Id:            "invalid-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9",
	}

	_, _, err := client.DefaultApi.DeleteDinosaurById(ctx, dinosaur.Id, true)
	Expect(err).To(HaveOccurred())
	// The id is invalid, so the metric is not expected to exist
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.FleetManager, metrics.DinosaurOperationsSuccessCount), false)
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.FleetManager, metrics.DinosaurOperationsTotalCount), false)
}

func TestDinosaur_DeleteAdminNonOwner(t *testing.T) {
	owner := "test-user"

	sampleDinosaurID := api.NewID()

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

	otherUserAccount := h.NewAccountWithNameAndOrg("non-owner", "13640203")

	adminAccount := h.NewAccountWithNameAndOrg("admin", "13640203")

	otherUserCtx := h.NewAuthenticatedContext(otherUserAccount, nil)

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	type args struct {
		ctx        context.Context
		dinosaurID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting dinosaur by other user within the same org (not an owner)",
			args: args{
				ctx:        otherUserCtx,
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should not fail when deleting dinosaur by other user within the same org (not an owner but org admin)",
			args: args{
				ctx:        adminCtx,
				dinosaurID: sampleDinosaurID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

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
		Owner:                  owner,
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
		InstanceType:           types.STANDARD.String(),
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteDinosaurById(tt.args.ctx, tt.args.dinosaurID, true)
			tt.verifyResponse(resp, err)
		})
	}
}

// TestDinosaurList_Success tests getting dinosaur requests list
func TestDinosaurList_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.GetDinosaurs(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(initList.Items).To(BeEmpty(), "Expected empty dinosaur requests list")
	Expect(initList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(initList.Total).To(Equal(int32(0)), "Expected Total == 0")

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	reauthenticationEnabled := true
	k := public.DinosaurRequestPayload{
		Region:                  mocks.MockCluster.Region().ID(),
		CloudProvider:           mocks.MockCluster.CloudProvider().ID(),
		Name:                    mockDinosaurName,
		MultiAz:                 testMultiAZ,
		ReauthenticationEnabled: &reauthenticationEnabled,
	}

	// POST dinosaur request to populate the list
	seedDinosaur, _, err := client.DefaultApi.CreateDinosaur(initCtx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded DinosaurRequest: %s", err.Error())
	}

	// get populated list of dinosaur requests
	afterPostList, _, err := client.DefaultApi.GetDinosaurs(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected dinosaur requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get dinosaur request item from the list
	listItem := afterPostList.Items[0]

	// check whether the seedDinosaur properties are the same as those from the dinosaur request list item
	Expect(seedDinosaur.Id).To(Equal(listItem.Id))
	Expect(seedDinosaur.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindDinosaur))
	Expect(seedDinosaur.Href).To(Equal(listItem.Href))
	Expect(seedDinosaur.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(clusterservicetest.MockClusterRegion))
	Expect(seedDinosaur.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest.MockClusterCloudProvider))
	Expect(seedDinosaur.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockDinosaurName))
	Expect(listItem.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))
	Expect(listItem.ReauthenticationEnabled).To(BeTrue())

	// new account setup to prove that users can list dinosaurs instances created by a member of their org
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// get populated list of dinosaur requests

	afterPostList, _, err = client.DefaultApi.GetDinosaurs(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected dinosaur requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get dinosaur request item from the list
	listItem = afterPostList.Items[0]

	// check whether the seedDinosaur properties are the same as those from the dinosaur request list item
	Expect(seedDinosaur.Id).To(Equal(listItem.Id))
	Expect(seedDinosaur.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindDinosaur))
	Expect(seedDinosaur.Href).To(Equal(listItem.Href))
	Expect(seedDinosaur.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(clusterservicetest.MockClusterRegion))
	Expect(seedDinosaur.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest.MockClusterCloudProvider))
	Expect(seedDinosaur.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockDinosaurName))
	Expect(listItem.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) dinosaur instances
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)

	// expecting empty list for user that hasn't created any dinosaurs yet
	newUserList, _, err := client.DefaultApi.GetDinosaurs(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(newUserList.Items)).To(Equal(0), "Expected dinosaur requests list length to be 0")
	Expect(newUserList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(newUserList.Total).To(Equal(int32(0)), "Expected Total == 0")
}

// TestDinosaurList_InvalidToken - tests listing dinosaurs with invalid token
func TestDinosaurList_UnauthUser(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// create empty context
	ctx := context.Background()

	dinosaurRequests, resp, err := client.DefaultApi.GetDinosaurs(ctx, nil)
	Expect(err).To(HaveOccurred()) // expecting an error here due unauthenticated user
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(dinosaurRequests.Items).To(BeNil())
	Expect(dinosaurRequests.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(dinosaurRequests.Total).To(Equal(int32(0)), "Expected Total == 0")
}

func deleteTestDinosaur(t *testing.T, h *coreTest.Helper, ctx context.Context, client *public.APIClient, dinosaurID string) {
	_, _, err := client.DefaultApi.DeleteDinosaurById(ctx, dinosaurID, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete dinosaur request: %v", err)

	// wait for dinosaur to be deleted
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(dinosaurCheckInterval, dinosaurReadyTimeout).
		RetryLogMessagef("Waiting for dinosaur '%s' to be deleted", dinosaurID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			db := test.TestServices.DBFactory.New()
			var dinosaurRequest dbapi.DinosaurRequest
			if err := db.Unscoped().Where("id = ?", dinosaurID).First(&dinosaurRequest).Error; err != nil {
				return false, err
			}
			return dinosaurRequest.DeletedAt.Valid, nil
		}).Build().Poll()
	Expect(err).NotTo(HaveOccurred(), "Failed to delete dinosaur request: %v", err)
}

func TestDinosaurList_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      "invalidiss",
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestDinosaurList_CorrectTokenIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test.TestServices.ServerConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetDinosaurs(ctx, &public.GetDinosaursOpts{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

// TestDinosaur_RemovingExpiredDinosaurs_NoStandardInstances tests that all eval dinosaurs are removed after their allocated life span has expired.
// No standard instances are present
func TestDinosaur_RemovingExpiredDinosaurs_NoStandardInstances(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelper(t, ocmServer)
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account with values from config/quota-management-list-configuration.yaml
	testuser1 := "testuser1@example.com"
	orgId := "13640203"

	// create dummy dinosaurs and assign it to user, at the end we'll verify that the dinosaur has been deleted
	db := test.TestServices.DBFactory.New()
	dinosaurRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	dinosaurCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned

	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-to-remain",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-49 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              dinosaurRegion,
			CloudProvider:       dinosaurCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-dinosaur-2",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.DinosaurRequestStatusProvisioning.String(),
			InstanceType:        types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-48 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              dinosaurRegion,
			CloudProvider:       dinosaurCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-dinosaur-3",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.DinosaurRequestStatusReady.String(),
			InstanceType:        types.EVAL.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any dinosaurs whose life has expired has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurDeletionErr := common.WaitForNumberOfDinosaurToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	Expect(dinosaurDeletionErr).NotTo(HaveOccurred(), "Error waiting for dinosaur deletion: %v", dinosaurDeletionErr)
}

// TestDinosaur_RemovingExpiredDinosaurs_EmptyList tests that all eval dinosaurs are removed after their allocated life span has expired
// while standard instances are left behind
func TestDinosaur_RemovingExpiredDinosaurs_WithStandardInstances(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelper(t, ocmServer)
	defer tearDown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account without values from config/allow-list-configuration.yaml
	testuser1 := "testuser@noallowlist.com"
	orgId := "13640203"

	// create dummy dinosaurs and assign it to user, at the end we'll verify that the dinosaur has been deleted
	db := test.TestServices.DBFactory.New()
	dinosaurRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	dinosaurCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned

	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         dinosaurRegion,
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-not-yet-expired",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-49 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              dinosaurRegion,
			CloudProvider:       dinosaurCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-dinosaur-to-remove",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.DinosaurRequestStatusReady.String(),
			InstanceType:        types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-48 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              dinosaurRegion,
			CloudProvider:       dinosaurCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-dinosaur-long-lived",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.DinosaurRequestStatusReady.String(),
			InstanceType:        types.STANDARD.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any dinosaurs whose life has expired has been deleted.
	account := h.NewRandAccount()
	account.Organization().ID()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurDeletionErr := common.WaitForNumberOfDinosaurToBeGivenCount(ctx, test.TestServices.DBFactory, client, 2)
	Expect(dinosaurDeletionErr).NotTo(HaveOccurred(), "Error waiting for dinosaur deletion: %v", dinosaurDeletionErr)
}
