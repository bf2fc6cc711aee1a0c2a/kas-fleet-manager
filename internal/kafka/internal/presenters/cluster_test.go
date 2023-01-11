package presenters

import (
	"fmt"
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

func Test_PresentEnterpriseClusterRegistrationResponse(t *testing.T) {
	type args struct {
		cluster          api.Cluster
		fleetShardParams services.ParameterList
	}

	tests := []struct {
		name string
		args args
		want public.EnterpriseClusterRegistrationResponse
	}{
		{
			name: "should successfully convert api.Cluster to EnterpriseClusterRegistrationResponse",
			args: args{
				cluster: api.Cluster{
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: true,
				},
				fleetShardParams: services.ParameterList{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
			},
			want: public.EnterpriseClusterRegistrationResponse{
				Id:                            clusterId,
				ClusterId:                     clusterId,
				Status:                        status.String(),
				AccessKafkasViaPrivateNetwork: true,
				FleetshardParameters: []public.FleetshardParameter{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
				Kind: "Cluster",
				Href: fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseClusterRegistrationResponse(tt.args.cluster, tt.args.fleetShardParams)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_PresentEnterpriseCluster(t *testing.T) {
	type args struct {
		cluster api.Cluster
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
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: false,
				},
			},
			want: public.EnterpriseCluster{
				Id:                            clusterId,
				ClusterId:                     clusterId,
				Status:                        status.String(),
				Kind:                          "Cluster",
				AccessKafkasViaPrivateNetwork: false,
				Href:                          fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseCluster(tt.args.cluster)).To(gomega.Equal(tt.want))
		})
	}
}
