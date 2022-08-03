package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestNodePrewarmingConfig_Validate(t *testing.T) {
	tests := []struct {
		name                 string
		kafkaConfig          *KafkaConfig
		nodePrewarmingConfig NodePrewarmingConfig
		wantErr              bool
	}{
		{
			name: "should return an error when the instance type does not exists in supported instance types",
			kafkaConfig: &KafkaConfig{
				SupportedInstanceTypes: &KafkaSupportedInstanceTypesConfig{
					Configuration: SupportedKafkaInstanceTypesConfig{
						[]KafkaInstanceType{
							{
								Id: "standard",
								Sizes: []KafkaInstanceSize{
									{
										Id: "x1",
									},
								},
							},
						},
					},
				},
			},
			nodePrewarmingConfig: NodePrewarmingConfig{
				Configuration: map[string]InstanceTypeNodePrewarmingConfig{
					"instance-type": {
						BaseStreamingUnitSize: "x1",
						NumReservedInstances:  9,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error when the base streaming unit size does not exists in supported instance types",
			kafkaConfig: &KafkaConfig{
				SupportedInstanceTypes: &KafkaSupportedInstanceTypesConfig{
					Configuration: SupportedKafkaInstanceTypesConfig{
						[]KafkaInstanceType{
							{
								Id: "instance-type",
								Sizes: []KafkaInstanceSize{
									{
										Id: "x1",
									},
								},
							},
						},
					},
				},
			},
			nodePrewarmingConfig: NodePrewarmingConfig{
				Configuration: map[string]InstanceTypeNodePrewarmingConfig{
					"instance-type": {
						BaseStreamingUnitSize: "x2",
						NumReservedInstances:  10,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error when the instance type and base streaming unit size exists and have a invalid reserved streaming unit configuration",
			kafkaConfig: &KafkaConfig{
				SupportedInstanceTypes: &KafkaSupportedInstanceTypesConfig{
					Configuration: SupportedKafkaInstanceTypesConfig{
						[]KafkaInstanceType{
							{
								Id: "instance-type",
								Sizes: []KafkaInstanceSize{
									{
										Id: "x2",
									},
								},
							},
						},
					},
				},
			},
			nodePrewarmingConfig: NodePrewarmingConfig{
				Configuration: map[string]InstanceTypeNodePrewarmingConfig{
					"instance-type": {
						BaseStreamingUnitSize: "x2",
						NumReservedInstances:  -10,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when the instance type and base streaming unit size exists and have a valid reserved streaming unit configuration",
			kafkaConfig: &KafkaConfig{
				SupportedInstanceTypes: &KafkaSupportedInstanceTypesConfig{
					Configuration: SupportedKafkaInstanceTypesConfig{
						[]KafkaInstanceType{
							{
								Id: "instance-type",
								Sizes: []KafkaInstanceSize{
									{
										Id: "x2",
									},
								},
							},
						},
					},
				},
			},
			nodePrewarmingConfig: NodePrewarmingConfig{
				Configuration: map[string]InstanceTypeNodePrewarmingConfig{
					"instance-type": {
						BaseStreamingUnitSize: "x2",
						NumReservedInstances:  10,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should not return an error when the instance type and default base streaming unit size exists and have a valid reserved streaming unit configuration",
			kafkaConfig: &KafkaConfig{
				SupportedInstanceTypes: &KafkaSupportedInstanceTypesConfig{
					Configuration: SupportedKafkaInstanceTypesConfig{
						[]KafkaInstanceType{
							{
								Id: "instance-type",
								Sizes: []KafkaInstanceSize{
									{
										Id: "x1",
									},
								},
							},
						},
					},
				},
			},
			nodePrewarmingConfig: NodePrewarmingConfig{
				Configuration: map[string]InstanceTypeNodePrewarmingConfig{
					"instance-type": {
						BaseStreamingUnitSize: "", // base default streaming unit size of x1 will be used instead
						NumReservedInstances:  10,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			err := tt.nodePrewarmingConfig.validate(tt.kafkaConfig)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
