package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/onsi/gomega"
)

func TestDynamicScalingConfig_IsDataplaneScaleDownTriggerEnabled(t *testing.T) {
	t.Parallel()
	type fields struct {
		EnableDynamicScaleDownManagerScaleDownTrigger bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return false when scale down trigger is disabled",
			fields: fields{
				EnableDynamicScaleDownManagerScaleDownTrigger: false,
			},
			want: false,
		},
		{
			name: "return true when scale down trigger is enabled",
			fields: fields{
				EnableDynamicScaleDownManagerScaleDownTrigger: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				EnableDynamicScaleDownManagerScaleDownTrigger: testcase.fields.EnableDynamicScaleDownManagerScaleDownTrigger,
			}
			scaleDownIsEnabled := c.IsDataplaneScaleDownTriggerEnabled()
			g.Expect(scaleDownIsEnabled).To(gomega.Equal(testcase.want))
		})
	}
}

func TestDynamicScalingConfig_IsDataplaneScaleUpTriggerEnabled(t *testing.T) {
	t.Parallel()
	type fields struct {
		EnableDynamicScaleUpManagerScaleUpTrigger bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return false when scale up trigger is disabled",
			fields: fields{
				EnableDynamicScaleUpManagerScaleUpTrigger: false,
			},
			want: false,
		},
		{
			name: "return true when scale up trigger is enabled",
			fields: fields{
				EnableDynamicScaleUpManagerScaleUpTrigger: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				EnableDynamicScaleUpManagerScaleUpTrigger: testcase.fields.EnableDynamicScaleUpManagerScaleUpTrigger,
			}
			scaleUpIsEnabled := c.IsDataplaneScaleUpTriggerEnabled()
			g.Expect(scaleUpIsEnabled).To(gomega.Equal(testcase.want))
		})
	}
}

func TestComputeMachineConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		ComputeMachineType      string
		ComputeNodesAutoscaling *ComputeNodesAutoscalingConfig
	}
	type args struct {
		logKey        string
		cloudProvider cloudproviders.CloudProviderID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when ComputeMachineType is missing",
			fields: fields{
				ComputeMachineType: "",
				ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
					MaxComputeNodes: 3,
					MinComputeNodes: 9,
				},
			},
			args: args{
				logKey:        "key",
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when ComputeNodesAutoscaling is missing",
			fields: fields{
				ComputeMachineType:      "some-machine-type",
				ComputeNodesAutoscaling: nil,
			},
			args: args{
				logKey:        "key",
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when min nodes is zero",
			fields: fields{
				ComputeMachineType: "some-machine-type",
				ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
					MaxComputeNodes: 9,
					MinComputeNodes: 0,
				},
			},
			args: args{
				logKey:        "key",
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when min nodes is greater than max nodes",
			fields: fields{
				ComputeMachineType: "some-machine-type",
				ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
					MaxComputeNodes: 1,
					MinComputeNodes: 3,
				},
			},
			args: args{
				logKey:        "key",
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should not return an error when valid configurations are provided",
			fields: fields{
				ComputeMachineType: "some-machine-type",
				ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
					MaxComputeNodes: 1,
					MinComputeNodes: 1,
				},
			},
			args: args{
				logKey:        "key",
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			c := ComputeMachineConfig{
				ComputeMachineType:      testcase.fields.ComputeMachineType,
				ComputeNodesAutoscaling: testcase.fields.ComputeNodesAutoscaling,
			}
			err := c.validate(testcase.args.logKey, testcase.args.cloudProvider)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestComputeMachinesConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		ClusterWideWorkload          *ComputeMachineConfig
		KafkaWorkloadPerInstanceType map[string]ComputeMachineConfig
	}
	type args struct {
		cloudProvider cloudproviders.CloudProviderID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when ClusterWideWorkload is missing",
			fields: fields{
				ClusterWideWorkload: nil,
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {},
				},
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when KafkaWorkloadPerInstanceType is missing",
			fields: fields{
				ClusterWideWorkload:          &ComputeMachineConfig{},
				KafkaWorkloadPerInstanceType: nil,
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when ClusterWideWorkload is invalid when min nodes is less than 3",
			fields: fields{
				ClusterWideWorkload: &ComputeMachineConfig{
					ComputeMachineType: "some-machine",
					ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
						MinComputeNodes: 1, // This will cause a validation error as min should be 3
						MaxComputeNodes: 9,
					},
				},
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "machine-type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 1,
							MinComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when ClusterWideWorkload is invalid when marshalled configurations are missing",
			fields: fields{
				ClusterWideWorkload: &ComputeMachineConfig{
					ComputeMachineType: "",
					ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
						MinComputeNodes: 0,
						MaxComputeNodes: 0,
					},
				},
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "machine-type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 1,
							MinComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should return an error when KafkaWorkloadPerInstanceType is invalid",
			fields: fields{
				ClusterWideWorkload: &ComputeMachineConfig{
					ComputeMachineType: "some-machine",
					ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
						MinComputeNodes: 3,
						MaxComputeNodes: 9,
					},
				},
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "machine-type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 1,
							MinComputeNodes: 0,
						},
					},
				},
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: true,
		},
		{
			name: "should not return an error when valid configurations are provided",
			fields: fields{
				ClusterWideWorkload: &ComputeMachineConfig{
					ComputeMachineType: "some-machine",
					ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
						MinComputeNodes: 3,
						MaxComputeNodes: 9,
					},
				},
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "machine-type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 10,
							MinComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				cloudProvider: cloudproviders.AWS,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			c := ComputeMachinesConfig{
				ClusterWideWorkload:          testcase.fields.ClusterWideWorkload,
				KafkaWorkloadPerInstanceType: testcase.fields.KafkaWorkloadPerInstanceType,
			}
			err := c.validate(testcase.args.cloudProvider)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestComputeMachinesConfig_GetKafkaWorkloadConfigForInstanceType(t *testing.T) {
	t.Parallel()
	type fields struct {
		KafkaWorkloadPerInstanceType map[string]ComputeMachineConfig
	}
	type args struct {
		instanceTypeID string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantConfig  ComputeMachineConfig
		configFound bool
	}{
		{
			name: "return the found configuration and true when configuration for instance type exists",
			fields: fields{
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 1,
							MinComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				instanceTypeID: "type",
			},
			wantConfig: ComputeMachineConfig{
				ComputeMachineType: "type",
				ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
					MaxComputeNodes: 1,
					MinComputeNodes: 1,
				},
			},
			configFound: true,
		},
		{
			name: "return false when configuration for instance type exists",
			fields: fields{
				KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
					"type": {
						ComputeMachineType: "type",
						ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
							MaxComputeNodes: 1,
							MinComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				instanceTypeID: "some-other-type",
			},
			wantConfig:  ComputeMachineConfig{},
			configFound: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			c := &ComputeMachinesConfig{
				KafkaWorkloadPerInstanceType: testcase.fields.KafkaWorkloadPerInstanceType,
			}
			config, ok := c.GetKafkaWorkloadConfigForInstanceType(testcase.args.instanceTypeID)
			g.Expect(config).To(gomega.Equal(testcase.wantConfig))
			g.Expect(ok).To(gomega.Equal(testcase.configFound))
		})
	}
}

func TestDynamicScalingConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		FilePath                       string
		ComputeMachinePerCloudProvider map[cloudproviders.CloudProviderID]ComputeMachinesConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "return an error when FilePath is empty",
			fields: fields{
				FilePath: "",
				ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{
					cloudproviders.AWS: {
						ClusterWideWorkload: &ComputeMachineConfig{
							ComputeMachineType: "some-type",
							ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
								MinComputeNodes: 3,
								MaxComputeNodes: 3,
							},
						},
						KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
							"type": {
								ComputeMachineType: "some-type",
								ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
									MinComputeNodes: 3,
									MaxComputeNodes: 3,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "return an error when ComputeMachinePerCloudProvider is nil",
			fields: fields{
				FilePath:                       "some-file-path",
				ComputeMachinePerCloudProvider: nil,
			},
			wantErr: true,
		},
		{
			name: "return an error when ComputeMachinePerCloudProvider has invalid configuration",
			fields: fields{
				FilePath: "some-file-path",
				ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{
					cloudproviders.AWS: {
						ClusterWideWorkload: &ComputeMachineConfig{
							ComputeMachineType: "some-type",
							ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
								MinComputeNodes: 3,
								MaxComputeNodes: 3,
							},
						},
						KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
							"type": {
								ComputeMachineType: "some-type",
								ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
									MinComputeNodes: 3,
									MaxComputeNodes: 1, // max should be greater than min
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when the configuration is valid",
			fields: fields{
				FilePath: "some-file-path",
				ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{
					cloudproviders.AWS: {
						ClusterWideWorkload: &ComputeMachineConfig{
							ComputeMachineType: "some-type",
							ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
								MinComputeNodes: 3,
								MaxComputeNodes: 3,
							},
						},
						KafkaWorkloadPerInstanceType: map[string]ComputeMachineConfig{
							"type": {
								ComputeMachineType: "some-type",
								ComputeNodesAutoscaling: &ComputeNodesAutoscalingConfig{
									MinComputeNodes: 3,
									MaxComputeNodes: 9,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			c := &DynamicScalingConfig{
				FilePath:                       testcase.fields.FilePath,
				ComputeMachinePerCloudProvider: testcase.fields.ComputeMachinePerCloudProvider,
			}
			err := c.validate()
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
