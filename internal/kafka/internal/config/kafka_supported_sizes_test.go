package config

import (
	"testing"
)

func TestKafkaSupportedSizesConfig_Validate(t *testing.T) {

	type fields struct {
		SupportedKafkaSizesConfig SupportedKafkaSizesConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should not return an error with valid configuration",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
						{
							Size:                        "standard.x2",
							IngressThroughputPerSec:     "60Mi",
							EgressThroughputPerSec:      "60Mi",
							TotalMaxConnections:         2000,
							MaxDataRetentionSize:        "200Gi",
							MaxPartitions:               2000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 200,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should fail because size was repeated",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail because property TotalMaxConnections was not specified",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property IngressThroughputPerSec is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is invalid",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Midk",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is Zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "0Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is less than Zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "-30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14D",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is invalid",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P14Dygyuook",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaSizes: []KafkaInstanceSize{
						{
							Size:                        "standard.x1",
							IngressThroughputPerSec:     "30Mi",
							EgressThroughputPerSec:      "30Mi",
							TotalMaxConnections:         1000,
							MaxDataRetentionSize:        "100Gi",
							MaxPartitions:               1000,
							MaxDataRetentionPeriod:      "P0S",
							MaxConnectionAttemptsPerSec: 100,
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &KafkaSupportedSizesConfig{
				SupportedKafkaSizesConfig: tt.fields.SupportedKafkaSizesConfig,
			}
			if err := s.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("KafkaInstanceSizesConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
