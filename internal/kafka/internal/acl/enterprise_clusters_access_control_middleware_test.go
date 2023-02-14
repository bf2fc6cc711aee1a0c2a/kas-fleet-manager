package acl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

func Test_EnterpriseClustersAccessControlMiddleware(t *testing.T) {
	type args struct {
		kafkaConfig         *config.KafkaConfig
		quotaServiceFactory services.QuotaServiceFactory
	}
	tests := []struct {
		name    string
		args    args
		token   *jwt.Token
		next    http.Handler
		request *http.Request
		want    int
	}{
		{
			name: "should return an error when standard instance type is missing from kafka configuration",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id": "13640203",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "some-id",
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusInternalServerError,
		},
		{
			name: "should return an error when standard instance type does not support enterprise billing model",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id": "13640203",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "some-billing-model",
										},
									},
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusInternalServerError,
		},
		{
			name: "should return an error when quota service factory returns an error",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id":   "13640203",
					"username": "some-user",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: constants.BillingModelEnterprise.String(),
										},
									},
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return &services.QuotaServiceMock{}, &errors.ServiceError{}
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusInternalServerError,
		},
		{
			name: "should return an error when enterprise quota check fails",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id":   "13640203",
					"username": "some-user",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: constants.BillingModelEnterprise.String(),
										},
									},
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return &services.QuotaServiceMock{
							CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username, externalID string, instanceTypeID types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
								return false, &errors.ServiceError{}
							},
						}, nil
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusInternalServerError,
		},
		{
			name: "should return a forbidden error when organization doesn't have enterprise quota",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id":   "13640203",
					"username": "some-user",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: constants.BillingModelEnterprise.String(),
										},
									},
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return &services.QuotaServiceMock{
							CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username, externalID string, instanceTypeID types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
								return false, nil
							},
						}, nil
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusForbidden,
		},
		{
			name: "should return a http status okay when organization has enterprise quota",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id":   "13640203",
					"username": "some-user",
				},
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type: "some-quota-type",
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: constants.BillingModelEnterprise.String(),
										},
									},
								},
							},
						},
					},
				},
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return &services.QuotaServiceMock{
							CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username, externalID string, instanceTypeID types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
								return true, nil
							},
						}, nil
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			h := NewEnterpriseClustersAccessControlMiddleware(tt.args.kafkaConfig, tt.args.quotaServiceFactory)
			toTest := setContextToken(h.Authorize(tt.next), tt.token)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, tt.request)
			resp := recorder.Result()
			resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, resp.StatusCode)
			}
		})
	}
}

func setContextToken(next http.Handler, token *jwt.Token) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		ctx = authentication.ContextWithToken(ctx, token)
		request = request.WithContext(ctx)
		next.ServeHTTP(writer, request)
	})
}
