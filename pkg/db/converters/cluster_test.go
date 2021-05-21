package converters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

var (
	testRegion        = "us-west-1"
	testProvider      = "aws"
	testCloudProvider = "aws"
	testMultiAZ       = true
	testStatus        = api.ClusterProvisioned
	testClusterID     = "123"
)

// build a test cluster
func buildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
		MultiAZ:       testMultiAZ,
		ClusterID:     testClusterID,
		ExternalID:    testClusterID,
		Status:        testStatus,
	}
	if modifyFn != nil {
		modifyFn(cluster)
	}
	return cluster
}

func Test_ConvertCluster(t *testing.T) {
	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "ConvertCluster returning a cluster in form of []map[string]interface{}",
			args: args{
				cluster: buildCluster(nil),
			},
			want: []map[string]interface{}{
				{
					"region":         testRegion,
					"cloud_provider": testCloudProvider,
					"multi_az":       testMultiAZ,
					"status":         testStatus,
					"cluster_id":     testClusterID,
					"external_id":    testClusterID,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertCluster(tt.args.cluster)

			if got[0]["cloud_provider"] != tt.want[0]["cloud_provider"] || got[0]["region"] != tt.want[0]["region"] {
				t.Errorf("ConvertCluster() = got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ConvertClusterList(t *testing.T) {
	var nonEmptyClusterList = []api.Cluster{
		api.Cluster{
			Region:        testRegion,
			CloudProvider: testProvider,
			MultiAZ:       testMultiAZ,
			ClusterID:     testClusterID,
			ExternalID:    testClusterID,
			Status:        testStatus,
		},
	}
	type args struct {
		clusterList []api.Cluster
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "ConvertClusterList returning a cluster list in form of []map[string]interface{}",
			args: args{
				clusterList: nonEmptyClusterList,
			},
			want: []map[string]interface{}{
				{
					"region":         testRegion,
					"cloud_provider": testCloudProvider,
					"multi_az":       testMultiAZ,
					"status":         testStatus,
					"cluster_id":     testClusterID,
					"external_id":    testClusterID,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertClusterList(tt.args.clusterList)
			if got[0]["cloud_provider"] != tt.want[0]["cloud_provider"] || got[0]["region"] != tt.want[0]["region"] {
				t.Errorf("ConvertClusterList() = got %v, want %v", got, tt.want)
			}
		})
	}
}
