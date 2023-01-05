package promotion

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/onsi/gomega"
	"testing"
)

func TestNewPromotionKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService *services.KafkaServiceMock
		quotaService *services.QuotaServiceMock
		kafkaConfig  config.KafkaConfig
	}
	type KafkaService_ListKafkasToBePromotedExpect struct {
		calls int
	}
	type KafkaService_UpdateExpect struct {
		promotionStatus dbapi.KafkaPromotionStatus
		calls           int
	}
	type QuotaService_ReserveQuotaIfNotAlreadyReservedExpect struct {
		calls int
	}
	type QuotaService_DeleteQuotaForBillingModelExpect struct {
		calls int
	}
	type expect struct {
		kafkaService_ListKafkasToBePromoted           KafkaService_ListKafkasToBePromotedExpect
		kafkaService_Updates                          KafkaService_UpdateExpect
		kafkaService_Update                           KafkaService_UpdateExpect
		quotaService_ReserveQuotaIfNotAlreadyReserved QuotaService_ReserveQuotaIfNotAlreadyReservedExpect
		quotaService_DeleteQuotaForBillingModel       QuotaService_DeleteQuotaForBillingModelExpect
		wantDesiredBillingModel                       string
		wantActualBillingModel                        string
	}
	tests := []struct {
		name         string
		fields       fields
		wantErrCount int
		expect       expect
	}{
		{
			name:         "Test nothing to promote",
			wantErrCount: 0,
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasToBePromotedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, nil
					},
				},
				quotaService: &services.QuotaServiceMock{},
			},
			expect: expect{
				kafkaService_ListKafkasToBePromoted: KafkaService_ListKafkasToBePromotedExpect{
					calls: 1,
				},
			},
		},
		{
			name:         "Test clean promotion",
			wantErrCount: 0,
			expect: expect{
				wantDesiredBillingModel: "standard",
				wantActualBillingModel:  "standard",
				kafkaService_ListKafkasToBePromoted: KafkaService_ListKafkasToBePromotedExpect{
					calls: 1,
				},
				kafkaService_Updates: KafkaService_UpdateExpect{
					promotionStatus: "",
					calls:           1,
				},
				quotaService_DeleteQuotaForBillingModel: QuotaService_DeleteQuotaForBillingModelExpect{
					calls: 1,
				},
				quotaService_ReserveQuotaIfNotAlreadyReserved: QuotaService_ReserveQuotaIfNotAlreadyReservedExpect{
					calls: 1,
				},
			},
			fields: fields{
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                         api.AMSQuotaType.String(),
						AllowDeveloperInstance:       true,
						MaxAllowedDeveloperInstances: 10,
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "standard",
									DisplayName: "standard",
									Sizes:       nil,
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID:               "eval",
											AMSResource:      ocm.RHOSAKResourceName,
											AMSProduct:       string(ocm.RHOSAKEvalProduct),
											AMSBillingModels: []string{"standard"},
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					ListKafkasToBePromotedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							{
								Meta: api.Meta{
									ID: "123-kafka",
								},
								ActualKafkaBillingModel:  "eval",
								DesiredKafkaBillingModel: "standard",
								InstanceType:             types.STANDARD.String(),
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaIfNotAlreadyReservedFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "subscription-id", nil
					},
					DeleteQuotaForBillingModelFunc: func(subscriptionId string, kafkaBillingModel config.KafkaBillingModel) *errors.ServiceError {
						return nil
					},
				},
			},
		},
		{
			name:         "Test fail deleting quota",
			wantErrCount: 1,
			expect: expect{
				wantDesiredBillingModel: "standard",
				wantActualBillingModel:  "standard",
				kafkaService_ListKafkasToBePromoted: KafkaService_ListKafkasToBePromotedExpect{
					calls: 1,
				},
				kafkaService_Update: KafkaService_UpdateExpect{
					promotionStatus: dbapi.KafkaPromotionStatusPromoting,
					calls:           1,
				},
				quotaService_DeleteQuotaForBillingModel: QuotaService_DeleteQuotaForBillingModelExpect{
					calls: 1,
				},
				quotaService_ReserveQuotaIfNotAlreadyReserved: QuotaService_ReserveQuotaIfNotAlreadyReservedExpect{
					calls: 1,
				},
			},
			fields: fields{
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                         api.AMSQuotaType.String(),
						AllowDeveloperInstance:       true,
						MaxAllowedDeveloperInstances: 10,
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "standard",
									DisplayName: "standard",
									Sizes:       nil,
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID:               "eval",
											AMSResource:      ocm.RHOSAKResourceName,
											AMSProduct:       string(ocm.RHOSAKEvalProduct),
											AMSBillingModels: []string{"standard"},
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					ListKafkasToBePromotedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							{
								Meta: api.Meta{
									ID: "123-kafka",
								},
								ActualKafkaBillingModel:  "eval",
								DesiredKafkaBillingModel: "standard",
								InstanceType:             types.STANDARD.String(),
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaIfNotAlreadyReservedFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "subscription-id", nil
					},
					DeleteQuotaForBillingModelFunc: func(subscriptionId string, kafkaBillingModel config.KafkaBillingModel) *errors.ServiceError {
						return errors.NewServiceErrorBuilder().Recoverable().WithReason("Test delete failure").Build()
					},
				},
			},
		},
		{
			name:         "Test fail reserving quota",
			wantErrCount: 1,
			expect: expect{
				wantDesiredBillingModel: "standard",
				wantActualBillingModel:  "standard",
				kafkaService_ListKafkasToBePromoted: KafkaService_ListKafkasToBePromotedExpect{
					calls: 1,
				},
				kafkaService_Update: KafkaService_UpdateExpect{
					promotionStatus: dbapi.KafkaPromotionStatusFailed,
					calls:           1,
				},
				quotaService_ReserveQuotaIfNotAlreadyReserved: QuotaService_ReserveQuotaIfNotAlreadyReservedExpect{
					calls: 1,
				},
			},
			fields: fields{
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                         api.AMSQuotaType.String(),
						AllowDeveloperInstance:       true,
						MaxAllowedDeveloperInstances: 10,
					},
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "standard",
									DisplayName: "standard",
									Sizes:       nil,
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID:               "eval",
											AMSResource:      ocm.RHOSAKResourceName,
											AMSProduct:       string(ocm.RHOSAKEvalProduct),
											AMSBillingModels: []string{"standard"},
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					ListKafkasToBePromotedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							{
								Meta: api.Meta{
									ID: "123-kafka",
								},
								ActualKafkaBillingModel:  "eval",
								DesiredKafkaBillingModel: "standard",
								InstanceType:             types.STANDARD.String(),
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaIfNotAlreadyReservedFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "", errors.NewServiceErrorBuilder().WithReason("Test reserve failure").Build()
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := NewPromotionKafkaManager(
				workers.Reconciler{},
				tt.fields.kafkaService,
				&tt.fields.kafkaConfig,
				&services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			)

			errs := k.Reconcile()
			g.Expect(errs).To(gomega.HaveLen(tt.wantErrCount))

			g.Expect(tt.fields.kafkaService.UpdatesCalls()).To(gomega.HaveLen(tt.expect.kafkaService_Updates.calls))
			if tt.expect.kafkaService_Updates.calls > 0 {
				values := tt.fields.kafkaService.UpdatesCalls()[0].Values
				g.Expect(values["subscription_id"]).To(gomega.Equal("subscription-id"))
				g.Expect(values["promotion_status"]).To(gomega.Equal(tt.expect.kafkaService_Updates.promotionStatus.String()))
				g.Expect(values["actual_kafka_billing_model"]).To(gomega.Equal(tt.expect.wantActualBillingModel))
			}
			g.Expect(tt.fields.kafkaService.UpdateCalls()).To(gomega.HaveLen(tt.expect.kafkaService_Update.calls))
			if tt.expect.kafkaService_Update.calls > 0 {
				g.Expect(tt.fields.kafkaService.UpdateCalls()[0].KafkaRequest.PromotionStatus).To(gomega.Equal(tt.expect.kafkaService_Update.promotionStatus))
			}

			g.Expect(tt.fields.kafkaService.ListKafkasToBePromotedCalls()).To(gomega.HaveLen(tt.expect.kafkaService_ListKafkasToBePromoted.calls))
			g.Expect(tt.fields.quotaService.ReserveQuotaIfNotAlreadyReservedCalls()).To(gomega.HaveLen(tt.expect.quotaService_ReserveQuotaIfNotAlreadyReserved.calls))
			g.Expect(tt.fields.quotaService.DeleteQuotaForBillingModelCalls()).To(gomega.HaveLen(tt.expect.quotaService_DeleteQuotaForBillingModel.calls))
			//}
		})
	}
}
