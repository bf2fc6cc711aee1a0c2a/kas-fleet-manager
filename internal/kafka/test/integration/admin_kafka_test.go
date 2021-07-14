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
