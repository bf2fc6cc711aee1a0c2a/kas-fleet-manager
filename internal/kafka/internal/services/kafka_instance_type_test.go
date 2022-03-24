package services

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

var supportedKafkaSizeStandard = []api.SupportedKafkaSize{
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

var supportedKafkaSizeEval = []api.SupportedKafkaSize{
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
		want    []api.SupportedKafkaInstanceType
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
			want: []api.SupportedKafkaInstanceType{
				{
					Id:                  "standard",
					SupportedKafkaSizes: supportedKafkaSizeStandard,
				},
				{
					Id:                  "eval",
					SupportedKafkaSizes: supportedKafkaSizeEval,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := supportedKafkaInstanceTypesService{
				providerConfig: tt.fields.providerConfig,
				kafkaConfig:    tt.fields.kafkaConfig,
			}
			got, err := k.GetSupportedKafkaInstanceTypesByRegion(tt.args.cloudProvider, tt.args.cloudRegion)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("supported Kafka instance type lists dont match. want: %v got: %v", tt.want, got)
			}
		})
	}
}
