package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
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

func TestAdminKafka_Get(t *testing.T) {
	sampleKafkaID := api.NewID()
	desiredStrimziVersion := "test"
	type args struct {
		ctx     func(h *coreTest.Helper) context.Context
		kafkaID string
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result adminprivate.Kafka, resp *http.Response, err error)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
				Expect(result.DesiredStrimziVersion).To(Equal(desiredStrimziVersion))
				Expect(result.ClusterId).ShouldNot(BeNil())
			},
		},
		{
			name: "should fail when the requested kafka does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				kafkaID: "unexistingkafkaID",
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()
	db := test.TestServices.DBFactory.New()
	kafka := &dbapi.KafkaRequest{
		MultiAZ:               false,
		Owner:                 "test-user",
		Region:                "test",
		CloudProvider:         "test",
		Name:                  "test-kafka",
		OrganisationId:        "13640203",
		DesiredStrimziVersion: desiredStrimziVersion,
		Status:                constants.KafkaRequestStatusReady.String(),
	}
	kafka.ID = sampleKafkaID

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.GetKafkaById(ctx, tt.args.kafkaID)
			tt.verifyResponse(result, resp, err)
		})
	}
}

func TestAdminKafka_Delete(t *testing.T) {
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

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	kafkaId := api.NewID()
	db := test.TestServices.DBFactory.New()
	kafka := &dbapi.KafkaRequest{
		MultiAZ:        false,
		Owner:          "test-user",
		Region:         "test",
		CloudProvider:  "test",
		Name:           "test-kafka",
		OrganisationId: "13640203",
		Status:         constants.KafkaRequestStatusReady.String(),
	}
	kafka.ID = kafkaId

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaId, true)
			tt.verifyResponse(resp, err)
		})
	}
}

func TestAdminKafka_List(t *testing.T) {
	type args struct {
		ctx           func(h *coreTest.Helper) context.Context
		kafkaListSize int
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, expectedSize int)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				kafkaListSize: 0,
			},
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, expectedSize int) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				Expect(len(kafkaList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(kafkaList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				kafkaListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(kafkaList.Items)).To(Equal(expectedSize))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminReadRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				kafkaListSize: 2,
			},
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, expectedSize int) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(len(kafkaList.Items)).To(Equal(expectedSize))
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

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()

	db := test.TestServices.DBFactory.New()
	kafkas := []dbapi.KafkaRequest{
		{
			MultiAZ:        false,
			Owner:          "test-user1",
			Region:         "test",
			CloudProvider:  "test",
			Name:           "test-kafka1",
			OrganisationId: "13640203",
			Status:         constants.KafkaRequestStatusReady.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "test-user2",
			Region:         "test",
			CloudProvider:  "test",
			Name:           "test-kafka2",
			OrganisationId: "12147054",
			Status:         constants.KafkaRequestStatusReady.String(),
		},
	}

	for _, kafka := range kafkas {
		if err := db.Create(&kafka).Error; err != nil {
			t.Errorf("failed to create Kafka db record due to error: %v", err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			kafkaList, resp, err := client.DefaultApi.GetKafkas(ctx, nil)
			tt.verifyResponse(resp, err, kafkaList, tt.args.kafkaListSize)
		})
	}
}

func TestAdminKafka_Update(t *testing.T) {
	sampleKafkaID := api.NewID()
	fullyPopulatedKafkaUpdateRequest := adminprivate.KafkaUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.23.0-0",
		KafkaVersion:   "2.7.0",
	}
	emptyKafkaUpdateRequest := adminprivate.KafkaUpdateRequest{}
	updateRequestWithKafkaVersion := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "2.7.0",
	}
	updateRequestWithStrimziVersion := adminprivate.KafkaUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.23.0-0",
	}
	type args struct {
		ctx                func(h *coreTest.Helper) context.Context
		kafkaID            string
		kafkaUpdateRequest adminprivate.KafkaUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(result adminprivate.Kafka, resp *http.Response, err error)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID: sampleKafkaID,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: fullyPopulatedKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
			},
		},
		{
			name: fmt.Sprintf("should success when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: fullyPopulatedKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
			},
		},
		{
			name: "should succeed when updating kafka version only",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: updateRequestWithKafkaVersion,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
			},
		},
		{
			name: "should succeed when updating strimzi version only",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: updateRequestWithStrimziVersion,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID))
			},
		},
		{
			name: "should fail when kafkaUpdateRequest is empty",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: emptyKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when the requested kafka does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				kafkaID:            "nonexistentkafkaID",
				kafkaUpdateRequest: fullyPopulatedKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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
				kafkaID:            sampleKafkaID,
				kafkaUpdateRequest: fullyPopulatedKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
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

	h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
	defer tearDown()
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

	cluster.SetAvailableStrimziVersions([]api.StrimziVersion{{
		Version: updateRequestWithStrimziVersion.StrimziVersion,
		Ready:   true,
	}, {
		Ready:   true,
		Version: fullyPopulatedKafkaUpdateRequest.StrimziVersion,
	}})

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
		Owner:                 "test-user",
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
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.UpdateKafkaById(ctx, tt.args.kafkaID, tt.args.kafkaUpdateRequest)
			tt.verifyResponse(result, resp, err)
		})
	}
}
