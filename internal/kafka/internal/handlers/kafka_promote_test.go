package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	kafkaConstants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

func Test_kafkaPromoteHandler_Promote(t *testing.T) {
	type fields struct {
		kafkaService                 services.KafkaService
		kafkaConfig                  *config.KafkaConfig
		kafkaPromoteValidatorFactory KafkaPromoteValidatorFactory
	}
	type args struct {
		kafkaID             string
		promoteURL          string
		kafkaPromoteRequest public.KafkaPromoteRequest
	}

	promoteURL := "/api/kafkas_mgmt/v1/kafkas/{id}/promote?async=true"

	testKafkaID := "test-kafka-id"
	testKafkaConfig := &config.KafkaConfig{
		Quota: &config.KafkaQuotaConfig{
			Type: string(api.AMSQuotaType),
		},
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					config.KafkaInstanceType{
						Id: types.STANDARD.String(),
						SupportedBillingModels: []config.KafkaBillingModel{
							config.KafkaBillingModel{
								ID:               "standard",
								AMSBillingModels: []string{"standard"},
							},
							config.KafkaBillingModel{
								ID:               "marketplace",
								AMSBillingModels: []string{"marketplace-aws", "marketplace-rhm"},
							},
							config.KafkaBillingModel{
								ID:               "eval",
								AMSBillingModels: []string{"standard"},
							},
						},
					},
				},
			},
		},
	}

	testKafkaConfigQuotMgmtList := &config.KafkaConfig{
		Quota: &config.KafkaQuotaConfig{
			Type: string(api.QuotaManagementListQuotaType),
		},
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					config.KafkaInstanceType{
						Id: types.STANDARD.String(),
						SupportedBillingModels: []config.KafkaBillingModel{
							config.KafkaBillingModel{
								ID: "standard",
							},
							config.KafkaBillingModel{
								ID: "marketplace",
							},
							config.KafkaBillingModel{
								ID: "eval",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "promote returns 202 Accepted when an eval instance is promoted to standard",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusAccepted,
		},
		{
			name: "promotion is allowed returning 202 Accepted when a previously performed promotion failed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
							PromotionStatus:          dbapi.KafkaPromotionStatusFailed,
							PromotionDetails:         "promotion failed example detail",
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusAccepted,
		},

		{
			name: "promote returns 400 Bad Request if the method is not called with the 'async' query parameter set to true",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    "unexistingkafkaID",
				promoteURL: "/api/kafkas_mgmt/v1/kafkas/{id}/promote",
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel: "standard",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 404 Not found if the provided kafka instance does not exist",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.NotFound("instance id %s not found", id)
					},
				},
			},
			args: args{
				kafkaID:    "unexistingkafkaID",
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "promote returns 400 Bad Request when desired kafka billing model is not provided",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 403 Forbidden when the user that requested the promotion is not the kafka owner",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
							Owner:                    "adifferentowner",
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "promote returns 500 Internal Server Error when a non-eval instance is promoted",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "marketplace",
							ActualKafkaBillingModel:  "marketplace",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "promote returns 500 Internal Server Error when the kafka request to be promoted is not in one of the promotable statuses",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   string(kafkaConstants.KafkaRequestStatusAccepted),
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "promote returns 400 Bad Request when the kafka request to promote already has a kafka billing model than the desired kafka billing model to promote",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
				},
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "eval",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 500 Internal Server Error when a promotion is already in progress",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "standard",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
							PromotionStatus:          dbapi.KafkaPromotionStatusPromoting,
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "standard",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "promote returns 500 Internal Server Error when the desired kafka billing model to promote is not marketplace nor standard",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "standard",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "anotherbillingmodel",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 500 Internal Server Error when the billing account fails to validate",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return errors.GeneralError("test error validating billing account")
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "marketplace",
					DesiredMarketplace:           "aws",
					DesiredBillingCloudAccountId: "123456",
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "promote returns 500 Internal Server Error when promoting to marketplace and not providing the desired marketplace",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "marketplace",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "123456",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 500 Internal Server Error when promoting to marketplace and not providing the desired cloud account id",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfig,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfig),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "marketplace",
					DesiredMarketplace:           "aws",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "promote returns 202 Accepted when an eval instance is promoted to marketplace and no marketplace nor cloud account id are provided when running in quota mgmt list mode",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						res := &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: id,
							},
							InstanceType:             types.STANDARD.String(),
							DesiredKafkaBillingModel: "eval",
							ActualKafkaBillingModel:  "eval",
							Status:                   kafkaConstants.KafkaRequestStatusReady.String(),
						}
						return res, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingModelID, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig:                  testKafkaConfigQuotMgmtList,
				kafkaPromoteValidatorFactory: NewDefaultKafkaPromoteValidatorFactory(testKafkaConfigQuotMgmtList),
			},
			args: args{
				kafkaID:    testKafkaID,
				promoteURL: promoteURL,
				kafkaPromoteRequest: public.KafkaPromoteRequest{
					DesiredKafkaBillingModel:     "marketplace",
					DesiredMarketplace:           "",
					DesiredBillingCloudAccountId: "",
				},
			},
			wantStatusCode: http.StatusAccepted,
		},
	}
	_ = testKafkaConfigQuotMgmtList

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewKafkaPromoteHandler(tt.fields.kafkaService, tt.fields.kafkaConfig, tt.fields.kafkaPromoteValidatorFactory)
			promoteRequest := tt.args.kafkaPromoteRequest
			requestBody, err := json.Marshal(promoteRequest)
			if err != nil {
				panic(fmt.Errorf("unexpected test error: %v", err))
			}
			req, rw := GetHandlerParams(http.MethodPost, tt.args.promoteURL, bytes.NewBuffer(requestBody), t)
			req = mux.SetURLVars(req, map[string]string{kafkaPromoteRouteVariableID: tt.args.kafkaID})
			h.Promote(rw, req)
			resp := rw.Result()
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(fmt.Errorf("unexpected test error: %v", err))
			}
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode), "returned body: '%s'", responseBody)
		})
	}
}
