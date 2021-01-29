package converters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusterservicetest"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"reflect"
	"testing"
)

func TestConvertFromOCMCluster(t *testing.T) {
	type args struct {
		clusterFn func() *clustersmgmtv1.Cluster
	}
	tests := []struct {
		name string
		args args
		want *api.Cluster
	}{
		{
			name: "successful conversion",
			args: args{
				clusterFn: func() *clustersmgmtv1.Cluster {
					cluster, err := clusterservicetest.NewMockCluster(nil)
					if err != nil {
						panic(err)
					}
					return cluster
				},
			},
			want: &api.Cluster{
				CloudProvider: clusterservicetest.MockClusterCloudProvider,
				Region:        clusterservicetest.MockClusterRegion,
				BYOC:          clusterservicetest.MockClusterBYOC,
				Managed:       clusterservicetest.MockClusterManaged,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertCluster(tt.args.clusterFn()); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCluster() = %v, want %v", got, tt.want)
			}
		})
	}
}
