package config

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

func TestOSDClusterConfig_IsDataPlaneAutoScalingEnabled(t *testing.T) {
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
			conf := OSDClusterConfig{
				DataPlaneClusterScalingType: tt.fields.DataPlaneClusterScalingType,
			}
			got := conf.IsDataPlaneAutoScalingEnabled()
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestOSDClusterConfig_IsDataPlaneManualScalingEnabled(t *testing.T) {
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
			conf := OSDClusterConfig{
				DataPlaneClusterScalingType: tt.fields.DataPlaneClusterScalingType,
			}
			got := conf.IsDataPlaneManualScalingEnabled()
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestOSDClusterConfig_IsWithinClusterLimit(t *testing.T) {
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

func TestOSDClusterConfig_IsClusterSchedulable(t *testing.T) {
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

func TestOSDClusterConfig_MissingClusters(t *testing.T) {
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
					"test01": api.Cluster{
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
					"test02": api.Cluster{
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

func TestOSDClusterConfig_ExcessClusters(t *testing.T) {
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
