package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/common"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	mockKafkaName      = "test"
	mockKafkaOwner     = "owner"
	kafkaReadyTimeout  = time.Minute * 10
	kafkaCheckInterval = time.Second * 10
	testMultiAZ        = true
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
	k := openapi.KafkaRequest{
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
		return foundKafka.Status == services.KafkaRequestStatusComplete.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become complete: %v", err)
	// final check on the status
	Expect(foundKafka.Status).To(Equal(services.KafkaRequestStatusComplete.String()))
	// check the owner is set correctly
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))
	common.CheckMetricExposed(h, t, metrics.KafkaCreateRequestDuration)
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
		body     openapi.KafkaRequest
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: openapi.KafkaRequest{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        "us-east-3",
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: openapi.KafkaRequest{
				MultiAz:       mocks.MockCluster.MultiAZ(),
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when MultiAZ false provided",
			body: openapi.KafkaRequest{
				MultiAz:       false,
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: openapi.KafkaRequest{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when id is provided",
			body: openapi.KafkaRequest{
				Id:            mocks.MockCluster.ID(),
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Owner:         mockKafkaOwner,
				Name:          mockKafkaName,
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

// TestKafkaGet tests getting kafkas via the API endpoint
func TestKafkaGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)
	k := openapi.KafkaRequest{
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
	Expect(kafka.Status).To(Equal(services.KafkaRequestStatusAccepted.String()))

	// 404 Not Found
	kafka, resp, err = client.DefaultApi.GetKafkaById(ctx, fmt.Sprintf("not-%s", seedKafka.Id))
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
	k := openapi.KafkaRequest{
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
		return foundKafka.Status == services.KafkaRequestStatusComplete.String(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "Error waiting for kafka request to become complete: %v", err)
	Expect(foundKafka.Status).To(Equal(services.KafkaRequestStatusComplete.String()))
	Expect(foundKafka.Owner).To(Equal(account.Username()))
	Expect(foundKafka.BootstrapServerHost).To(Not(BeEmpty()))

	_, _, err = client.DefaultApi.DeleteKafkaById(ctx, kafka.Id)
	Expect(err).NotTo(HaveOccurred(), "Failed to delete kafka request: %v", err)

	foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, kafka.Id)
	Expect(foundKafka.Id).Should(BeEmpty(), " Kafka ID should be deleted")
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
	k := openapi.KafkaRequest{
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

	k := openapi.KafkaRequest{
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
	err = wait.PollImmediate(kafkaCheckInterval, kafkaReadyTimeout, func() (done bool, err error) {
		foundKafka, _, err = client.DefaultApi.GetKafkaById(ctx, seedKafka.Id)
		if err != nil {
			return true, err
		}
		return foundKafka.Status == services.KafkaRequestStatusComplete.String(), nil
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
	Expect(listItem.Status).To(Equal(services.KafkaRequestStatusComplete.String()))

	// new account setup to prove that users can only list their own kafka instances
	account = h.NewRandAccount()
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
