package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

func TestDataplaneClusterConfig_IsDataPlaneAutoScalingEnabled(t *testing.T) {
	type fields struct {
		DataPlaneClusterScalingType string
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "auto scaling enabled",
			fields: fields{
				DataPlaneClusterScalingType: AutoScaling,
			},
			want: true,
		},
		{
			name: "auto scaling disabled",
			fields: fields{
				DataPlaneClusterScalingType: ManualScaling,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			conf := DataplaneClusterConfig{
				DataPlaneClusterScalingType: tt.fields.DataPlaneClusterScalingType,
			}
			got := conf.IsDataPlaneAutoScalingEnabled()
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestDataplaneClusterConfig_IsDataPlaneManualScalingEnabled(t *testing.T) {
	type fields struct {
		DataPlaneClusterScalingType string
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "manual scaling enabled",
			fields: fields{
				DataPlaneClusterScalingType: ManualScaling,
			},
			want: true,
		},
		{
			name: "manual scaling disabled",
			fields: fields{
				DataPlaneClusterScalingType: AutoScaling,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			conf := DataplaneClusterConfig{
				DataPlaneClusterScalingType: tt.fields.DataPlaneClusterScalingType,
			}
			got := conf.IsDataPlaneManualScalingEnabled()
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestDataplaneClusterConfig_IsWithinClusterLimit(t *testing.T) {
	type fields struct {
		DataPlaneClusterScalingType string
		ClusterList                 ClusterList
	}

	type args struct {
		clusterId string
		count     int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "within the limit",
			fields: fields{
				DataPlaneClusterScalingType: ManualScaling,
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 3},
				},
			},
			args: args{
				clusterId: "test01",
				count:     3,
			},
			want: true,
		},
		{
			name: "exceed the limit",
			fields: fields{
				DataPlaneClusterScalingType: ManualScaling,
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 3},
				},
			},
			args: args{
				clusterId: "test01",
				count:     4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewClusterConfig(tt.fields.ClusterList)
			if got := conf.IsNumberOfKafkaWithinClusterLimit(tt.args.clusterId, tt.args.count); got != tt.want {
				t.Errorf("IsWithinClusterLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataplaneClusterConfig_IsClusterSchedulable(t *testing.T) {
	type fields struct {
		ClusterList ClusterList
	}
	type args struct {
		clusterId string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "schedulable",
			fields: fields{
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", Schedulable: true},
				},
			},
			args: args{
				clusterId: "test01",
			},
			want: true,
		},
		{
			name: "unschedulable",
			fields: fields{
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", Schedulable: false},
				},
			},
			args: args{
				clusterId: "test01",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewClusterConfig(tt.fields.ClusterList)
			if got := conf.IsClusterSchedulable(tt.args.clusterId); got != tt.want {
				t.Errorf("IsClusterSchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataplaneClusterConfig_MissingClusters(t *testing.T) {
	type fields struct {
		ClusterList ClusterList
	}
	type args struct {
		clusterList map[string]api.Cluster
	}
	var emptyResult []ManualCluster
	var result []ManualCluster
	result = append(result, ManualCluster{ClusterId: "test02", Region: "us-east", MultiAZ: true, CloudProvider: "aws"})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []ManualCluster
	}{
		{
			name: "Missing clusters find",
			fields: fields{
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test02", Region: "us-east", MultiAZ: true, CloudProvider: "aws"},
				},
			},
			args: args{
				clusterList: map[string]api.Cluster{
					"test01": {
						ClusterID: "test01",
					},
				},
			},
			want: result,
		},
		{
			name: "No Missing clusters find",
			fields: fields{
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test02", Region: "us-east", MultiAZ: true, CloudProvider: "aws"},
				},
			},
			args: args{
				clusterList: map[string]api.Cluster{
					"test02": {
						ClusterID: "test02",
					},
				},
			},
			want: emptyResult,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewClusterConfig(tt.fields.ClusterList)
			if got := conf.MissingClusters(tt.args.clusterList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MissingClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataplaneClusterConfig_ExcessClusters(t *testing.T) {
	type fields struct {
		ClusterList ClusterList
	}
	type args struct {
		clusterList map[string]api.Cluster
	}
	var emptyResult []string
	var result []string
	result = append(result, "test01")
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "Excess clusters find",
			fields: fields{
				ClusterList{
					ManualCluster{ClusterId: "test02"},
				},
			},
			args: args{
				clusterList: map[string]api.Cluster{
					"test01": {ClusterID: "test01"},
				},
			},
			want: result,
		},
		{
			name: "No Excess clusters find",
			fields: fields{
				ClusterList{
					ManualCluster{ClusterId: "test01"},
				},
			},
			args: args{
				clusterList: map[string]api.Cluster{
					"test01": {ClusterID: "test01"},
				},
			},
			want: emptyResult,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewClusterConfig(tt.fields.ClusterList)
			if got := conf.ExcessClusters(tt.args.clusterList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExcessClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusterConfig_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		output  ManualCluster
		wantErr bool
	}{
		{
			name: "should have default value set",
			input: `
---
name: "test"
cluster_id: "test"
cloud_provider: "aws"
region: "east-1"
multi_az: true
schedulable: true
kafka_instance_limit: 1
`,
			output: ManualCluster{
				Name:                  "test",
				ClusterId:             "test",
				CloudProvider:         "aws",
				Region:                "east-1",
				MultiAZ:               true,
				Schedulable:           true,
				KafkaInstanceLimit:    1,
				Status:                api.ClusterProvisioning,
				ProviderType:          api.ClusterProviderOCM,
				SupportedInstanceType: api.AllInstanceTypeSupport.String(),
			},
			wantErr: false,
		},
		{
			name: "should use the provided value if they are set",
			input: `
---
name: "test"
cluster_id: "test"
cloud_provider: "aws"
region: "east-1"
multi_az: true
schedulable: true
kafka_instance_limit: 1
status: "ready"
provider_type: "aws_eks"
supported_instance_type: "developer"
`,
			output: ManualCluster{
				Name:                  "test",
				ClusterId:             "test",
				CloudProvider:         "aws",
				Region:                "east-1",
				MultiAZ:               true,
				Schedulable:           true,
				KafkaInstanceLimit:    1,
				Status:                api.ClusterReady,
				ProviderType:          api.ClusterProviderAwsEKS,
				SupportedInstanceType: api.DeveloperTypeSupport.String(),
			},
			wantErr: false,
		},
		{
			name: "should return error because invalid status value",
			input: `
---
name: "test"
cluster_id: "test"
cloud_provider: "aws"
region: "east-1"
multi_az: true
schedulable: true
kafka_instance_limit: 1
status: "not_valid"
provider_type: "aws_eks"
`,
			output:  ManualCluster{},
			wantErr: true,
		},
		{
			name: "should return error because invalid provider_type value",
			input: `
---
name: "test"
cluster_id: "test"
cloud_provider: "aws"
region: "east-1"
multi_az: true
schedulable: true
kafka_instance_limit: 1
status: "ready"
provider_type: "invalid"
`,
			output:  ManualCluster{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v ManualCluster
			err := yaml.Unmarshal([]byte(tt.input), &v)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(v, tt.output) {
				t.Errorf("want %v but got %v", tt.output, v)
			}
		})
	}
}

func TestFindClusterNameByClusterId(t *testing.T) {
	type fields struct {
		ClusterList ClusterList
	}
	type args struct {
		clusterId string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "return empty when clusterId does not exist",
			fields: fields{
				ClusterList{
					ManualCluster{
						ClusterId: "12",
						Name:      "sturgis",
					},
				},
			},
			args: args{
				clusterId: "123",
			},
			want: "",
		},
		{
			name: "return correct cluster name when clusterId does exist",
			fields: fields{
				ClusterList{
					ManualCluster{
						ClusterId: "123",
						Name:      "sturgis",
					},
				},
			},
			args: args{
				clusterId: "123",
			},
			want: "sturgis",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			clusterConfig := NewClusterConfig(tt.fields.ClusterList)
			dataplaneClusterConfig := &DataplaneClusterConfig{
				ClusterConfig: clusterConfig,
			}
			got := dataplaneClusterConfig.FindClusterNameByClusterId(tt.args.clusterId)
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestValidateClusterIsInKubeContext(t *testing.T) {
	type args struct {
		rawKubeconfig clientcmdapi.Config
		cluster       ManualCluster
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "return error if cluster is not in kubeconfig context ",
			args: args{
				rawKubeconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{},
				},
				cluster: ManualCluster{
					Name:      "glen",
					ClusterId: "123",
				},
			},
			wantErr: true,
		},
		{
			name: "return nil if cluster is in kubeconfig context ",
			args: args{
				rawKubeconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"glen": {},
					},
				},
				cluster: ManualCluster{
					Name:      "glen",
					ClusterId: "123",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			err := validateClusterIsInKubeconfigContext(tt.args.rawKubeconfig, tt.args.cluster)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
