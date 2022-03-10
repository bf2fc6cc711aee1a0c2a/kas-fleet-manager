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
				Expect(result.AccountNumber).ToNot(BeEmpty())
				Expect(result.Namespace).ToNot(BeEmpty())
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
				Expect(result.Namespace).ToNot(BeEmpty())
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
				Expect(result.Namespace).ToNot(BeEmpty())
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
		Namespace:             fmt.Sprintf("kafka-%s", sampleKafkaID),
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
	sampleKafkaID1 := api.NewID()
	sampleKafkaID2 := api.NewID()
	sampleKafkaID3 := api.NewID()
	sampleKafkaID4 := api.NewID()
	fullyPopulatedKafkaVersionUpdateRequest := adminprivate.KafkaUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		KafkaVersion:    "2.8.2",
		KafkaIbpVersion: "2.8.1",
	}
	emptyKafkaUpdateRequest := adminprivate.KafkaUpdateRequest{}
	ibpDowngrade := adminprivate.KafkaUpdateRequest{
		KafkaIbpVersion: "2.6.0",
	}
	ibpUpgrade := adminprivate.KafkaUpdateRequest{
		KafkaIbpVersion: "2.8.0",
	}
	ibpUpgradeHigherThanKafka := adminprivate.KafkaUpdateRequest{
		KafkaIbpVersion: "2.8.2",
	}
	kafkaVersionMajorDowngrade := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "1.8.0",
	}
	kafkaVersionMinorDowngrade := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "2.7.0",
	}
	kafkaVersionPatchDowngrade := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "2.8.0",
	}
	kafkaVersionPatchUpgradeInStatus := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "2.9.0",
	}
	kafkaVersionPatchUpgradeNotInStatus := adminprivate.KafkaUpdateRequest{
		KafkaVersion: "2.10.0",
	}
	strimziUpgrade := adminprivate.KafkaUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.25.0-0",
	}
	nonExistentStrimziUpgrade := adminprivate.KafkaUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.29.0-0",
	}
	notReadyStrimziUpgrade := adminprivate.KafkaUpdateRequest{
		StrimziVersion: "strimzi-cluster-operator.v0.25.1-0",
	}
	allVersionsUpgrade := adminprivate.KafkaUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.25.0-0",
		KafkaIbpVersion: "2.8.1",
		KafkaVersion:    "2.8.2",
	}
	versionsNotInStatusForValidStrimziVersion := adminprivate.KafkaUpdateRequest{
		StrimziVersion:  "strimzi-cluster-operator.v0.25.0-0",
		KafkaIbpVersion: "2.8.3",
		KafkaVersion:    "2.8.3",
	}

	allFieldsUpdateRequest := adminprivate.KafkaUpdateRequest{
		StrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		KafkaVersion:     "2.8.2",
		KafkaIbpVersion:  "2.8.1",
		KafkaStorageSize: "70Gi",
	}

	allFieldsEmptyParams := adminprivate.KafkaUpdateRequest{
		StrimziVersion:   " ",
		KafkaVersion:     " ",
		KafkaIbpVersion:  " ",
		KafkaStorageSize: " ",
	}

	sameStorageSizeUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "60Gi",
	}

	biggerStorageUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "70Gi",
	}

	biggerStorageDifferentFormatUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "75000Mi",
	}

	smallerStorageUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "50Gi",
	}

	wrongFormatStorageUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "70Gb",
	}

	randomStringStorageUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "M2h9O8wO7k",
	}

	smallerStorageDifferentFormatUpdateRequest := adminprivate.KafkaUpdateRequest{
		KafkaStorageSize: "50000Mi",
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
				kafkaID: sampleKafkaID1,
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
				kafkaID: sampleKafkaID1,
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
				kafkaID: sampleKafkaID1,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: "should fail when kafkaUpdateRequest is empty",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: emptyKafkaUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when kafkaUpdateRequest request params contain only strings with whitespaces",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: allFieldsEmptyParams,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: ibpDowngrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version to version higher than kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: ibpUpgradeHigherThanKafka,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID2,
				kafkaUpdateRequest: ibpUpgrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower minor kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: kafkaVersionMinorDowngrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower patch kafka version smaller than ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID3,
				kafkaUpdateRequest: kafkaVersionPatchDowngrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower major kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID4,
				kafkaUpdateRequest: kafkaVersionMajorDowngrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading to higher minor kafka version when not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: kafkaVersionPatchUpgradeNotInStatus,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading kafka version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID2,
				kafkaUpdateRequest: kafkaVersionPatchUpgradeInStatus,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID2,
				kafkaUpdateRequest: strimziUpgrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version not in the status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: nonExistentStrimziUpgrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading simultaneously kafka, ibp and strimzi version when any of those not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: versionsNotInStatusForValidStrimziVersion,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version that is not ready",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID2,
				kafkaUpdateRequest: notReadyStrimziUpgrade,
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
				kafkaUpdateRequest: fullyPopulatedKafkaVersionUpdateRequest,
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
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: fullyPopulatedKafkaVersionUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: fullyPopulatedKafkaVersionUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
			},
		},
		{
			name: "should not fail when attempting to upgrade to the same storage size",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: sameStorageSizeUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.KafkaStorageSize).To(Equal(sameStorageSizeUpdateRequest.KafkaStorageSize))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: fullyPopulatedKafkaVersionUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
			},
		},
		{
			name: "should succeed when upgrading ibp version to lower than kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: ibpUpgrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.DesiredKafkaIbpVersion).To(Equal(ibpUpgrade.KafkaIbpVersion))
			},
		},
		{
			name: "should succeed when downgrading to lower patch kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: kafkaVersionPatchDowngrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.DesiredKafkaVersion).To(Equal(kafkaVersionPatchDowngrade.KafkaVersion))
			},
		},
		{
			name: "should succeed when upgrading to higher minor kafka version when in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: kafkaVersionPatchUpgradeInStatus,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.DesiredKafkaVersion).To(Equal(kafkaVersionPatchUpgradeInStatus.KafkaVersion))
			},
		},
		{
			name: "should succeed when upgrading simultaneously kafka, ibp and strimzi version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: allVersionsUpgrade,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.DesiredKafkaVersion).To(Equal(allVersionsUpgrade.KafkaVersion))
				Expect(result.DesiredKafkaIbpVersion).To(Equal(allVersionsUpgrade.KafkaIbpVersion))
				Expect(result.DesiredStrimziVersion).To(Equal(allVersionsUpgrade.StrimziVersion))
			},
		},
		{
			name: "should succeed when upgrading all possible values",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: allFieldsUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.DesiredKafkaVersion).To(Equal(allFieldsUpdateRequest.KafkaVersion))
				Expect(result.DesiredKafkaIbpVersion).To(Equal(allFieldsUpdateRequest.KafkaIbpVersion))
				Expect(result.DesiredStrimziVersion).To(Equal(allFieldsUpdateRequest.StrimziVersion))
				Expect(result.KafkaStorageSize).To(Equal(allFieldsUpdateRequest.KafkaStorageSize))
			},
		},
		{
			name: "should succeed when upgrading to bigger storage in different format",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: biggerStorageDifferentFormatUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(result.Id).To(Equal(sampleKafkaID1))
				Expect(result.KafkaStorageSize).To(Equal(biggerStorageDifferentFormatUpdateRequest.KafkaStorageSize))
			},
		},
		{
			name: "should fail when attempting to upgrade to smaller storage size",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: smallerStorageUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when attempting to upgrade to smaller storage size in different format",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: smallerStorageDifferentFormatUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when attempting to upgrade to smaller storage size in wrong format",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: wrongFormatStorageUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when attempting to upgrade to smaller storage size when providing random string as new storage size",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: randomStringStorageUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when attempting to upgrade storage with current storage size not set",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID4,
				kafkaUpdateRequest: biggerStorageUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when attempting to upgrade storage with current storage size set to some incorrect value",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID3,
				kafkaUpdateRequest: biggerStorageUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
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

	err2 := cluster.SetAvailableStrimziVersions(getTestStrimziVersionsMatrix())

	if err2 != nil {
		t.Error("failed to set available strimzi versions")
		return
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create kafkas that will be upgraded in the tests
	kafka1 := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID1,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-kafka1",
		OrganisationId:         "13640203",
		Status:                 constants.KafkaRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualKafkaVersion:     "2.8.1",
		DesiredKafkaVersion:    "2.8.1",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualKafkaIBPVersion:  "2.7.0",
		DesiredKafkaIBPVersion: "2.7.0",
		KafkaStorageSize:       "60Gi",
	}

	kafka2 := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID2,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-kafka2",
		OrganisationId:         "13640203",
		Status:                 constants.KafkaRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualKafkaVersion:     "2.8.0",
		DesiredKafkaVersion:    "2.8.0",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualKafkaIBPVersion:  "2.7.0",
		DesiredKafkaIBPVersion: "2.7.0",
		KafkaIBPUpgrading:      true,
		KafkaUpgrading:         true,
		StrimziUpgrading:       true,
	}

	kafka3 := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID3,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-kafka3",
		OrganisationId:         "13640203",
		Status:                 constants.KafkaRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualKafkaVersion:     "2.8.2",
		DesiredKafkaVersion:    "2.8.2",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.24.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
		ActualKafkaIBPVersion:  "2.8.1",
		DesiredKafkaIBPVersion: "2.8.1",
		KafkaIBPUpgrading:      true,
		KafkaStorageSize:       "random",
	}

	kafka4 := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: sampleKafkaID4,
		},
		MultiAZ:                false,
		Owner:                  "test-user",
		Region:                 "test",
		CloudProvider:          "test",
		Name:                   "test-kafk4",
		OrganisationId:         "13640203",
		Status:                 constants.KafkaRequestStatusReady.String(),
		ClusterID:              cluster.ClusterID,
		ActualKafkaVersion:     "2.6.0",
		DesiredKafkaVersion:    "2.6.0",
		ActualStrimziVersion:   "strimzi-cluster-operator.v0.20.0-0",
		DesiredStrimziVersion:  "strimzi-cluster-operator.v0.20.0-0",
		ActualKafkaIBPVersion:  "1.8.0",
		DesiredKafkaIBPVersion: "1.8.0",
		KafkaIBPUpgrading:      true,
		KafkaUpgrading:         true,
		StrimziUpgrading:       true,
	}

	if err := db.Create(kafka1).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	if err := db.Create(kafka2).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	if err := db.Create(kafka3).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	if err := db.Create(kafka4).Error; err != nil {
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

func getTestStrimziVersionsMatrix() []api.StrimziVersion {
	return []api.StrimziVersion{
		{
			Version: "strimzi-cluster-operator.v0.20.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "1.8.0"},
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "1.8.0"},
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v0.22.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v0.23.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v0.24.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
				{Version: "2.8.2"},
				{Version: "2.9.0"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
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
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
				{Version: "2.8.2"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v0.25.1-0",
			Ready:   false,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
				{Version: "2.8.2"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.6.0"},
				{Version: "2.7.0"},
				{Version: "2.8.0"},
				{Version: "2.8.1"},
			},
		},
		{
			Version: "strimzi-cluster-operator.v0.26.0-0",
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{Version: "2.8.2"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.8.2"},
			},
		},
	}
}
