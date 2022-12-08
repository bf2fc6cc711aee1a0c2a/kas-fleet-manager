package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

const (
	clusterId = "1234abcd1234abcd1234abcd1234abcd"
	paramId   = "some-id"
	paraValue = "some-value"
	status    = api.ClusterAccepted
)

func Test_PresentEnterpriseCluster(t *testing.T) {
	type args struct {
		cluster          api.Cluster
		fleetShardParams services.ParameterList
	}

	tests := []struct {
		name string
		args args
		want public.EnterpriseCluster
	}{
		{
			name: "should successfully convert api.Cluster to EnterpriseCluster",
			args: args{
				cluster: api.Cluster{
					ClusterID: clusterId,
					Status:    status,
				},
				fleetShardParams: services.ParameterList{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
			},
			want: public.EnterpriseCluster{
				ClusterId: clusterId,
				Status:    status.String(),
				FleetshardParameters: []public.FleetshardParameter{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseCluster(tt.args.cluster, tt.args.fleetShardParams)).To(gomega.Equal(tt.want))
		})
	}
}
