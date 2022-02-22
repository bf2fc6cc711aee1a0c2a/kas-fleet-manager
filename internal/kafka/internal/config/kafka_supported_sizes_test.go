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
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									EgressThroughputPerSec:      "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
									Cost:                        1,
								},
								{
									Id:                          "x2",
									IngressThroughputPerSec:     "60Mi",
									EgressThroughputPerSec:      "60Mi",
									TotalMaxConnections:         2000,
									MaxDataRetentionSize:        "200Gi",
									MaxPartitions:               2000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 200,
									Cost:                        2,
								},
							},
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
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									EgressThroughputPerSec:      "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
								},
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail because property TotalMaxConnections was not specified",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail because property MaxPartitions was not specified",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									EgressThroughputPerSec:      "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail because property MaxConnectionAttemptsPerSec was not specified",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                      "x1",
									IngressThroughputPerSec: "30Mi",
									EgressThroughputPerSec:  "30Mi",
									TotalMaxConnections:     1000,
									MaxDataRetentionSize:    "100Gi",
									MaxDataRetentionPeriod:  "P14D",
									MaxPartitions:           1000,
								},
							},
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
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property EgressThroughputPerSec is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionSize is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									EgressThroughputPerSec:      "30Mi",
									IngressThroughputPerSec:     "30Mi",
									TotalMaxConnections:         1000,
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									EgressThroughputPerSec:      "30Mi",
									IngressThroughputPerSec:     "30Mi",
									TotalMaxConnections:         1000,
									MaxPartitions:               1000,
									MaxConnectionAttemptsPerSec: 100,
									MaxDataRetentionSize:        "100Gi",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property Id is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									EgressThroughputPerSec:      "30Mi",
									IngressThroughputPerSec:     "30Mi",
									TotalMaxConnections:         1000,
									MaxPartitions:               1000,
									MaxConnectionAttemptsPerSec: 100,
									MaxDataRetentionSize:        "100Gi",
									MaxDataRetentionPeriod:      "P14D",
								},
							},
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
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is Zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is less than Zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is invalid",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is zero",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when property KafkaProfile.id is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return error when KafkaProfile.Sizes is empty",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id:    "standard",
							Sizes: []KafkaInstanceSize{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail because profile was repeted",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return an error when cost is less than 1",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									EgressThroughputPerSec:      "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
									Cost:                        -1,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should return an error when cost is undefined",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "standard",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
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
				},
			},
			wantErr: true,
		},
		{
			name: "Should return an error when profile id is invalid",
			fields: fields{
				SupportedKafkaSizesConfig: SupportedKafkaSizesConfig{
					SupportedKafkaProfiles: []KafkaProfile{
						{
							Id: "invalid",
							Sizes: []KafkaInstanceSize{
								{
									Id:                          "x1",
									IngressThroughputPerSec:     "30Mi",
									EgressThroughputPerSec:      "30Mi",
									TotalMaxConnections:         1000,
									MaxDataRetentionSize:        "100Gi",
									MaxPartitions:               1000,
									MaxDataRetentionPeriod:      "P14D",
									MaxConnectionAttemptsPerSec: 100,
									Cost:                        1,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.SupportedKafkaSizesConfig.validate(); (err != nil) != tt.wantErr {
				t.Errorf("SupportedKafkaSizesConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
