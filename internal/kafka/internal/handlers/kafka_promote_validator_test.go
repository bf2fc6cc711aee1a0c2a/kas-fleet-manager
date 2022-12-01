package handlers

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/onsi/gomega"
)

func Test_amsPromoteValidator_Validate(t *testing.T) {
	type fields struct {
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaPromoteValidatorRequest kafkaPromoteValidatorRequest
	}

	testKafkaConfigWithStandardBillingModel := &config.KafkaConfig{
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
						},
					},
				},
			},
		},
	}

	testKafkaConfigWithMarketplaceBillingModel := &config.KafkaConfig{
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					config.KafkaInstanceType{
						Id: types.STANDARD.String(),
						SupportedBillingModels: []config.KafkaBillingModel{
							config.KafkaBillingModel{
								ID:               "marketplace",
								AMSBillingModels: []string{"marketplace-aws"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "validation succeeds when standard kafka billing model is provided and it is supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithStandardBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "standard",
					KafkaInstanceType:        types.STANDARD.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "validation succeeds when marketplace kafka billing model is provided with a supported marketplace",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "marketplace",
					DesiredKafkaMarketplace:    "aws",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          types.STANDARD.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "validation succeeds when standard kafka billing model is provided with unknown provided marketplace and cloud account (they are ignored)",
			fields: fields{
				kafkaConfig: testKafkaConfigWithStandardBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "standard",
					DesiredKafkaMarketplace:    "unknownmarketplace",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          types.STANDARD.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails when marketplace kafka billing model is provided but desired marketplace is not provided",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "marketplace",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          types.STANDARD.String(),
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails when marketplace kafka billing model is provided but desired cloud account id is not provided",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "marketplace",
					DesiredKafkaMarketplace:  "aws",
					KafkaInstanceType:        types.STANDARD.String(),
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails when the provided instance type is not supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "marketplace",
					DesiredKafkaMarketplace:    "aws",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          "unsupportedinstancetype",
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails when the provided kafka billing model for the provided instance type is not supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "unknownkafkabillingmodel",
					DesiredKafkaMarketplace:    "aws",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          types.STANDARD.String(),
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails when the provided marketplace is not supported as an AMS billing model",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel:   "marketplace",
					DesiredKafkaMarketplace:    "unknownmarketplace",
					DesiredKafkaCloudAccountID: "123456",
					KafkaInstanceType:          types.STANDARD.String(),
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			validator := amsKafkaPromoteValidator{
				KafkaConfig: tt.fields.kafkaConfig,
			}
			err := validator.Validate(tt.args.kafkaPromoteValidatorRequest)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "Returned error: %v", err)
		})
	}
}

func Test_quotaManagementListKafkaPromoteValidator_Validate(t *testing.T) {
	type fields struct {
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaPromoteValidatorRequest kafkaPromoteValidatorRequest
	}

	testKafkaConfigWithStandardBillingModel := &config.KafkaConfig{
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					config.KafkaInstanceType{
						Id: types.STANDARD.String(),
						SupportedBillingModels: []config.KafkaBillingModel{
							config.KafkaBillingModel{
								ID: "standard",
							},
						},
					},
				},
			},
		},
	}

	testKafkaConfigWithMarketplaceBillingModel := &config.KafkaConfig{
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					config.KafkaInstanceType{
						Id: types.STANDARD.String(),
						SupportedBillingModels: []config.KafkaBillingModel{
							config.KafkaBillingModel{
								ID:               "marketplace",
								AMSBillingModels: []string{"marketplace-aws"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "validation succeeds when standard kafka billing model is provided and it is supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithStandardBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "standard",
					KafkaInstanceType:        types.STANDARD.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "validation succeeds when marketplace kafka billing model is provided with unknown provided marketplace and cloud account (they are ignored)",
			fields: fields{
				kafkaConfig: testKafkaConfigWithMarketplaceBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "marketplace",
					KafkaInstanceType:        types.STANDARD.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails when the provided instance type is not supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithStandardBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "standard",
					KafkaInstanceType:        "unknowninstancetype",
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails when the provided kafka billing model is not supported",
			fields: fields{
				kafkaConfig: testKafkaConfigWithStandardBillingModel,
			},
			args: args{
				kafkaPromoteValidatorRequest: kafkaPromoteValidatorRequest{
					DesiredKafkaBillingModel: "unknownkbm",
					KafkaInstanceType:        types.STANDARD.String(),
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			validator := quotaManagementListKafkaPromoteValidator{
				KafkaConfig: tt.fields.kafkaConfig,
			}
			err := validator.Validate(tt.args.kafkaPromoteValidatorRequest)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "error content: '%v'", err)
		})
	}
}
