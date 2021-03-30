package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"reflect"
	"testing"
)

func TestOSDClusterConfig_IsWithinClusterLimit(t *testing.T) {
	type fields struct {
		HorizontalScalingConfig string
		ClusterListMap          map[string]ManualCluster
		ClusterList             ClusterList
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
				HorizontalScalingConfig: "manual",
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 3},
				},
				ClusterListMap: map[string]ManualCluster{
					"test01": {KafkaInstanceLimit: 3},
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
				HorizontalScalingConfig: "manual",
				ClusterList: ClusterList{
					ManualCluster{ClusterId: "test01", KafkaInstanceLimit: 3},
				},
				ClusterListMap: map[string]ManualCluster{
					"test01": {KafkaInstanceLimit: 3},
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
			conf := &ClusterConfig{
				ClusterConfigMap: tt.fields.ClusterListMap,
			}
			if got := conf.IsNumberOfKafkaWithinClusterLimit(tt.args.clusterId, tt.args.count); got != tt.want {
				t.Errorf("IsWithinClusterLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOSDClusterConfig_IsClusterSchedulable(t *testing.T) {
	type fields struct {
		HorizontalScalingConfig string
		ClusterListMap          map[string]ManualCluster
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
				HorizontalScalingConfig: "manual",
				ClusterListMap: map[string]ManualCluster{
					"test01": {Schedulable: true},
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
				HorizontalScalingConfig: "manual",
				ClusterListMap: map[string]ManualCluster{
					"test01": {Schedulable: false},
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
			conf := &ClusterConfig{
				ClusterConfigMap: tt.fields.ClusterListMap,
			}
			if got := conf.IsClusterSchedulable(tt.args.clusterId); got != tt.want {
				t.Errorf("IsClusterSchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOSDClusterConfig_MissingClusters(t *testing.T) {
	type fields struct {
		ClusterList    ClusterList
		ClusterListMap map[string]ManualCluster
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
				ClusterListMap: map[string]ManualCluster{
					"test02": {ClusterId: "test02", Region: "us-east", MultiAZ: true, CloudProvider: "aws"},
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
				ClusterListMap: map[string]ManualCluster{
					"test02": {ClusterId: "test02", Region: "us-east", MultiAZ: true, CloudProvider: "aws"},
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
			conf := &ClusterConfig{
				ClusterList:      tt.fields.ClusterList,
				ClusterConfigMap: tt.fields.ClusterListMap,
			}
			if got := conf.MissingClusters(tt.args.clusterList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MissingClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOSDClusterConfig_ExcessClusters(t *testing.T) {
	type fields struct {
		ClusterListMap map[string]ManualCluster
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
				ClusterListMap: map[string]ManualCluster{
					"test02": {},
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
				ClusterListMap: map[string]ManualCluster{
					"test01": {},
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
			conf := &ClusterConfig{
				ClusterConfigMap: tt.fields.ClusterListMap,
			}
			if got := conf.ExcessClusters(tt.args.clusterList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExcessClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}
