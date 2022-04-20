package converters

import (
	"testing"

	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	. "github.com/onsi/gomega"
)

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
				cluster: mocks.BuildCluster(nil),
			},
			want: mocks.BuildClusterMap(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertCluster(tt.args.cluster)).To(Equal(tt.want))
		})
	}
}

func Test_ConvertClusterList(t *testing.T) {
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
				clusterList: []api.Cluster{
					*mocks.BuildCluster(nil),
				},
			},
			want: mocks.BuildClusterMap(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertClusterList(tt.args.clusterList)).To(Equal(tt.want))
		})
	}
}

func Test_ConvertClusters(t *testing.T) {
	type args struct {
		clusterList []*api.Cluster
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "ConvertClusters returning a cluster list in form of []map[string]interface{}",
			args: args{
				clusterList: []*api.Cluster{
					mocks.BuildCluster(nil),
				},
			},
			want: mocks.BuildClusterMap(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertClusters(tt.args.clusterList)).To(Equal(tt.want))
		})
	}
}
