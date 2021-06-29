package integration

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	test2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	common2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	kasfleetshardsync2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	clusterservicetest2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm/clusterservicetest"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/presenters"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
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
	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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

	kafka, resp, err := common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)

	// kafka successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	foundKafka, err := common2.WaitForKafkaToReachStatus(ctx, test2.TestServices.DBFactory, client, kafka.Id, constants.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
	Expect(foundKafka.DeprecatedBootstrapServerHost).To(Not(BeEmpty()))
	Expect(foundKafka.Version).To(Equal(test2.TestServices.AppConfig.Kafka.DefaultKafkaVersion))

	// checking kafka_request bootstrap server port number being present
	kafka, _, err = client.DefaultApi.GetKafkaById(ctx, foundKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error getting created kafka_request:  %v", err)
	Expect(strings.HasSuffix(kafka.BootstrapServerHost, ":443")).To(Equal(true))
	Expect(kafka.Version).To(Equal(test2.TestServices.AppConfig.Kafka.DefaultKafkaVersion))

	db := test2.TestServices.DBFactory.New()
	var kafkaRequest api.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}
	Expect(kafkaRequest.SsoClientID).To(BeEmpty())
	Expect(kafkaRequest.SsoClientSecret).To(BeEmpty())
	Expect(kafkaRequest.QuotaType).To(Equal(h.Env.Config.Kafka.Quota.Type))
	Expect(kafkaRequest.PlacementId).To(Not(BeEmpty()))

	common2.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationCreate.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationCreate.String()))

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
	h, client, tearDown := test2.NewKafkaHelperWithHooks(t, ocmServer, func(c *config.KafkaConfig) {
		c.KafkaCapacity.MaxCapacity = 2
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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
	// this value is taken from config/allow-list-configuration.yaml
	orgId := "13640203"

	// create dummy kafkas
	db := test2.TestServices.DBFactory.New()
	kafkas := []*api.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-2",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
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

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
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

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create two random accounts in same organisation
	account1 := h.NewRandAccount()
	ctx1 := h.NewAuthenticatedContext(account1, nil)

	account2 := h.NewRandAccount()
	ctx2 := h.NewAuthenticatedContext(account2, nil)

	// this value if taken from config/allow-list-configuration.yaml
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

// TestKafkaAllowList_UnauthorizedValidation tests the allow list API access validations is performed when enabled
func TestKafkaAllowList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	account := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), h.NewUUID())
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			resp := tt.operation()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
		})
	}
}

// TestKafkaDenyList_UnauthorizedValidation tests the deny list API access validations is performed when enabled
func TestKafkaDenyList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// this value if taken from config/deny-list-configuration.yaml
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

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, err := common2.GetRunningOsdClusterID(h, t)
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

	// this value is taken from config/allow-list-configuration.yaml
	orgId := "13640203"

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test2.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	kafkas := []*api.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          username1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-2",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
		},
		{
			MultiAZ:        false,
			Owner:          username2,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			ClusterID:      clusterID,
			Name:           "dummy-kafka-3",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusPreparing.String(),
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
			Status:              constants.KafkaRequestStatusProvisioning.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "some-other-user",
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "this-kafka-will-remain",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// Verify all Kafkas that needs to be deleted are removed
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common2.WaitForNumberOfKafkaToBeGivenCount(ctx, test2.TestServices.DBFactory, client, 1)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for first kafka deletion: %v", kafkaDeletionErr)
}

// TestKafkaAllowList_MaxAllowedInstances tests the allow list max allowed instances creation validations is performed when enabled
// The max allowed instances limit is set to "1" set for one organusation is not depassed.
// At the same time, users of a different organisation should be able to create instances.
func TestKafkaAllowList_MaxAllowedInstances(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelperWithHooks(t, ocmServer, func(acl *config.AccessControlListConfig) {
		acl.AllowList.AllowAnyRegisteredUsers = true
		acl.EnableInstanceLimitControl = true
	})
	defer teardown()

	// this value if taken from config/allow-list-configuration.yaml
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
	_, resp1, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)

	// create the second kafka
	_, resp2, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)

	// verify that the first request was accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request errored with 403 forbidden for the ownerAccount
	Expect(resp2.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp2.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of the same org cannot create a new kafka since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	internalUserCtx = h.NewAuthenticatedContext(accountInSameOrg, nil)

	// attempt to create kafka for this user account
	_, resp3, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka2)

	// verify that the request errored with 403 forbidden for the account in same organisation
	Expect(resp3.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp3.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of a different organisation can still create kafka instances
	accountInDifferentOrg := h.NewRandAccount()
	internalUserCtx = h.NewAuthenticatedContext(accountInDifferentOrg, nil)

	// attempt to create kafka for this user account
	_, resp4, _ := client.DefaultApi.CreateKafka(internalUserCtx, true, kafka1)

	// verify that the request was accepted
	Expect(resp4.StatusCode).To(Equal(http.StatusAccepted))

	// verify that an external user not listed in the allow list can create the default maximum allowed kafka instances
	externalUserAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserCtx := h.NewAuthenticatedContext(externalUserAccount, nil)

	_, resp5, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka1)
	_, resp6, _ := client.DefaultApi.CreateKafka(externalUserCtx, true, kafka2)

	// verify that the first request was accepted
	Expect(resp5.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp6.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp6.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that another external user in the same org can also create the default maximum allowed kafka instances
	externalUserSameOrgAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), "ext-org-id")
	externalUserSameOrgCtx := h.NewAuthenticatedContext(externalUserSameOrgAccount, nil)

	_, resp7, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka3)
	_, resp8, _ := client.DefaultApi.CreateKafka(externalUserSameOrgCtx, true, kafka2)

	// verify that the first request was accepted
	Expect(resp7.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request for the external user errored with 403 Forbidden
	Expect(resp8.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp8.Header.Get("Content-Type")).To(Equal("application/json"))
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
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
	Expect(kafka.Status).To(Equal(constants.KafkaRequestStatusAccepted.String()))
	Expect(kafka.Version).To(Equal(test2.TestServices.AppConfig.Kafka.DefaultKafkaVersion))

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
	// this value if taken from config/allow-list-configuration.yaml
	anotherOrgID := "12147054"
	account = h.NewAccountWithNameAndOrg(faker.Name(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// An allowed serviceaccount in config/allow-list-configuration.yaml
	// without an organization ID should get a 401 Unauthorized
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

// TestKafkaDelete - tests Success kafka delete
func TestKafkaDelete_Success(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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

	kafka, resp, err := common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))

	foundKafka, err := common2.WaitForKafkaToReachStatus(ctx, test2.TestServices.DBFactory, client, kafka.Id, constants.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
	Expect(foundKafka.DeprecatedBootstrapServerHost).To(Not(BeEmpty()))

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "It should be possible to retrieve kafka marked for deletion")
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))

	// wait for kafka to be deleted
	err = common2.WaitForKafkaToBeDeleted(ctx, test2.TestServices.DBFactory, client, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)
}

// TestKafkaDelete - tests Fail kafka delete if calling as sync
func TestKafkaDelete_FailSync(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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

	kafka, resp, err := common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/kafkas_mgmt/v1/kafkas/%s", kafka.Id)))

	_, resp, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, false)
	Expect(err).To(HaveOccurred(), "Failed to delete kafka request: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest), "Status code for delete kafka response with async 'false' should be %d. Got: %d", http.StatusBadRequest, resp.StatusCode)
}

// TestKafkaDelete_WithoutID - should result in 405 code being returned
func TestKafkaDelete_WithoutID(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	_, resp, err := client.DefaultApi.DeleteKafkaById(ctx, "", true)
	Expect(err).To(HaveOccurred(), "Error should be thrown if no ID is provided: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed), "Status code for delete kafka response without kafka ID should be %d. Got: %d", http.StatusMethodNotAllowed, resp.StatusCode)
	Expect(resp.Body).NotTo(Equal(""), "Body should be returned when trying to hit /kafkas delete without kafka ID")
}

// TestKafkaDelete - test deleting kafka instance during creation
func TestKafkaDelete_DeleteDuringCreation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelperWithHooks(t, ocmServer, func(ocmConfig *config.OCMConfig) {
		if ocmConfig.MockMode == config.MockModeEmulateServer {
			// increase repeat interval to allow time to delete the kafka instance before moving onto the next state
			// no need to reset this on teardown as it is always set at the start of each test within the registerIntegrationWithHooks setup for emulated servers.
			workers.RepeatInterval = 10 * time.Second
		}
	})
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	// custom update kafka status implementation to ensure that kafkas created within this test never gets updated to a 'ready' state.
	mockKasFleetshardSyncBuilder.SetUpdateKafkaStatusFunc(func(helper *test.Helper, privateClient *private.APIClient) error {
		dataplaneCluster, findDataplaneClusterErr := test2.TestServices.ClusterService.FindCluster(services.FindClusterCriteria{
			Status: api.ClusterReady,
		})
		if findDataplaneClusterErr != nil {
			return fmt.Errorf("Unable to retrieve a ready data plane cluster: %v", findDataplaneClusterErr)
		}

		// only update kafka status if a data plane cluster is available
		if dataplaneCluster != nil {
			ctx := kasfleetshardsync2.NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)
			kafkaList, _, err := privateClient.AgentClustersApi.GetKafkas(ctx, dataplaneCluster.ClusterID)
			if err != nil {
				return err
			}
			kafkaStatusList := make(map[string]private.DataPlaneKafkaStatus)
			for _, kafka := range kafkaList.Items {
				id := kafka.Metadata.Annotations.Id
				if kafka.Spec.Deleted {
					kafkaStatusList[id] = kasfleetshardsync2.GetDeletedKafkaStatusResponse()
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

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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
	kafka, resp, err := common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common2.WaitForKafkaToBeDeleted(ctx, test2.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDelete.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'preparing' state
	kafka, resp, err = common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	kafka, err = common2.WaitForKafkaToReachStatus(ctx, test2.TestServices.DBFactory, client, kafka.Id, constants.KafkaRequestStatusPreparing)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be preparing: %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common2.WaitForKafkaToBeDeleted(ctx, test2.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDelete.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 2", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDelete.String()))

	// Test deletion of a kafka in a 'provisioning' state
	kafka, resp, err = common2.WaitForKafkaCreateToBeAccepted(ctx, test2.TestServices.DBFactory, client, k)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(err).NotTo(HaveOccurred(), "Error waiting for accepted kafka:  %v", err)

	kafka, err = common2.WaitForKafkaToReachStatus(ctx, test2.TestServices.DBFactory, client, kafka.Id, constants.KafkaRequestStatusProvisioning)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be provisioning: %v", err)

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	_ = common2.WaitForKafkaToBeDeleted(ctx, test2.TestServices.DBFactory, client, kafka.Id)
	kafkaList, _, err = client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka list response should be empty")

	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDelete.String()))
	common2.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 3", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDelete.String()))
}

// TestKafkaDelete - tests fail kafka delete
func TestKafkaDelete_Fail(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
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
	common2.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount), false)
	common2.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount), false)
}

// TestKafkaDelete_NonOwnerDelete
// tests delete kafka by the user who hasn't created that kafka instance
func TestKafkaDelete_NonOwnerDelete(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)
	k := public.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka, resp, err := common2.WaitForKafkaCreateToBeAccepted(initCtx, test2.TestServices.DBFactory, client, k)

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")

	foundKafka, err := common2.WaitForKafkaToReachStatus(initCtx, test2.TestServices.DBFactory, client, kafka.Id, constants.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)

	// attempt to delete kafka not created by the owner (should result in an error)
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).To(HaveOccurred())

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, initCtx, client, foundKafka.Id)
}

// TestKafkaList_Success tests getting kafka requests list
func TestKafkaList_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
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

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
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

	foundKafka, err := common2.WaitForKafkaToReachStatus(initCtx, test2.TestServices.DBFactory, client, seedKafka.Id, constants.KafkaRequestStatusReady)
	Expect(err).NotTo(HaveOccurred(), "Error occurred while waiting for kafka to be ready:  %v", err)

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
	Expect(listItem.Region).To(Equal(clusterservicetest2.MockClusterRegion))
	Expect(seedKafka.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest2.MockClusterCloudProvider))
	Expect(seedKafka.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockKafkaName))
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

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
	Expect(listItem.Region).To(Equal(clusterservicetest2.MockClusterRegion))
	Expect(seedKafka.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(clusterservicetest2.MockClusterCloudProvider))
	Expect(seedKafka.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockKafkaName))
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) kafka instances
	// this value if taken from config/allow-list-configuration.yaml
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

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(t, h, initCtx, client, foundKafka.Id)
}

// TestKafkaList_InvalidToken - tests listing kafkas with invalid token
func TestKafkaList_UnauthUser(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, client, teardown := test2.NewKafkaHelper(t, ocmServer)
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

func deleteTestKafka(t *testing.T, h *test.Helper, ctx context.Context, client *public.APIClient, kafkaID string) {
	_, _, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaID, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	// wait for kafka to be deleted
	err = common2.NewPollerBuilder(test2.TestServices.DBFactory).
		OutputFunction(t.Logf).
		IntervalAndTimeout(kafkaCheckInterval, kafkaReadyTimeout).
		RetryLogMessagef("Waiting for kafka '%s' to be deleted", kafkaID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			db := test2.TestServices.DBFactory.New()
			var kafkaRequest api.KafkaRequest
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
	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
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

func TestKafkaList_CorrectOCMIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test2.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test2.TestServices.OCMConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.GetKafkas(ctx, &public.GetKafkasOpts{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

// TestKafka_RemovingExpiredKafkas_EmptyList tests that all kafkas are removed after their allocated life span has expired
func TestKafka_RemovingExpiredKafkas_EmptyLongLivedKafkasList(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test2.NewKafkaHelperWithHooks(t, ocmServer, func(c *config.KafkaConfig) {
		c.KafkaLifespan.LongLivedKafkas = []string{}
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account with values from config/allow-list-configuration.yaml
	testuser1 := "testuser1@example.com"
	orgId := "13640203"

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test2.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned

	kafkas := []*api.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-to-remain",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
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
			Status:              constants.KafkaRequestStatusProvisioning.String(),
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
			Status:              constants.KafkaRequestStatusReady.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common2.WaitForNumberOfKafkaToBeGivenCount(ctx, test2.TestServices.DBFactory, client, 1)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}

// TestKafka_RemovingExpiredKafkas_EmptyList tests that all kafkas are removed after their allocated life span has expired
func TestKafka_RemovingExpiredKafkas_NonEmptyLongLivedKafkaList(t *testing.T) {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test2.NewKafkaHelperWithHooks(t, ocmServer, func(c *config.KafkaConfig) {
		c.KafkaLifespan.LongLivedKafkas = []string{}
	})
	defer tearDown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync2.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common2.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create an account with values from config/allow-list-configuration.yaml
	testuser1 := "testuser1@example.com"
	orgId := "13640203"

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := test2.TestServices.DBFactory.New()
	kafkaRegion := "dummy"        // set to dummy as we do not want this cluster to be provisioned
	kafkaCloudProvider := "dummy" // set to dummy as we do not want this cluster to be provisioned
	// set the long lived kafka id list at the beginning of the tests to avoid potential timing issues when testing its case
	longLivedKafkaId := "123456"
	h.Env.Config.Kafka.KafkaLifespan.LongLivedKafkas = []string{longLivedKafkaId}

	kafkas := []*api.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          testuser1,
			Region:         kafkaRegion,
			CloudProvider:  kafkaCloudProvider,
			Name:           "dummy-kafka-not-yet-expired",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
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
			Status:              constants.KafkaRequestStatusReady.String(),
		},
		{
			Meta: api.Meta{
				ID:        longLivedKafkaId,
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
			Status:              constants.KafkaRequestStatusReady.String(),
		},
	}

	if err := db.Create(&kafkas).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// also verify that any kafkas whose life has expired has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	kafkaDeletionErr := common2.WaitForNumberOfKafkaToBeGivenCount(ctx, test2.TestServices.DBFactory, client, 2)
	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for kafka deletion: %v", kafkaDeletionErr)
}
