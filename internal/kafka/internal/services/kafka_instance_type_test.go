package services

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/onsi/gomega"
)

func Test_KafkaInstanceTypes_GetSupportedKafkaInstanceTypesByRegion(t *testing.T) {
	type fields struct {
		providerConfig *config.ProviderConfig
		kafkaConfig    *config.KafkaConfig
	}

	type args struct {
		cloudProvider string
		cloudRegion   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []config.KafkaInstanceType
	}{
		{
			name: "success when instance type list",
			fields: fields{
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig:    &defaultKafkaConf,
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-1",
			},
			wantErr: false,
			want: []config.KafkaInstanceType{
				{
					Id:                     "developer",
					DisplayName:            "Trial",
					SupportedBillingModels: testSupportedKafkaBillingModelsDeveloper,
					Sizes:                  supportedKafkaSizeDeveloper,
				},
				{
					Id:                     "standard",
					DisplayName:            "Standard",
					SupportedBillingModels: testSupportedKafkaBillingModelsStandard,
					Sizes:                  supportedKafkaSizeStandard,
				},
			},
		},
		{
			name: "fail when cloud region not supported",
			fields: fields{
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig:    &defaultKafkaConf,
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-2",
			},
			wantErr: true,
		},
		{
			name: "fail when instance type not supported",
			fields: fields{
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "unsupported",
								},
							},
						},
					},
				},
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-1",
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := supportedKafkaInstanceTypesService{
				providerConfig: tt.fields.providerConfig,
				kafkaConfig:    tt.fields.kafkaConfig,
			}
			got, err := k.GetSupportedKafkaInstanceTypesByRegion(tt.args.cloudProvider, tt.args.cloudRegion)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
