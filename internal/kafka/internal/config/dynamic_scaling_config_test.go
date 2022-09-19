package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestDynamicScalingComputeNodesConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		MaxComputeNodes int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when MaxComputesNodes is less than 1",
			fields: fields{
				MaxComputeNodes: 0,
			},
			wantErr: true,
		},
		{
			name: "shouldn't return an error when MaxComputesNodes is greater than 0",
			fields: fields{
				MaxComputeNodes: 3,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingComputeNodesConfig{
				MaxComputeNodes: testcase.fields.MaxComputeNodes,
			}

			err := c.validate()
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

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
			err := c.validate()
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestDynamicScalingConfig_validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		filePath      string
		Configuration map[string]InstanceTypeDynamicScalingConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when filepath is missing",
			fields: fields{
				filePath:      "", // an empty file path
				Configuration: map[string]InstanceTypeDynamicScalingConfig{},
			},
			wantErr: true,
		},
		{
			name: "should return an error when Configuration is missing",
			fields: fields{
				filePath:      "some-file-path",
				Configuration: nil, // missing configuration
			},
			wantErr: true,
		},
		{
			name: "should return an error when Configuration contains an invalid instance type configuration",
			fields: fields{
				filePath: "some-file-path",
				Configuration: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: nil, // a nil configuration is an invalid configuration
					},
				},
			},
			wantErr: true,
		},
		{
			name: "shouldn't return an error when Configuration is valid",
			fields: fields{
				filePath: "some-file-path",
				Configuration: map[string]InstanceTypeDynamicScalingConfig{
					"instance-type": {
						ComputeNodesConfig: &DynamicScalingComputeNodesConfig{
							MaxComputeNodes: 10,
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
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				filePath:      testcase.fields.filePath,
				Configuration: testcase.fields.Configuration,
			}
			err := c.validate()
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}

func TestDynamicScalingConfig_InstanceTypeConfigs(t *testing.T) {
	t.Parallel()
	type fields struct {
		Configuration map[string]InstanceTypeDynamicScalingConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]InstanceTypeDynamicScalingConfig
	}{
		{
			name: "should return an empty configuration if the original configuration is empty",
			fields: fields{
				Configuration: map[string]InstanceTypeDynamicScalingConfig{},
			},
			want: map[string]InstanceTypeDynamicScalingConfig{},
		},
		{
			name: "should return a copy of the original configuration map",
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
			want: map[string]InstanceTypeDynamicScalingConfig{
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
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &DynamicScalingConfig{
				Configuration: testcase.fields.Configuration,
			}
			instanceTypeConfigs := c.InstanceTypeConfigs()
			g.Expect(instanceTypeConfigs).To(gomega.Equal(testcase.want))
		})
	}
}

func TestDynamicScalingConfig_ForInstanceType(t *testing.T) {
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
				Configuration: testcase.fields.Configuration,
			}
			instanceTypeDynamicScalingConfig, found := c.ForInstanceType(testcase.args.instanceTypeID)
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
				filePath:      "config/dynamic-scaling-configuration.yaml",
				Configuration: map[string]InstanceTypeDynamicScalingConfig{},
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
