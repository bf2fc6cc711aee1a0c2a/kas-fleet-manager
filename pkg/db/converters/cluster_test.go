package converters

import (
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
)

var (
	testRegion        = "us-west-1"
	testProvider      = "aws"
	testCloudProvider = "aws"
	testMultiAZ       = true
	testStatus        = api.ClusterProvisioned
	testClusterID     = "123"
	testBYOC          = true
	testManaged       = true
)

// build a test cluster
func buildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
		MultiAZ:       testMultiAZ,
		ClusterID:     testClusterID,
		ExternalID:    testClusterID,
		BYOC:          testBYOC,
		Status:        testStatus,
		Managed:       testManaged,
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
					"managed":        testManaged,
					"status":         testStatus,
					"byoc":           testBYOC,
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
			BYOC:          testBYOC,
			Status:        testStatus,
			Managed:       testManaged,
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
					"managed":        testManaged,
					"status":         testStatus,
					"byoc":           testBYOC,
					"cluster_id":     testClusterID,
					"external_id":    testClusterID,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertClusterList(tt.args.clusterList)

			if err != nil {
				t.Errorf("ConvertClusterList() resulted in an error %v", err)
			}

			if got[0]["cloud_provider"] != tt.want[0]["cloud_provider"] || got[0]["region"] != tt.want[0]["region"] {
				t.Errorf("ConvertClusterList() = got %v, want %v", got, tt.want)
			}
		})
	}
}
