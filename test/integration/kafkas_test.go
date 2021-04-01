package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusterservicetest"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	mockKafkaName      = "test-kafka1"
	kafkaReadyTimeout  = time.Minute * 10
	kafkaDeleteTimeout = time.Minute * 10
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
// - kafka becomes ready once syncset is created
func TestKafkaCreate_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// POST responses per openapi spec: 201, 409, 500
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	var kafka openapi.KafkaRequest
	var resp *http.Response
	err := wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
		if err != nil {
			return true, err
		}
		return resp.StatusCode == http.StatusAccepted, err
	})

	// kafka successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	// wait until the kafka goes into a ready state
	// the timeout here assumes a backing cluster has already been provisioned
	var foundKafka openapi.KafkaRequest
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	// final check on the status
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusReady.String()))
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))

	// checking kafka_request bootstrap server port number being present
	kafka, _, err = client.DefaultApi.GetKafkaById(ctx, foundKafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Error getting created kafka_request:  %v", err)
	Expect(strings.HasSuffix(kafka.BootstrapServerHost, ":443")).To(Equal(true))

	db := h.Env().DBFactory.New()
	var kafkaRequest api.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find kafka request")
	}
	Expect(kafkaRequest.SsoClientID).To(BeEmpty())
	Expect(kafkaRequest.SsoClientSecret).To(BeEmpty())

	common.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationCreate.String()))

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(ctx, client, foundKafka.Id)
}

func TestKafkaCreate_TooManyKafkas(t *testing.T) {
	startHook := func(h *test.Helper) {
		h.Env().Config.Kafka.KafkaCapacity.MaxCapacity = 2
	}
	tearDownHook := func(h *test.Helper) {
		h.Env().Config.Kafka.KafkaCapacity.MaxCapacity = 1000
	}
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.RegisterIntegrationWithHooks(t, ocmServer, startHook, tearDownHook)
	defer tearDown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
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
	db := h.Env().DBFactory.New()
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

	for _, kafka := range kafkas {
		if err := db.Save(kafka).Error; err != nil {
			Expect(err).NotTo(HaveOccurred())
			return
		}
	}

	k := openapi.KafkaRequestPayload{
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name     string
		body     openapi.KafkaRequestPayload
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: openapi.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        "us-east-3",
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: openapi.KafkaRequestPayload{
				MultiAz:       mocks.MockCluster.MultiAZ(),
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when MultiAZ false provided",
			body: openapi.KafkaRequestPayload{
				MultiAz:       false,
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: openapi.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is not valid",
			body: openapi.KafkaRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          invalidKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is too long",
			body: openapi.KafkaRequestPayload{
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
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

	k := openapi.KafkaRequestPayload{
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
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
				_, resp, _ := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new kafka request",
			operation: func() *http.Response {
				body := openapi.KafkaRequestPayload{
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
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
				_, resp, _ := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
				return resp
			},
		},
		{
			name: "HTTP 403 when creating a new kafka request",
			operation: func() *http.Response {
				body := openapi.KafkaRequestPayload{
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

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plane team organisation id which is used by default
	// these values are taken from config/deny-list-configuration.yaml
	username1 := "denied-test-user1@example.com"
	username2 := "denied-test-user2@example.com"

	// this value is taken from config/allow-list-configuration.yaml
	orgId := "13640203"

	// create dummy kafkas and assign it to user, at the end we'll verify that the kafka has been deleted
	db := h.Env().DBFactory.New()
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
			Name:           "dummy-kafka-3",
			OrganisationId: orgId,
			Status:         constants.KafkaRequestStatusAccepted.String(),
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

	for _, kafka := range kafkas {
		if err := db.Save(kafka).Error; err != nil {
			Expect(err).NotTo(HaveOccurred())
			return
		}
	}

	// also verify that any kafkas held by the first user has been deleted.
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	kafkaDeletionErr := wait.PollImmediate(kafkaCheckInterval, kafkaDeleteTimeout, func() (done bool, err error) {
		list, _, err := client.DefaultApi.ListKafkas(ctx, nil)
		return list.Size == 1, err
	})

	Expect(kafkaDeletionErr).NotTo(HaveOccurred(), "Error waiting for first kafka deletion: %v", kafkaDeletionErr)
}

// TestKafkaAllowList_MaxAllowedInstances tests the allow list max allowed instances creation validations is performed when enabled
// The max allowed instances limit is set to "1" set for one organusation is not depassed.
// At the same time, users of a different organisation should be able to create instances.
func TestKafkaAllowList_MaxAllowedInstances(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// this value if taken from config/allow-list-configuration.yaml
	orgIdWithLimitOfOne := "12147054"
	ownerAccount := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	ctx := h.NewAuthenticatedContext(ownerAccount, nil)

	kafka1 := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	kafka2 := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          "test-kafka-2",
		MultiAz:       testMultiAZ,
	}

	// create the first kafka
	_, resp1, _ := client.DefaultApi.CreateKafka(ctx, true, kafka1)

	// create the second kafka
	_, resp2, _ := client.DefaultApi.CreateKafka(ctx, true, kafka2)

	// verify that the first request was accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request errored with 403 forbidden for the ownerAccount
	Expect(resp2.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp2.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of the same org cannot create a new kafka since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	ctx = h.NewAuthenticatedContext(accountInSameOrg, nil)

	// attempt to create kafka for this user account
	_, resp3, _ := client.DefaultApi.CreateKafka(ctx, true, kafka2)

	// verify that the request errored with 403 forbidden for the account in same organisation
	Expect(resp3.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp3.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of a different organisation can still create kafka instances
	accountInDifferentOrg := h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(accountInDifferentOrg, nil)

	// attempt to create kafka for this user account
	_, resp4, _ := client.DefaultApi.CreateKafka(ctx, true, kafka1)

	// verify that the request was accepted
	Expect(resp4.StatusCode).To(Equal(http.StatusAccepted))
}

func TestKafkaAllowList_FixMGDSTRM_1052(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// a random organisationId to make sure that it is set in context
	orgId := "12345688097"

	// the values are taken from config/allow-list-configuration.yaml
	email1 := "testuser2@example.com"
	email2 := "testuser3@example.com"

	serviceAccount1 := h.NewAccount(email1, faker.Name(), email1, orgId)
	serviceAccount2 := h.NewAccount(email2, faker.Name(), email2, orgId)

	ctx1 := h.NewAuthenticatedContext(serviceAccount1, nil)
	ctx2 := h.NewAuthenticatedContext(serviceAccount2, nil)

	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	// create the first kafka for first service account
	kafka1, resp1, _ := client.DefaultApi.CreateKafka(ctx1, true, k)

	// create the second kafka for the second service account
	_, resp2, _ := client.DefaultApi.CreateKafka(ctx2, true, k)

	// verify that the two requests were accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp2.StatusCode).To(Equal(http.StatusAccepted))

	// verify get works for owner and not the other
	_, respGet1, _ := client.DefaultApi.GetKafkaById(ctx1, kafka1.Id)
	_, respGet2, _ := client.DefaultApi.GetKafkaById(ctx2, kafka1.Id)
	// the owner should get the kafka
	Expect(respGet1.StatusCode).To(Equal(http.StatusOK))
	// non owner should not get the kafka
	Expect(respGet2.StatusCode).To(Equal(http.StatusNotFound))

	// check the list of kafkas size for the first service account to equal one
	list1, list1Resp, list1Err := client.DefaultApi.ListKafkas(ctx1, nil)
	Expect(list1Err).NotTo(HaveOccurred())
	Expect(list1Resp.StatusCode).To(Equal(http.StatusOK))
	Expect(list1.Size).To(Equal(int32(1)))
	Expect(list1.Total).To(Equal(int32(1)))

	// check the list of kafkas size for the second service account to equal one
	list2, list2Resp, list2Err := client.DefaultApi.ListKafkas(ctx2, nil)
	Expect(list2Err).NotTo(HaveOccurred())
	Expect(list2Resp.StatusCode).To(Equal(http.StatusOK))
	Expect(list2.Size).To(Equal(int32(1)))
	Expect(list2.Total).To(Equal(int32(1)))
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := openapi.KafkaRequestPayload{
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
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))
	Expect(kafka.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(kafka.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(kafka.Name).To(Equal(mockKafkaName))
	Expect(kafka.Status).To(Equal(constants.KafkaRequestStatusAccepted.String()))

	// 404 Not Found
	kafka, resp, _ = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// different account but same org, should be able to read the Kafka cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	kafka, _, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(kafka.Id).NotTo(BeEmpty())

	// a serviceaccount that doesn't have orgId, and owner field is different too so it should get 404
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

// TestKafkaDelete - tests Success kafka delete
func TestKafkaDelete_Success(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	var kafka openapi.KafkaRequest
	var resp *http.Response
	err := wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
		if err != nil {
			return true, err
		}
		return resp.StatusCode == http.StatusAccepted, err
	})

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	var foundKafka openapi.KafkaRequest
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusReady.String()))
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "It should be possible to retrieve kafka marked for deletion")
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))

	// wait for kafka to be deleted
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		if _, _, err := client.DefaultApi.GetKafkaById(ctx, kafka.Id); err != nil {
			if err.Error() == "404 Not Found" {
				return true, nil
			}

			return false, err
		}
		return false, nil
	})
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)
}

// TestKafkaDelete - tests Fail kafka delete if calling as sync
func TestKafkaDelete_FailSync(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	var kafka openapi.KafkaRequest
	var resp *http.Response
	err := wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
		if err != nil {
			return true, err
		}
		return resp.StatusCode == http.StatusAccepted, err
	})

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	_, resp, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, false)
	Expect(err).To(HaveOccurred(), "Failed to delete kafka request: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest), "Status code for delete kafka response with async 'false' should be %d. Got: %d", http.StatusBadRequest, resp.StatusCode)
}

// TestKafkaDelete_WithoutID - should result in 405 code being returned
func TestKafkaDelete_WithoutID(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
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
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	// Set a custom request handler for POST /syncsets with a delay so we can trigger the deletion during Kafka creation
	ocmServerBuilder.SetClusterSyncsetPostRequestHandler(func() func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second * 1)
			w.Header().Set("Content-Type", "application/json")
			if err := clustersmgmtv1.MarshalSyncset(mocks.MockSyncset, w); err != nil {
				t.Error(err)
			}
		}
	})
	ocmServer := ocmServerBuilder.Build()

	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// cannot reproduce in real ocm environment, only run on mock mode.
	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
		t.SkipNow()
	}

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	var kafka openapi.KafkaRequest
	var resp *http.Response
	err := wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		kafka, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
		if err != nil {
			return true, err
		}
		return resp.StatusCode == http.StatusAccepted, err
	})

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(kafka.Kind).To(Equal(presenters.KindKafka))
	Expect(kafka.Href).To(Equal(fmt.Sprintf("/api/managed-services-api/v1/kafkas/%s", kafka.Id)))

	var foundKafka openapi.KafkaRequest
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusPreparing.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be provisioning: %v", err)
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusPreparing.String()))
	Expect(foundKafka.Owner).To(Equal(account.Username()))

	// wait a few seconds to ensure that deletion is triggered during kafka syncset creation
	time.Sleep(kafkaCheckInterval)
	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	// Sleep for worker interval duration to ensure kafka manager reconciliation has finished: 1 time for
	// updating the status to `deprovision` and one time for the real deletion
	time.Sleep(workers.RepeatInterval * 2)
	kafkaList, _, err := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka List response should be empty")
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDeprovision.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.KasFleetManager, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDeprovision.String()))

	// Check status of soft deleted kafka request. Status should not be 'deprovision'
	db := h.Env().DBFactory.New()
	var kafkaRequest openapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find soft deleted kafka request")
	}
	Expect(kafkaRequest.Status).To(Equal(constants.KafkaRequestStatusDeprovision.String()))
}

// TestKafkaDelete - tests fail kafka delete
func TestKafkaDelete_Fail(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	kafka := openapi.KafkaRequest{
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

// TestKafkaDelete_NonOwnerDelete
// tests delete kafka by the user who hasn't created that kafka instance
func TestKafkaDelete_NonOwnerDelete(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)
	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	var kafka openapi.KafkaRequest
	var resp *http.Response
	err := wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		kafka, resp, err = client.DefaultApi.CreateKafka(initCtx, true, k)
		if err != nil {
			return true, err
		}
		return resp.StatusCode == http.StatusAccepted, err
	})

	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(kafka.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")

	var foundKafka openapi.KafkaRequest
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(initCtx, kafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become ready: %v", err)

	// attempt to delete kafka not created by the owner (should result in an error)
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id, true)
	Expect(err).To(HaveOccurred())

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(initCtx, client, foundKafka.Id)
}

// TestKafkaList_Success tests getting kafka requests list
func TestKafkaList_Success(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.ListKafkas(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(initList.Items).To(BeEmpty(), "Expected empty kafka requests list")
	Expect(initList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(initList.Total).To(Equal(int32(0)), "Expected Total == 0")

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	k := openapi.KafkaRequestPayload{
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

	var foundKafka openapi.KafkaRequest
	_ = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(initCtx, seedKafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusReady.String(), nil
	})

	// get populated list of kafka requests
	afterPostList, _, err := client.DefaultApi.ListKafkas(initCtx, nil)
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
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

	// new account setup to prove that users can list kafkas instances created by a member of their org
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// get populated list of kafka requests

	afterPostList, _, err = client.DefaultApi.ListKafkas(ctx, nil)
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
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusReady.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) kafka instances
	// this value if taken from config/allow-list-configuration.yaml
	anotherOrgID := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)

	// expecting empty list for user that hasn't created any kafkas yet
	newUserList, _, err := client.DefaultApi.ListKafkas(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(newUserList.Items)).To(Equal(0), "Expected kafka requests list length to be 0")
	Expect(newUserList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(newUserList.Total).To(Equal(int32(0)), "Expected Total == 0")

	// delete test kafka to free up resources on an OSD cluster
	deleteTestKafka(initCtx, client, foundKafka.Id)
}

// TestKafkaList_InvalidToken - tests listing kafkas with invalid token
func TestKafkaList_UnauthUser(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// create empty context
	ctx := context.Background()

	kafkaRequests, resp, err := client.DefaultApi.ListKafkas(ctx, nil)
	Expect(err).To(HaveOccurred()) // expecting an error here due unauthenticated user
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	Expect(kafkaRequests.Items).To(BeNil())
	Expect(kafkaRequests.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(kafkaRequests.Total).To(Equal(int32(0)), "Expected Total == 0")
}

func deleteTestKafka(ctx context.Context, client *openapi.APIClient, kafkaID string) {
	_, _, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaID, true)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	// wait for kafka to be deleted
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		if _, _, err := client.DefaultApi.GetKafkaById(ctx, kafkaID); err != nil {
			if err.Error() == "404 Not Found" {
				return true, nil
			}

			return false, err
		}
		return false, nil
	})
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)
}

func TestKafkaList_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      "invalidiss",
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestKafkaList_CorrectOCMIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      h.Env().Config.OCM.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}
