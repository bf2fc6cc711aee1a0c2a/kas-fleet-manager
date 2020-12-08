package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	constants "gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/workers"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/common"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	mockKafkaName      = "test-kafka1"
	kafkaReadyTimeout  = time.Minute * 10
	kafkaCheckInterval = time.Second * 10
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
	ctx := h.NewAuthenticatedContext(account)

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
		return foundKafka.Status == constants.KafkaRequestStatusComplete.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become complete: %v", err)
	// final check on the status
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusComplete.String()))
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
	common.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationCreate.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsTotalCount, constants.KafkaOperationCreate.String()))
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
	ctx := h.NewAuthenticatedContext(account)

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

// TestKafkaAllowList_UnauthorizedValidation tests the allow list API access validations is performed when enabled
func TestKafkaAllowList_UnauthorizedValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// create an account with a random organisation id. This is different than the control plance team organisation id which is used by default
	account := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), h.NewUUID())
	ctx := h.NewAuthenticatedContext(account)

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
				_, resp, _ := client.DefaultApi.DeleteKafkaById(ctx, "kafka-id")
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
	ctx := h.NewAuthenticatedContext(ownerAccount)

	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
	}

	// create the first kafka
	_, resp1, _ := client.DefaultApi.CreateKafka(ctx, true, k)

	// create the second kafka
	_, resp2, _ := client.DefaultApi.CreateKafka(ctx, true, k)

	// verify that the first request was accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request errored with 403 forbidden for the ownerAccount
	Expect(resp2.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp2.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of the same org cannot create a new kafka since limit has been reached
	accountInSameOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), orgIdWithLimitOfOne)
	ctx = h.NewAuthenticatedContext(accountInSameOrg)

	// attempt to create kafka for this user account
	_, resp3, _ := client.DefaultApi.CreateKafka(ctx, true, k)

	// verify that the request errored with 403 forbidden for the account in same organisation
	Expect(resp3.StatusCode).To(Equal(http.StatusForbidden))
	Expect(resp3.Header.Get("Content-Type")).To(Equal("application/json"))

	// verify that user of a different organisation can still create kafka instances
	accountInDifferentOrg := h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(accountInDifferentOrg)

	// attempt to create kafka for this user account
	_, resp4, _ := client.DefaultApi.CreateKafka(ctx, true, k)

	// verify that the request was accepted
	Expect(resp4.StatusCode).To(Equal(http.StatusAccepted))
}

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
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
	ctx := h.NewAuthenticatedContext(account)
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
		return foundKafka.Status == constants.KafkaRequestStatusComplete.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become complete: %v", err)
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusComplete.String()))
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
	Expect(err).To(HaveOccurred(), "retrieving non-existent kafka should throw an error")
	Expect(foundKafka.Id).Should(BeEmpty(), " Kafka ID should be deleted")
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDelete.String()))
}

// TestKafkaDelete - test deleting kafka instance during creation
func TestKafkaDelete_DeleteDuringCreation(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	// Set a custom request handler for POST /syncsets with a delay so we can trigger the deletion during Kafka creation
	ocmServerBuilder.SetClusterSyncsetPostRequestHandler(func() func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(30 * time.Second)
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
	ctx := h.NewAuthenticatedContext(account)
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
		return foundKafka.Status == constants.KafkaRequestStatusProvisioning.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to be provisioning: %v", err)
	Expect(foundKafka.Status).To(Equal(constants.KafkaRequestStatusProvisioning.String()))
	Expect(foundKafka.Owner).To(Equal(account.Username()))

	// wait a few seconds to ensure that deletion is triggered during kafka syncset creation
	time.Sleep(kafkaCheckInterval)
	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	// Sleep for worker interval duration to ensure kafka manager reconciliation has finished
	time.Sleep(workers.RepeatInterval)
	kafkaList, _, err := client.DefaultApi.ListKafkas(ctx, &openapi.ListKafkasOpts{})
	Expect(err).NotTo(HaveOccurred(), "Failed to list kafka request: %v", err)
	Expect(kafkaList.Total).Should(BeZero(), " Kafka List response should be empty")
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsSuccessCount, constants.KafkaOperationDelete.String()))
	common.CheckMetricExposed(h, t, fmt.Sprintf("%s_%s{operation=\"%s\"} 1", metrics.ManagedServicesSystem, metrics.KafkaOperationsTotalCount, constants.KafkaOperationDelete.String()))

	// Check status of soft deleted kafka request. Status should not be updated
	db := h.Env().DBFactory.New()
	var kafkaRequest openapi.KafkaRequest
	if err := db.Unscoped().Where("id = ?", kafka.Id).First(&kafkaRequest).Error; err != nil {
		t.Error("failed to find soft deleted kafka request")
	}
	Expect(kafkaRequest.Status).To(Equal(constants.KafkaRequestStatusProvisioning.String()))
}

// TestKafkaDelete - tests fail kafka delete
func TestKafkaDelete_Fail(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
	kafka := openapi.KafkaRequest{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName,
		MultiAz:       testMultiAZ,
		Id:            "invalid-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9",
	}

	_, _, err := client.DefaultApi.DeleteKafkaById(ctx, kafka.Id)
	Expect(err).To(HaveOccurred())
	// The id is invalid, so the metric is not expected to exist
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.ManagedServicesSystem, metrics.KafkaOperationsSuccessCount), false)
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.ManagedServicesSystem, metrics.KafkaOperationsTotalCount), false)
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
	ctx := h.NewAuthenticatedContext(account)
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

	// attempt to delete kafka not created by the owner (should result in an error)
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account)
	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id)
	Expect(err).To(HaveOccurred())
	// user auth is failed, so the metric is not expected to exist
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.ManagedServicesSystem, metrics.KafkaOperationsSuccessCount), false)
	common.CheckMetric(h, t, fmt.Sprintf("%s_%s", metrics.ManagedServicesSystem, metrics.KafkaOperationsTotalCount), false)
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
	ctx := h.NewAuthenticatedContext(account)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.ListKafkas(ctx, nil)
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
	seedKafka, _, err := client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	var foundKafka openapi.KafkaRequest
	_ = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == constants.KafkaRequestStatusComplete.String(), nil
	})

	// get populated list of kafka requests
	afterPostList, _, err := client.DefaultApi.ListKafkas(ctx, nil)
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
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusComplete.String()))

	// new account setup to prove that users can list kafkas instances created by a memeber of their org
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account)

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
	Expect(listItem.Status).To(Equal(constants.KafkaRequestStatusComplete.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a memeber of their org) kafka instances
	// this value if taken from config/allow-list-configuration.yaml
	anotherOrgId := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgId)
	ctx = h.NewAuthenticatedContext(account)

	// expecting empty list for user that hasn't created any kafkas yet
	newUserList, _, err := client.DefaultApi.ListKafkas(ctx, nil)
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
