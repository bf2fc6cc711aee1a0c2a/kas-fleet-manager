package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	adminprivate "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	coreTest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
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
	g := gomega.NewWithT(t)
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID))
				g.Expect(result.DesiredStrimziVersion).To(gomega.Equal(desiredStrimziVersion))
				g.Expect(result.AccountNumber).ToNot(gomega.BeEmpty())
				g.Expect(result.Namespace).ToNot(gomega.BeEmpty())
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
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID))
				g.Expect(result.DesiredStrimziVersion).To(gomega.Equal(desiredStrimziVersion))
				g.Expect(result.ClusterId).ShouldNot(gomega.BeNil())
				g.Expect(result.Namespace).ToNot(gomega.BeEmpty())
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
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID))
				g.Expect(result.DesiredStrimziVersion).To(gomega.Equal(desiredStrimziVersion))
				g.Expect(result.ClusterId).ShouldNot(gomega.BeNil())
				g.Expect(result.Namespace).ToNot(gomega.BeEmpty())
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
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
	kafka := mockkafka.BuildKafkaRequest(
		mockkafka.WithPredefinedTestValues(),
		mockkafka.With(mockkafka.ID, sampleKafkaID),
		mockkafka.With(mockkafka.DESIRED_STRIMZI_VERSION, desiredStrimziVersion),
	)

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.GetKafkaById(ctx, tt.args.kafkaID)
			tt.verifyResponse(result, resp, err)
			if resp != nil {
				_ = resp.Body.Close()
			}
		})
	}
}

func TestAdminKafka_Delete(t *testing.T) {
	g := gomega.NewWithT(t)
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusAccepted))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			_, resp, err := client.DefaultApi.DeleteKafkaById(ctx, kafkaId, true)
			if resp != nil {
				resp.Body.Close()
			}
			tt.verifyResponse(resp, err)
		})
	}
}

func TestAdminKafka_List(t *testing.T) {
	g := gomega.NewWithT(t)
	type args struct {
		ctx           func(h *coreTest.Helper) context.Context
		kafkaListSize int
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, ExpectedSize int)
	}{
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				kafkaListSize: 0,
			},
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, ExpectedSize int) {
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
				g.Expect(len(kafkaList.Items)).To(gomega.Equal(ExpectedSize))
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
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, ExpectedSize int) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(len(kafkaList.Items)).To(gomega.Equal(ExpectedSize))
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
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, ExpectedSize int) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(len(kafkaList.Items)).To(gomega.Equal(ExpectedSize))
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
			verifyResponse: func(resp *http.Response, err error, kafkaList adminprivate.KafkaList, ExpectedSize int) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(len(kafkaList.Items)).To(gomega.Equal(ExpectedSize))
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
		*mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
		),
		*mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.ORGANISATION_ID, "12147054"),
		),
	}

	for i := range kafkas {
		if err := db.Create(&kafkas[i]).Error; err != nil {
			t.Errorf("failed to create Kafka db record due to error: %v", err)
		}
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			kafkaList, resp, err := client.DefaultApi.GetKafkas(ctx, nil)
			if resp != nil {
				resp.Body.Close()
			}
			tt.verifyResponse(resp, err, kafkaList, tt.args.kafkaListSize)
		})
	}
}

func TestAdminKafka_Update(t *testing.T) {
	g := gomega.NewWithT(t)
	sampleKafkaID1 := api.NewID()
	sampleKafkaID2 := api.NewID()
	sampleKafkaID3 := api.NewID()
	sampleKafkaID4 := api.NewID()

	allFieldsUpdateRequest := adminprivate.KafkaUpdateRequest{
		StrimziVersion:       "strimzi-cluster-operator.v0.25.0-0",
		KafkaVersion:         "2.8.3",
		KafkaIbpVersion:      "2.8.1",
		MaxDataRetentionSize: "70Gi",
	}

	initialStorageSize := "60Gi"
	biggerStorageSizeDifferentFormat := "75000Mi"
	smallerStorageSize := "50Gi"
	smallerStorageSizeDifferentFormat := "50000Mi"
	wrongFormatStorageSize := "80Gb"
	randomStringStorageSize := "M2h9O8wO7k"

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
		// General
		{
			name: "should fail when kafkaUpdateRequest is empty",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when kafkaUpdateRequest request params contain only strings with whitespaces",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion:             " ",
					KafkaVersion:               " ",
					KafkaIbpVersion:            " ",
					DeprecatedKafkaStorageSize: " ",
					MaxDataRetentionSize:       " ",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when the requested kafka does not exist",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminReadRole})
				},
				kafkaID:            "nonexistentkafkaID",
				kafkaUpdateRequest: allFieldsUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DesiredKafkaVersion).To(gomega.Equal(allFieldsUpdateRequest.KafkaVersion))
				g.Expect(result.DesiredKafkaIbpVersion).To(gomega.Equal(allFieldsUpdateRequest.KafkaIbpVersion))
				g.Expect(result.DesiredStrimziVersion).To(gomega.Equal(allFieldsUpdateRequest.StrimziVersion))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(allFieldsUpdateRequest.MaxDataRetentionSize))

				dataRetentionSizeQuantity := config.Quantity(allFieldsUpdateRequest.MaxDataRetentionSize)
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
			},
		},
		// Auth tests
		{
			name: "should fail authentication when there is no role defined in the request",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{})
				},
				kafkaID: sampleKafkaID1,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
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
				g.Expect(err).NotTo(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.KasFleetManagerAdminWriteRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminWriteRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: allFieldsUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
			},
		},
		{
			name: fmt.Sprintf("should succeed when the role defined in the request is %s", auth.KasFleetManagerAdminFullRole),
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID:            sampleKafkaID1,
				kafkaUpdateRequest: allFieldsUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
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
				kafkaUpdateRequest: allFieldsUpdateRequest,
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusNotFound))
			},
		},
		// Kafka ibp downgrade/upgrade tests
		{
			name: "should fail when downgrading ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// based on allFieldsUpdateRequest values
					KafkaIbpVersion: "2.6.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version to version higher than kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// based on allFieldsUpdateRequest values
					KafkaIbpVersion: "2.8.5",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading ibp version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					KafkaIbpVersion: "2.8.2",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should succeed when upgrading ibp version to lower than kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// based on allFieldsUpdateRequest values
					KafkaIbpVersion: "2.8.2",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DesiredKafkaIbpVersion).To(gomega.Equal("2.8.2"))
			},
		},
		// Kafka version downgrade/upgrade tests
		{
			name: "should fail when downgrading to lower minor kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// must be lower than allFieldsUpdateRequest.KafkaVersion
					KafkaVersion: "2.7.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading to higher minor kafka version when not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					KafkaVersion: "2.8.15",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower patch kafka version smaller than ibp version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID3,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					KafkaVersion: "2.8.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when downgrading to lower major kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID4,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					KafkaVersion: "1.8.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading kafka version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					KafkaVersion: "2.9.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should succeed when downgrading to lower patch kafka version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// based on allFieldsUpdateRequest values
					KafkaVersion: "2.8.2",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DesiredKafkaVersion).To(gomega.Equal("2.8.2"))
			},
		},
		{
			name: "should succeed when upgrading to higher minor kafka version when in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// based on allFieldsUpdateRequest values
					KafkaVersion: "2.9.0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DesiredKafkaVersion).To(gomega.Equal("2.9.0"))
			},
		},
		// Strimzi version downgrade/upgrade tests
		{
			name: "should fail when upgrading strimzi version when already upgrade in progress",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion: "strimzi-cluster-operator.v0.25.0-0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version not in the status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion: "strimzi-cluster-operator.v0.29.0-0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading simultaneously kafka, ibp and strimzi version when any of those not in status",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.26.0-0",
					KafkaIbpVersion: "2.8.4",
					KafkaVersion:    "2.8.4",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when upgrading strimzi version to a version that is not ready",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion: "strimzi-cluster-operator.v0.25.1-0",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		// all version upgrades
		{
			name: "should succeed when upgrading simultaneously kafka, ibp and strimzi version",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.26.0-0",
					KafkaIbpVersion: "2.9.0",
					KafkaVersion:    "2.9.1",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DesiredKafkaVersion).To(gomega.Equal("2.9.1"))
				g.Expect(result.DesiredKafkaIbpVersion).To(gomega.Equal("2.9.0"))
				g.Expect(result.DesiredStrimziVersion).To(gomega.Equal("strimzi-cluster-operator.v0.26.0-0"))
			},
		},
		// Storage update tests - using kafka_storage_size (to be removed once kafka_storage_size has been removed)
		{
			name: "should succeed when attempting to update to the same storage size using kafka_storage_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					// current storage size for this kafka was updated to this size by
					// 'should succeed when upgrading all possible values' test case
					DeprecatedKafkaStorageSize: allFieldsUpdateRequest.MaxDataRetentionSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(allFieldsUpdateRequest.MaxDataRetentionSize))

				dataRetentionSizeQuantity := config.Quantity(allFieldsUpdateRequest.MaxDataRetentionSize)
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
			},
		},
		{
			name: "should fail when attempting to update to smaller storage size using kafka_storage_size_field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: smallerStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update to smaller storage size in different format using kafka_storage_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: smallerStorageSizeDifferentFormat,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update to smaller storage size in the wrong format using kafka_storage_size_field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: wrongFormatStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should succeed when updating to bigger storage size using kafka_storage_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: biggerStorageSizeDifferentFormat,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(biggerStorageSizeDifferentFormat))

				dataRetentionSizeQuantity := config.Quantity(biggerStorageSizeDifferentFormat)
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
			},
		},
		{
			name: "should fail when attempting to update storage size to a random string using kafka_storage_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: randomStringStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		// Storage update tests - using max_data_retention_size
		{
			name: "should succeed when attempting to update to the same storage size using max_data_retention_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: initialStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(initialStorageSize))

				dataRetentionSizeQuantity := config.Quantity(initialStorageSize)
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
			},
		},
		{
			name: "should fail when attempting to upgrade to smaller storage size using max_data_retention_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: smallerStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update to smaller storage size in different format using kafka_storage_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: smallerStorageSizeDifferentFormat,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update storage using an invalid format using max_data_retention_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: wrongFormatStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should succeed when upgrading to bigger storage size using max_data_retention_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID2,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: biggerStorageSizeDifferentFormat,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID2))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal(biggerStorageSizeDifferentFormat))

				dataRetentionSizeQuantity := config.Quantity(biggerStorageSizeDifferentFormat)
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
			},
		},
		{
			name: "should fail when attempting to update storage size to a random string using max_data_retention_size field",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: randomStringStorageSize,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update storage when current storage size not set",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID4,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: "100Gi",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should fail when attempting to update storage when current storage size is set to some incorrect value",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID3,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					MaxDataRetentionSize: biggerStorageSizeDifferentFormat,
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).NotTo(gomega.BeNil())
			},
		},
		{
			name: "should use max_data_retention_size over kafka_storage_size if both are specified",
			args: args{
				ctx: func(h *coreTest.Helper) context.Context {
					return NewAuthenticatedContextForAdminEndpoints(h, []string{auth.KasFleetManagerAdminFullRole})
				},
				kafkaID: sampleKafkaID1,
				kafkaUpdateRequest: adminprivate.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: "1000Gi",
					MaxDataRetentionSize:       "100Gi",
				},
			},
			verifyResponse: func(result adminprivate.Kafka, resp *http.Response, err error) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
				g.Expect(result.Id).To(gomega.Equal(sampleKafkaID1))
				g.Expect(result.DeprecatedKafkaStorageSize).To(gomega.Equal("100Gi"))

				dataRetentionSizeQuantity := config.Quantity("100Gi")
				dataRetentionSizeBytes, convErr := dataRetentionSizeQuantity.ToInt64()
				g.Expect(convErr).ToNot(gomega.HaveOccurred())
				g.Expect(result.MaxDataRetentionSize.Bytes).To(gomega.Equal(dataRetentionSizeBytes))
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
		KafkaStorageSize:       initialStorageSize,
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
		KafkaStorageSize:       initialStorageSize,
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.args.ctx(h)
			client := test.NewAdminPrivateAPIClient(h)
			result, resp, err := client.DefaultApi.UpdateKafkaById(ctx, tt.args.kafkaID, tt.args.kafkaUpdateRequest)
			if resp != nil {
				resp.Body.Close()
			}
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
				{Version: "2.8.3"},
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
				{Version: "2.9.1"},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{Version: "2.8.2"},
				{Version: "2.9.0"},
			},
		},
	}
}
