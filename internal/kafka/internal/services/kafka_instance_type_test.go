package services

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/onsi/gomega"
)

var supportedKafkaSizeStandard = []config.KafkaInstanceSize{
	{
		Id:                          "x1",
		IngressThroughputPerSec:     "30Mi",
		EgressThroughputPerSec:      "30Mi",
		TotalMaxConnections:         1000,
		MaxDataRetentionSize:        "100Gi",
		MaxPartitions:               1000,
		MaxDataRetentionPeriod:      "P14D",
		MaxConnectionAttemptsPerSec: 100,
		QuotaConsumed:               1,
		QuotaType:                   "rhosak",
		CapacityConsumed:            1,
	},
}

var supportedKafkaSizeDeveloper = []config.KafkaInstanceSize{
	{
		Id:                          "x2",
		IngressThroughputPerSec:     "60Mi",
		EgressThroughputPerSec:      "60Mi",
		TotalMaxConnections:         2000,
		MaxDataRetentionSize:        "200Gi",
		MaxPartitions:               2000,
		MaxDataRetentionPeriod:      "P14D",
		MaxConnectionAttemptsPerSec: 200,
		QuotaConsumed:               2,
		QuotaType:                   "rhosak",
		CapacityConsumed:            2,
	},
}

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
					Id:          "developer",
					DisplayName: "Trial",
					Sizes:       supportedKafkaSizeDeveloper,
				},
				{
					Id:          "standard",
					DisplayName: "Standard",
					Sizes:       supportedKafkaSizeStandard,
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := supportedKafkaInstanceTypesService{
				providerConfig: tt.fields.providerConfig,
				kafkaConfig:    tt.fields.kafkaConfig,
			}
			got, err := k.GetSupportedKafkaInstanceTypesByRegion(tt.args.cloudProvider, tt.args.cloudRegion)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
