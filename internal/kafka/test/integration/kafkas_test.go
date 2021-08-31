package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm/clusterservicetest"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
)

const (
	mockKafkaName      = "test-kafka1"
	kafkaReadyTimeout  = time.Minute * 10
	kafkaCheckInterval = time.Second * 1
	testMultiAZ        = true
	invalidKafkaName   = "Test_Cluster9"
	longKafkaName      = "thisisaninvalidkafkaclusternamethatexceedsthenamesizelimit"
)

// TestKafkaCreate_Success validates the happy path of the kafka post endpoint:
// - kafka successfully registered with database
// - kafka worker picks up on creation job
// - cluster is found for kafka
// - kafka is assigned cluster
func TestKafkaCreate_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka, resp, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)

	// kafka successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.InstanceType).To(Equal(types.STANDARD.String()))

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundKafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
	Expect(foundKafka.Version).To(Equal(kasfleetshardsync.GetDefaultReportedKafkaVersion()))
	Expect(foundKafka.Owner).To(Equal(kafka.Owner))
	// checking kafka_request bootstrap server port number being present
	kafka, _, err = client.DefaultApi.GetKafkaById(ctx, foundKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error getting created kafka_request:  %v", err)
	Expect(strings.HasSuffix(kafka.BootstrapServerHost, ":443")).To(Equal(true))
	Expect(kafka.Version).To(Equal(kasfleetshardsync.GetDefaultReportedKafkaVersion()))

	db := test.TestServices.DBFactory.New()
	var kafkaRequest dbapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}

	Expect(kafkaRequest.SsoClientID).To(BeEmpty())
	Expect(kafkaRequest.SsoClientSecret).To(BeEmpty())
	Expect(kafkaRequest.QuotaType).To(Equal(KafkaConfig(h).Quota.Type))
	Expect(kafkaRequest.PlacementId).To(Not(BeEmpty()))
	Expect(kafkaRequest.Owner).To(Not(BeEmpty()))
	Expect(kafkaRequest.Namespace).To(Equal(fmt.Sprintf("kafka-%s", strings.ToLower(kafkaRequest.ID))))
	// this is set by the mockKasfFleetshardSync
	Expect(kafkaRequest.DesiredStrimziVersion).To(Equal("strimzi-cluster-operator.v0.23.0-0"))

	common.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationCreate.String()))

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}

func TestKafka_Update(t *testing.T) {
	owner := "test-user"

	sampleKafkaID := api.NewID()
	emptyOwnerKafkaUpdateReq := public.KafkaUpdateRequest{}
	sameOwnerKafkaUpdateReq := public.KafkaUpdateRequest{Owner: owner}
	newOwnerKafkaUpdateReq := public.KafkaUpdateRequest{Owner: "test_kafka_service"}

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	account := h.NewRandAccount()

	ctx := h.NewAuthenticatedContext(account, nil)

	otherUserCtx := h.NewAuthenticatedContext(account, nil)

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(account, claims)

	type args struct {
		ctx                context.Context
		kafkaID            string
		kafkaUpdateRequest public.KafkaUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result public.KafkaRequest, resp *http.Response, err error)
	}{
		{
			name: "should fail if owner is empty",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: emptyOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail if trying to update owner as not an owner/ org_admin",
			args: args{
				ctx:                otherUserCtx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: newOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should not fail if passed owner is the same",
			args: args{
				ctx:                ctx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: sameOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
			},
		},
		{
			name: "should fail when trying to update non-existent kafka",
			args: args{
				ctx:                ctx,
				kafkaID:            "non-existent-id",
				kafkaUpdateRequest: sameOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should succeed when passing a valid new username as an owner",
			args: args{
				ctx:                adminCtx,
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: newOwnerKafkaUpdateReq,
			},
			verifyResponse: func(result public.KafkaRequest, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Owner).To(Equal(newOwnerKafkaUpdateReq.Owner))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
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

	// create a kafka that will be updated
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID,
		},
		MultiAZ:               false,
		Owner:                 owner,
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        "13640203",
		Status:                constants.KafkaRequestStatusReady.String(),
		ClusterID:             cluster.ClusterID,
		ActualKafkaVersion:    "2.6.0",
		DesiredKafkaVersion:   "2.6.0",
		ActualStrimziVersion:  "v.23.0",
		DesiredStrimziVersion: "v0.23.0",
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			result, resp, err := client.DefaultApi.UpdateKafkaById(tt.args.ctx, tt.args.kafkaID, tt.args.kafkaUpdateRequest)
			tt.verifyResponse(result, resp, err)
		})
	}
}

func TestKafka_Update_RemoveFailedReason(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka, resp, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	kafka.FailedReason = "test fail reason"

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.FailedReason).To(Equal("test fail reason"))
	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundKafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)

	db := test.TestServices.DBFactory.New()
	var kafkaRequest dbapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}
	Expect(kafkaRequest.FailedReason).To(Equal(""))

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}

func TestKafkaCreate_OverrideDesiredStrimziVersion(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	desiredStrimziVersion := "strimzi-test-v0.0.1-1"

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(config *config.DataplaneClusterConfig) {
		config.StrimziOperatorVersion = desiredStrimziVersion
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka, _, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	// kafka successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundKafka, err := common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)

	db := test.TestServices.DBFactory.New()
	var kafkaRequest dbapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}
	Expect(kafkaRequest.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, ctx, client, foundKafka.Id)
}

func TestKafkaCreate_TooManyKafkas(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewKafkaHelperWithHooks(t, ocmServer, func(c *config.KafkaConfig) {
		c.KafkaCapacity.MaxCapacity = 2
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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

	kafkaRegion := "dummy"
	kafkaCloudProvider := "dummy"
	// this value is taken from config/quota-management-list-configuration.yaml
	orgId := "13640203"

	// create dummy kafkas
	db := test.TestServices.DBFactory.New()
	kafkas := []*dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-2",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := client.DefaultApi.CreateKafka(ctx, true, k)

	Expect(err).To(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusTooManyRequests))
}

// TestKafkaPost_Validations tests the API validations performed by the kafka creation endpoint
//
// these could also be unit tests
func TestKafkaPost_Validations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name     string
		body     public.KafkaRequestPayload
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        "us-east-3",
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: public.KafkaRequestPayload{
				MultiAz:       mocks.MockCluster.MultiAZ(),
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when MultiAZ false provided",
			body: public.KafkaRequestPayload{
				MultiAz:       false,
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is not valid",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          invalidKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is too long",
			body: public.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          longKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			_, resp, _ := client.DefaultApi.CreateKafka(ctx, true, tt.body)
			Expect(resp.StatusCode).To(Equal(tt.wantCode))
		})
	}
}

// TestKafkaPost_NameUniquenessValidations tests kafka cluster name uniqueness verification during its creation by the API
func TestKafkaPost_NameUniquenessValidations(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

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

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	// create the first kafka
	_, resp1, _ := client.DefaultApi.CreateKafka(ctx1, true, k)

	// attempt to create the second kafka with same name
	_, resp2, _ := client.DefaultApi.CreateKafka(ctx2, true, k)

	// create another kafka with same name for different org
	_, resp3, _ := client.DefaultApi.CreateKafka(ctx3, true, k)

	// verify that the first and third requests were accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp3.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request resulted into an error accepted
	Expect(resp2.StatusCode).To(Equal(http.StatusConflict))

}

// TestKafkaDenyList_UnauthorizedValidation tests the deny list API access validations is performed when enabled
func TestKafkaDenyList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
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
			name: "HTTP 403 when listing kafkas",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new kafka request",
			operation: func() *http.Response {
				body := public.KafkaRequestPayload{
					CloudProvider: mocks.MockCluster.CloudProvider().ID(),
					MultiAz:       mocks.MockCluster.MultiAZ(),
					Region:        "us-east-3",
					Name:          mockKafkaName,
				}

				_, resp, _ := client.DefaultApi.CreateKafka(ctx, true, body)
				return resp
			},
		},
		{
			name: "HTTP 403 when deleting new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.DeleteKafkaById(ctx, "kafka-id", true)
				return resp
			},
		},
		{
			name: "HTTP 403 when getting a new kafka request",
			operation: func() *http.Response {
				_, resp, _ := client.DefaultApi.GetKafkaById(ctx, "kafka-id")
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

// TestKafkaDenyList_RemovingKafkaForDeniedOwners tests that all kafkas of denied owners are removed
func TestKafkaDenyList_RemovingKafkaForDeniedOwners(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	kafkas := []*dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          username1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-2",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			ClusterID:      clusterID,
			Name:           "dummy-kafka-3",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusPreparing.String(),
		},
		{
			MultiAZ:             false,
			Owner:               username2,
			Region:              kafkaRegion,
			CloudProvider:       kafkaCloudProvider,
			ClusterID:           clusterID,
			BootstrapServerHost: "dummy-bootstrap-server-host",
			Name:                "dummy-kafka-to-deprovision",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			OrganisationId:      orgId,
			Status:              constants2.KafkaRequestStatusProvisioning.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "some-other-user",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "this-kafka-will-remain",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// Verify all Kafkas that needs to be deleted are removed
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for first kafka deletion: %v", kafkaDeletionErr)
}

// TestKafkaQuotaManagementList_MaxAllowedInstances tests the quota management list max allowed instances creation validations is performed when enabled
// The max allowed instances limit is set to "1" set for one organisation is not depassed.
// At the same time, users of a different organisation should be able to create instances.
func TestKafkaQuotaManagementList_MaxAllowedInstances(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig) {
		acl.EnableInstanceLimitControl = true
	})
	defer teardown()

	// this value is taken from config/quota-management-list-configuration.yaml
	orgIdWithLimitOfOne := "12147054"
	internalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx := h.NewAuthenticatedContext(internalUserAccount, nil)

	kafka1 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka2 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-2",
		MultiAz:       testMultiAZ,
	}

	kafka3 := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-3",
		MultiAz:       testMultiAZ,
	}

	// create the first kafka
	resp1Body, resp1, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)

	// create the second kafka
	_, resp2, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)

	// verify that the request errored with 403 forbidden for the account in same organisation
	Expect(resp2.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp2.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that the first request was accepted as standard type
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp1Body.InstanceType).To(Equal("standard"))

	// verify that user of the same org cannot create a new kafka since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx = h.NewAuthenticatedContext(accountInSameOrg, nil)

	_, sameOrgUserResp, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)
	Expect(sameOrgUserResp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(sameOrgUserResp.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of a different organisation can still create kafka instances
	accountInDifferentOrg := h.NewRandAccount()
	internalUserCtx = h.NewAuthenticatedContext(accountInDifferentOrg, nil)

	// attempt to create kafka for this user account
	_, resp4, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)

	// verify that the request was accepted
	Expect(resp4.StatusCode).To(Equal(http.StatusAccepted))

	// verify that an external user not listed in the quota list can create the default maximum allowed kafka instances
	externalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserCtx := h.NewAuthenticatedContext(externalUserAccount, nil)

	resp5Body, resp5, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka1)
	_, resp6, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka2)

	// verify that the first request was accepted for an eval type
	Expect(resp5.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp5Body.InstanceType).To(Equal("eval"))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp6.StatusCode).To(Equal(http.StatusTooManyRequests))
	Expect(resp6.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that another external user in the same org can also create the default maximum allowed kafka instances
	externalUserSameOrgAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserSameOrgCtx := h.NewAuthenticatedContext(externalUserSameOrgAccount, nil)

	_, resp7, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka3)
	_, resp8, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka2)

	// verify that the first request was accepted
	Expect(resp7.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp8.StatusCode).To(Equal(http.StatusTooManyRequests))
	Expect(resp8.Header.Get("Content-Type")).To(Equal("application/json"))
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	seedKafka, _, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded kafka request: %s", err.Error())
	}

	// 200 OK
	kafka, resp, err := client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get kafka request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(constants2.KafkaRequestStatusAccepted.String()))
	// When kafka is in 'Accepted' state it means that it still has not been
	// allocated to a cluster, which means that kas fleetshard-sync has not reported
	// yet any status, so the version attribute (actual version) at this point
	// should still be empty
	Expect(kafka.Version).To(Equal(""))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// A different account than the one used to create the Kafka instance, within
	// the same organization than the used account used to create the Kafka instance,
	// should be able to read the Kafka cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	kafka, _, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(kafka.Id).NotTo(BeEmpty())

	// An account in a different organization than the one used to create the
	// Kafka instance should get a 404 Not Found.
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"
	account = h.NewAccountWithNameAndOrg(faker.Name(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// An allowed serviceaccount in config/quota-management-list-configuration.yaml
	// without an organization ID should get a 401 Unauthorized
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestKafka_Delete(t *testing.T) {
	owner := "test-user"

	orgId := "13640203"

	sampleKafkaID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgId)

	userCtx := h.NewAuthenticatedContext(userAccount, nil)

	type args struct {
		ctx     context.Context
		kafkaID string
		async   bool
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting kafka without async set to true",
			args: args{
				ctx:     userCtx,
				kafkaID: sampleKafkaID,
				async:   false,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when deleting kafka with empty id",
			args: args{
				ctx:     userCtx,
				kafkaID: "",
				async:   true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should succeed when deleting kafka with valid id and context",
			args: args{
				ctx:     userCtx,
				kafkaID: sampleKafkaID,
				async:   true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
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

	// create a kafka that will be updated
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID,
		},
		MultiAZ:               true,
		Owner:                 owner,
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        orgId,
		Status:                constants.KafkaRequestStatusReady.String(),
		ClusterID:             cluster.ClusterID,
		ActualKafkaVersion:    "2.6.0",
		DesiredKafkaVersion:   "2.6.0",
		ActualStrimziVersion:  "v.23.0",
		DesiredStrimziVersion: "v0.23.0",
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(tt.args.ctx, tt.args.kafkaID, tt.args.async)
			tt.verifyResponse(resp, err)
		})
	}
}

// TestKafkaDelete_DeleteDuringCreation - test deleting kafka instance during creation
func TestKafkaDelete_DeleteDuringCreation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(ocmConfig *ocm.OCMConfig) {
		if ocmConfig.MockMode == ocm.MockModeEmulateServer {
			// increase repeat interval to allow time to delete the kafka instance before moving onto the next state
			// no need to reset this on teardown as it is always set at the start of each test within the registerIntegrationWithHooks setup for emulated servers.
			workers.RepeatInterval = 10 * time.Second
		}
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	// custom update kafka status implementation to ensure that kafkas created within this test never gets updated to a 'ready' state.
	mockKasFleetshardSyncBuilder.SetUpdateKafkaStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error {
		dataplaneCluster, findDataplaneClusterErr := test.TestServices.ClusterService.FindCluster(services.FindClusterCriteria{
			Status: api.ClusterReady,
		})
		if findDataplaneClusterErr != nil {
			return fmt.Errorf("Unable to retrieve a ready data plane cluster: %v", findDataplaneClusterErr)
		}

		// only update kafka status if a data plane cluster is available
		if dataplaneCluster != nil {
			ctx := kasfleetshardsync.NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)
			kafkaList, _, err := privateClient.AgentClustersApi.GetKafkas(ctx, dataplaneCluster.ClusterID)
			if err != nil {
				return err
			}
			kafkaStatusList := make(map[string]private.DataPlaneKafkaStatus)
			for _, kafka := range kafkaList.Items {
				id := kafka.Metadata.Annotations.Bf2OrgId
				if kafka.Spec.Deleted {
					kafkaStatusList[id] = kasfleetshardsync.GetDeletedKafkaStatusResponse()
				}
			}

			if _, err = privateClient.AgentClustersApi.UpdateKafkaClusterStatus(ctx, dataplaneCluster.ClusterID, kafkaStatusList); err != nil {
				return err
			}
		}
		return nil
	})
	mockKasFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasFleetshardSync.Start()
	defer mockKasFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	// Test deletion of a kafka in an 'accepted' state
	kafka, resp, err := common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'preparing' state
	kafka, resp, err = common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	kafka, err = common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusPreparing)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be preparing: %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'provisioning' state
	kafka, resp, err = common.WaitForKafkaCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	kafka, err = common.WaitForKafkaToReachStatus(ctx, test.TestServices.DBFactory, client, kafka.Id, constants2.KafkaRequestStatusProvisioning)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be provisioning: %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common.WaitForKafkaToBeDeleted(ctx, test.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants2.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants2.KafkaOperationDelete.String()))
}

// TestKafkaDelete - tests fail kafka delete
func TestKafkaDelete_Fail(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	kafka := public.KafkaRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
		Id:            "invalid-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9",
	}

	_, _, err := client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).To(HaveOccurred())
	// The id is invalid, so the metric is not expected to exist
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount), false)
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount), false)
}

func TestKafka_DeleteAdminNonOwner(t *testing.T) {
	owner := "test-user"

	sampleKafkaID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	otherUserAccount := h.NewAccountWithNameAndOrg("non-owner", "13640203")

	adminAccount := h.NewAccountWithNameAndOrg("admin", "13640203")

	otherUserCtx := h.NewAuthenticatedContext(otherUserAccount, nil)

	claims := jwt.MapClaims{
		"is_org_admin": true,
	}

	adminCtx := h.NewAuthenticatedContext(adminAccount, claims)

	type args struct {
		ctx     context.Context
		kafkaID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting kafka by other user within the same org (not an owner)",
			args: args{
				ctx:     otherUserCtx,
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should not fail when deleting kafka by other user within the same org (not an owner but org admin)",
			args: args{
				ctx:     adminCtx,
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a kafka to it
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

	// create a kafka that will be updated
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID,
		},
		MultiAZ:               false,
		Owner:                 owner,
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        "13640203",
		Status:                constants.KafkaRequestStatusReady.String(),
		ClusterID:             cluster.ClusterID,
		ActualKafkaVersion:    "2.6.0",
		DesiredKafkaVersion:   "2.6.0",
		ActualStrimziVersion:  "v.23.0",
		DesiredStrimziVersion: "v0.23.0",
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(tt.args.ctx, tt.args.kafkaID, true)
			tt.verifyResponse(resp, err)
		})
	}
}

// TestKafkaList_Success tests getting kafka requests list
func TestKafkaList_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.GetKafkas(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(initList.Items).To(BeEmpty(), "Expected empty kafka requests list")
	Expect(initList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(initList.Total).To(Equal(int32(0)), "Expected Total == 0")

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	// POST kafka request to populate the list
	seedKafka, _, err := client.DefaultApi.CreateKafka(initCtx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	// get populated list of kafka requests
	afterPostList, _, err := client.DefaultApi.GetKafkas(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected kafka requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get kafka request item from the list
	listItem := afterPostList.Items[0]

	// check whether the seedKafka properties are the same as those from the kafka request list item
	Expect(seedKafka.Id).To(Equal(listItem.Id))
	Expect(seedKafka.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindKafka))
	Expect(seedKafka.Href).To(Equal(listItem.Href))
	Expect(seedKafka.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(clusterservicetest.MockClusterRegion))
	Expect(seedKafka.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest.MockClusterCloudProvider))
	Expect(seedKafka.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockKafkaName))
	Expect(listItem.Status).To(Equal(constants2.KafkaRequestStatusAccepted.String()))

	// new account setup to prove that users can list kafkas instances created by a member of their org
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// get populated list of kafka requests

	afterPostList, _, err = client.DefaultApi.GetKafkas(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected kafka requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get kafka request item from the list
	listItem = afterPostList.Items[0]

	// check whether the seedKafka properties are the same as those from the kafka request list item
	Expect(seedKafka.Id).To(Equal(listItem.Id))
	Expect(seedKafka.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindKafka))
	Expect(seedKafka.Href).To(Equal(listItem.Href))
	Expect(seedKafka.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(clusterservicetest.MockClusterRegion))
	Expect(seedKafka.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest.MockClusterCloudProvider))
	Expect(seedKafka.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockKafkaName))
	Expect(listItem.Status).To(Equal(constants2.KafkaRequestStatusAccepted.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) kafka instances
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)

	// expecting empty list for user that hasn't created any kafkas yet
	newUserList, _, err := client.DefaultApi.GetKafkas(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(newUserList.Items)).To(Equal(0), "Expected kafka requests list length to be 0")
	Expect(newUserList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(newUserList.Total).To(Equal(int32(0)), "Expected Total == 0")
}

// TestKafkaList_InvalidToken - tests listing kafkas with invalid token
func TestKafkaList_UnauthUser(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create empty context
	ctx := context.Background()

	kafkaRequests, resp, err := client.DefaultApi.GetKafkas(ctx, nil)
	Expect(err).To(HaveOccurred()) // expecting an error here due unauthenticated user
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(kafkaRequests.Items).To(BeNil())
	Expect(kafkaRequests.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(kafkaRequests.Total).To(Equal(int32(0)), "Expected Total == 0")
}

func deleteTestKafka(t *testing.T, h *coreTest.Helper, ctx context.Context, client *public.APIClient, kafkaID string) {
	_, _, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaID, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	// wait for kafka to be deleted
	err = common.NewPollerBuilder(test.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(kafkaCheckInterval, kafkaReadyTimeout).
		RetryLogMessagef("Waiting for kafka '%s' to be deleted", kafkaID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			db := test.TestServices.DBFactory.New()
			var kafkaRequest dbapi.KafkaRequest
			if err := db.Unscoped().Where("id = ?", kafkaID).First(&kafkaRequest).Error; err != nil {
				return false, err
			}
			return kafkaRequest.DeletedAt.Valid, nil
		}).Build().Poll()
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)
}

func TestKafkaList_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
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

	_, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestKafkaList_CorrectTokenIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test.TestServices.ServerConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

// TestKafka_RemovingExpiredKafkas_NoStandardInstances tests that all eval kafkas are removed after their allocated life span has expired.
// No standard instances are present
func TestKafka_RemovingExpiredKafkas_NoStandardInstances(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned

	kafkas := []*dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-to-remain",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-49 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              kafkaRegion,
			CloudProvider:       kafkaCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-kafka-2",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.KafkaRequestStatusProvisioning.String(),
			InstanceType:        types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-48 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              kafkaRegion,
			CloudProvider:       kafkaCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-kafka-3",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.KafkaRequestStatusReady.String(),
			InstanceType:        types.EVAL.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 1)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}

// TestKafka_RemovingExpiredKafkas_EmptyList tests that all eval kafkas are removed after their allocated life span has expired
// while standard instances are left behind
func TestKafka_RemovingExpiredKafkas_WithStandardInstances(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

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

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned

	kafkas := []*dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-not-yet-expired",
			OrganisationId: orgId,
			Status:         constants2.KafkaRequestStatusAccepted.String(),
			InstanceType:   types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-49 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              kafkaRegion,
			CloudProvider:       kafkaCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-kafka-to-remove",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.KafkaRequestStatusReady.String(),
			InstanceType:        types.EVAL.String(),
		},
		{
			Meta: api.Meta{
				CreatedAt: time.Now().Add(time.Duration(-48 * time.Hour)),
			},
			MultiAZ:             false,
			Owner:               testuser1,
			Region:              kafkaRegion,
			CloudProvider:       kafkaCloudProvider,
			ClusterID:           clusterID,
			Name:                "dummy-kafka-long-lived",
			OrganisationId:      orgId,
			BootstrapServerHost: "dummy-bootstrap-host",
			SsoClientID:         "dummy-sso-client-id",
			SsoClientSecret:     "dummy-sso-client-secret",
			Status:              constants2.KafkaRequestStatusReady.String(),
			InstanceType:        types.STANDARD.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	account.Organization().ID()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common.WaitForNumberOfKafkaToBeGivenCount(ctx, test.TestServices.DBFactory, client, 2)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}
