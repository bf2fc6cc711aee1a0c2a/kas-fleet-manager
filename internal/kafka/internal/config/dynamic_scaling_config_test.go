package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/onsi/gomega"
)

func TestInstanceTypeDynamicScalingConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		ComputeNodesConfig *DynamicScalingComputeNodesConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when ComputeNodesConfig is missing",
			fields: fields{
				ComputeNodesConfig: nil,
			},
			wantErr: true,
		},
		{
			name: "should return an error when ComputeNodesConfig is invalid",
			fields: fields{
				ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
					MaxComputeNodes: -10,
				},
			},
			wantErr: true,
		},
		{
			name: "shouldn't return an error when ComputeNodesConfig is valid",
			fields: fields{
				ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
					MaxComputeNodes: 1,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &InstanceTypeDynamicScalingConfig{
				ComputeNodesConfig: testcase.fields.ComputeNodesConfig,
			}
			err := c.validate("instance-type")
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestDynamicScalingConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		filePath                    string
		ComputeNodesPerInstanceType map[string]InstanceTypeDynamicScalingConfig
		MachineTypeConfig           map[cloudproviders.CloudProviderID]MachineTypeConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when filepath is missing",
			fields: fields{
				filePath:                    "", // an empty file path
				ComputeNodesPerInstanceType: map[string]InstanceTypeDynamicScalingConfig{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when ComputeNodesPerInstanceType is missing",
			fields: fields{
				filePath:                    "some-file-path",
				ComputeNodesPerInstanceType: nil, // missing configuration
			},
			wantErr: true,
		},
		{
			name: "should return an error when ComputeNodesPerInstanceType contains an invalid instance type configuration",
			fields: fields{
				filePath: "some-file-path",
				ComputeNodesPerInstanceType: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: nil, // a nil configuration is an invalid configuration
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error when MachineTypeConfig contains an invalid instance type configuration",
			fields: fields{
				filePath: "some-file-path",
				MachineTypeConfig: map[cloudproviders.CloudProviderID]MachineTypeConfig{
					"cp": {
						ClusterWideWorkloadMachineType: "",
						KafkaWorkloadMachineType:       "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "shouldn't return an error when dynamic configuration is valid",
			fields: fields{
				filePath: "some-file-path",
				ComputeNodesPerInstanceType: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
							MaxComputeNodes: 10,
						},
					},
				},
				MachineTypeConfig: map[cloudproviders.CloudProviderID]MachineTypeConfig{
					"cp": {
						ClusterWideWorkloadMachineType: "some-config",
						KafkaWorkloadMachineType:       "some-config",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				filePath:                    testcase.fields.filePath,
				ComputeNodesPerInstanceType: testcase.fields.ComputeNodesPerInstanceType,
				MachineTypePerCloudProvider: testcase.fields.MachineTypeConfig,
			}
			err := c.validate()
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestDynamicScalingConfig_GetConfigForInstanceType(t *testing.T) {
	t.Parallel()
	type fields struct {
		Configuration map[string]InstanceTypeDynamicScalingConfig
	}
	type args struct {
		instanceTypeID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   InstanceTypeDynamicScalingConfig
		found  bool
	}{
		{
			name: "return an empty InstanceTypeDynamicScalingConfig and false when the given instance type does not exist in Configuration",
			fields: fields{
				Configuration: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
							MaxComputeNodes: 10,
						},
					},
				},
			},
			args: args{
				instanceTypeID: "inexisting-instance-type",
			},
			found: false,
			want:  InstanceTypeDynamicScalingConfig{},
		},
		{
			name: "should return the found InstanceTypeDynamicScalingConfig and true when the given instance type exists in Configuration",
			fields: fields{
				Configuration: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
							MaxComputeNodes: 10,
						},
					},
					"instance-type-2": {
						ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
							MaxComputeNodes: 1,
						},
					},
				},
			},
			args: args{
				instanceTypeID: "instance-type-2",
			},
			found: true,
			want: InstanceTypeDynamicScalingConfig{
				ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
					MaxComputeNodes: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				ComputeNodesPerInstanceType: testcase.fields.Configuration,
			}
			instanceTypeDynamicScalingConfig, found := c.GetConfigForInstanceType(testcase.args.instanceTypeID)
			g.Expect(instanceTypeDynamicScalingConfig).To(gomega.Equal(testcase.want))
			g.Expect(found).To(gomega.Equal(testcase.found))
		})
	}
}

func TestNewDynamicScalingConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want DynamicScalingConfig
	}{
		{
			name: "should construct the dynamic scaling config",
			want: DynamicScalingConfig{
				filePath:                                      "config/dynamic-scaling-configuration.yaml",
				ComputeNodesPerInstanceType:                   map[string]InstanceTypeDynamicScalingConfig{},
				MachineTypePerCloudProvider:                   map[cloudproviders.CloudProviderID]MachineTypeConfig{},
				EnableDynamicScaleUpManagerScaleUpTrigger:     true,
				EnableDynamicScaleDownManagerScaleDownTrigger: true,
			},
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			newDynamicScalingConfig := NewDynamicScalingConfig()
			g.Expect(newDynamicScalingConfig).To(gomega.Equal(testcase.want))
		})
	}
}

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
			name: "return false when scale down is disabled",
			fields: fields{
				EnableDynamicScaleDownManagerScaleDownTrigger: false,
			},
			want: false,
		},
		{
			name: "return true when scale down is enabled",
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
			name: "return false when scale up is disabled",
			fields: fields{
				EnableDynamicScaleUpManagerScaleUpTrigger: false,
			},
			want: false,
		},
		{
			name: "return true when scale up is enabled",
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

func TestMachineTypeConfig_validate(t *testing.T) {
	type fields struct {
		ClusterWideWorkloadMachineType string
		KafkaWorkloadMachineType       string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if cluster wide workload machine type is missing",
			fields: fields{
				ClusterWideWorkloadMachineType: "",
				KafkaWorkloadMachineType:       "machine-type",
			},
			wantErr: true,
		},
		{
			name: "should return an error if kafka workload machine type is missing",
			fields: fields{
				ClusterWideWorkloadMachineType: "machine-type",
				KafkaWorkloadMachineType:       "",
			},
			wantErr: true,
		},
		{
			name: "should not return an error if both machine types are present",
			fields: fields{
				ClusterWideWorkloadMachineType: "machine-type",
				KafkaWorkloadMachineType:       "machine-type",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &MachineTypeConfig{
				ClusterWideWorkloadMachineType: testcase.fields.ClusterWideWorkloadMachineType,
				KafkaWorkloadMachineType:       testcase.fields.KafkaWorkloadMachineType,
			}
			err := c.validate("cp")
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
