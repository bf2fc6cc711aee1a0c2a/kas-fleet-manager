package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestKafkaSupportedSizesConfig_Validate(t *testing.T) {
	tests := []struct {
		name              string
		configFactoryFunc func() SupportedKafkaInstanceTypesConfig
		wantErr           bool
	}{
		{
			name: "Should not return an error with valid configuration",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex2 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex2.Id = "x2"
				testKafkaInstanceSizex2.DisplayName = "2"
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
								testKafkaInstanceSizex2,
							},
						},
					},
				}
				return res
			},
			wantErr: false,
		},
		{
			name: "Should fail because size was repeated",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should fail because property TotalMaxConnections was not specified",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.TotalMaxConnections = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should fail because property MaxPartitions was not specified",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxPartitions = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should fail because property MaxConnectionAttemptsPerSec was not specified",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxConnectionAttemptsPerSec = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property IngressThroughputPerSec is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.IngressThroughputPerSec = Quantity("")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property EgressThroughputPerSec is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.EgressThroughputPerSec = Quantity("")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionSize is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxDataRetentionSize = Quantity("")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxDataRetentionPeriod = ""
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property Id is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.Id = ""
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when a property with quantity format is invalid",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.IngressThroughputPerSec = "30Midk"
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is Zero",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxDataRetentionSize = Quantity("0Gi")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property with quantity format is less than Zero",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.EgressThroughputPerSec = Quantity("-30Mi")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is invalid",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxDataRetentionPeriod = "P14Dygyuook"
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxDataRetentionPeriod is zero",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxDataRetentionPeriod = "P0S"
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property KafkaProfile.id is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when KafkaProfile.Sizes is empty",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes:       []KafkaInstanceSize{},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should fail because profile was repeated",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when quota consumed is less than 1",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.QuotaConsumed = -1
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when quota consumed is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.QuotaConsumed = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when quota type is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.QuotaType = ""
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when capacity consumed is undefined",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.CapacityConsumed = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when capacity consumed is less than 1",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.CapacityConsumed = -1
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error when profile id is invalid",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "invalid",
							DisplayName: "Invalid",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxMessageSize is not set",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxMessageSize = Quantity("")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxMessageSize is invalid",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxMessageSize = Quantity("30Minonvalid")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MaxMessageSize is negative",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaxMessageSize = Quantity("-30Mi")
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MinInSyncReplicas is not set",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MinInSyncReplicas = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property MinInSyncReplicas is less than zero",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MinInSyncReplicas = -1
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property ReplicationFactor is not set",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.ReplicationFactor = 0
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property ReplicationFactor is less than zero",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.ReplicationFactor = -1
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property supportedAZModes is not set",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.SupportedAZModes = nil
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property supportedAZModes is empty",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.SupportedAZModes = []string{}
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property supportedAZModes contains an invalid value",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.SupportedAZModes = []string{"multi", "nonvalidvalue"}
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should succeed with multiple valid supportedAZModes values in a kafka instance size",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.SupportedAZModes = []string{"multi", "single"}
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: false,
		},
		{
			name: "Should return error when property DisplayName in a kafka instance size is not set",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.DisplayName = ""
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property LifespanSeconds in a kafka instance is set with an invalid value",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.LifespanSeconds = &[]int{-1}[0]
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return error when property LifespanSeconds in a kafka instance is set to 0",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.LifespanSeconds = &[]int{0}[0]
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error if maturity status is invalid",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaturityStatus = "invalid"
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
		{
			name: "Should return an error if maturity status is empty",
			configFactoryFunc: func() SupportedKafkaInstanceTypesConfig {
				testKafkaInstanceSizex1 := buildTestStandardKafkaInstanceSize()
				testKafkaInstanceSizex1.MaturityStatus = ""
				res := SupportedKafkaInstanceTypesConfig{
					SupportedKafkaInstanceTypes: []KafkaInstanceType{
						{
							Id:          "standard",
							DisplayName: "Standard",
							Sizes: []KafkaInstanceSize{
								testKafkaInstanceSizex1,
							},
						},
					},
				}
				return res
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			supportedKafkaInstanceTypesConfig := tt.configFactoryFunc()
			if err := supportedKafkaInstanceTypesConfig.validate(); (err != nil) != tt.wantErr {
				t.Errorf("SupportedKafkaSizesConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestKafkaInstanceType_HasAnInstanceSizeWithLifespan(t *testing.T) {
	tests := []struct {
		name              string
		kafkaInstanceType KafkaInstanceType
		want              bool
	}{
		{
			name: "returns true when kafka instance type has at least one size with lifespanSeconds set",
			kafkaInstanceType: KafkaInstanceType{
				Id: "myinstancetype",
				Sizes: []KafkaInstanceSize{
					KafkaInstanceSize{Id: "instancesize1"},
					KafkaInstanceSize{Id: "instancesize3", LifespanSeconds: &[]int{33513}[0]},
				},
			},
			want: true,
		},
		{
			name: "returns false when kafka instance type has no size with lifespanSeconds set",
			kafkaInstanceType: KafkaInstanceType{
				Id: "myinstancetype",
				Sizes: []KafkaInstanceSize{
					KafkaInstanceSize{Id: "instancesize1"},
					KafkaInstanceSize{Id: "instancesize3", LifespanSeconds: nil},
				},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.kafkaInstanceType.HasAnInstanceSizeWithLifespan()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}

}

func buildTestStandardKafkaInstanceSize() KafkaInstanceSize {
	return KafkaInstanceSize{
		Id:                          "x1",
		DisplayName:                 "1",
		IngressThroughputPerSec:     "30Mi",
		EgressThroughputPerSec:      "30Mi",
		TotalMaxConnections:         1000,
		MaxDataRetentionSize:        "100Gi",
		MaxPartitions:               1000,
		MaxDataRetentionPeriod:      "P14D",
		MaxConnectionAttemptsPerSec: 100,
		MaxMessageSize:              "1Mi",
		MinInSyncReplicas:           2,
		ReplicationFactor:           3,
		SupportedAZModes: []string{
			"multi",
		},
		QuotaConsumed:    1,
		QuotaType:        "rhosak",
		CapacityConsumed: 1,
		MaturityStatus:   MaturityStatusStable,
	}
}
