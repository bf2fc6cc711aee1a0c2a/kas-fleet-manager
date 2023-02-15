package presenters

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
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
		want public.EnterpriseClusterWithAddonParameters
	}{
		{
			name: "should successfully convert api.Cluster to EnterpriseClusterWithAddons",
			args: args{
				cluster: api.Cluster{
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: true,
					CloudProvider:                 "aws",
					Region:                        "us-east-1",
					MultiAZ:                       true,
				},
				fleetShardParams: services.ParameterList{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
			},
			want: public.EnterpriseClusterWithAddonParameters{
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
				CloudProvider: "aws",
				Region:        "us-east-1",
				MultiAz:       true,
				Kind:          "Cluster",
				Href:          fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseClusterWithAddonParams(tt.args.cluster, tt.args.fleetShardParams)).To(gomega.Equal(tt.want))
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
					CloudProvider:                 "aws",
					Region:                        "us-east-1",
					MultiAZ:                       true,
				},
			},
			want: public.EnterpriseCluster{
				Id:                            clusterId,
				ClusterId:                     clusterId,
				Status:                        status.String(),
				Kind:                          "Cluster",
				CloudProvider:                 "aws",
				Region:                        "us-east-1",
				MultiAz:                       true,
				CapacityInformation:           public.EnterpriseClusterAllOfCapacityInformation{},
				AccessKafkasViaPrivateNetwork: false,
				Href:                          fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
		{
			name: "should successfully convert api.Cluster to EnterpriseCluster with capacity information and supported types",
			args: args{
				cluster: api.Cluster{
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: false,
					CloudProvider:                 "azure",
					Region:                        "af-east",
					MultiAZ:                       true,
					DynamicCapacityInfo:           api.JSON([]byte(`{"standard":{"max_nodes":15,"max_units":5,"remaining_units":3}}`)),
				},
			},
			want: public.EnterpriseCluster{
				Id:            clusterId,
				ClusterId:     clusterId,
				Status:        status.String(),
				Kind:          "Cluster",
				CloudProvider: "azure",
				Region:        "af-east",
				MultiAz:       true,
				CapacityInformation: public.EnterpriseClusterAllOfCapacityInformation{
					KafkaMachinePoolNodeCount:    15,
					MaximumKafkaStreamingUnits:   5,
					RemainingKafkaStreamingUnits: 3,
					ConsumedKafkaStreamingUnits:  2,
				},
				AccessKafkasViaPrivateNetwork: false,
				SupportedInstanceTypes: public.SupportedKafkaInstanceTypesList{
					InstanceTypes: []public.SupportedKafkaInstanceType{
						{
							Id: types.STANDARD.String(),
							Sizes: []public.SupportedKafkaSize{
								{
									Id:               "x1",
									CapacityConsumed: 1,
								},
								{
									Id:               "x2",
									CapacityConsumed: 2,
								},
								{
									Id:               "x3",
									CapacityConsumed: 3,
								},
							},
							SupportedBillingModels: []public.SupportedKafkaBillingModel{
								{
									Id: constants.BillingModelEnterprise.String(),
								},
							},
						},
					},
				},
				Href: fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
		{
			name: "should successfully convert api.Cluster to EnterpriseCluster with capacity information and supported types that has an empty list of sizes when there is not capacity left",
			args: args{
				cluster: api.Cluster{
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: false,
					CloudProvider:                 "azure",
					Region:                        "af-east",
					MultiAZ:                       true,
					DynamicCapacityInfo:           api.JSON([]byte(`{"standard":{"max_nodes":6,"max_units":2,"remaining_units":0}}`)),
				},
			},
			want: public.EnterpriseCluster{
				Id:            clusterId,
				ClusterId:     clusterId,
				Status:        status.String(),
				Kind:          "Cluster",
				CloudProvider: "azure",
				Region:        "af-east",
				MultiAz:       true,
				CapacityInformation: public.EnterpriseClusterAllOfCapacityInformation{
					KafkaMachinePoolNodeCount:    6,
					MaximumKafkaStreamingUnits:   2,
					RemainingKafkaStreamingUnits: 0,
					ConsumedKafkaStreamingUnits:  2,
				},
				AccessKafkasViaPrivateNetwork: false,
				SupportedInstanceTypes: public.SupportedKafkaInstanceTypesList{
					InstanceTypes: []public.SupportedKafkaInstanceType{
						{
							Id:    types.STANDARD.String(),
							Sizes: []public.SupportedKafkaSize{},
							SupportedBillingModels: []public.SupportedKafkaBillingModel{
								{
									Id: constants.BillingModelEnterprise.String(),
								},
							},
						},
					},
				},
				Href: fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseCluster(tt.args.cluster, 2, &config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: types.STANDARD.String(),
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x1",
										CapacityConsumed: 1,
									},
									{
										Id:               "x2",
										CapacityConsumed: 2,
									},
									{
										Id:               "x3",
										CapacityConsumed: 3,
									},
									{
										Id:               "x4",
										CapacityConsumed: 4,
									},
									{
										Id:               "x5",
										CapacityConsumed: 5,
									},
								},
								SupportedBillingModels: []config.KafkaBillingModel{
									{
										ID: "some-other-billing-model",
									},
									{
										ID: constants.BillingModelEnterprise.String(),
									},
									{
										ID: "some-other-billing-model-2",
									},
								},
							},
							{
								Id: types.DEVELOPER.String(),
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x1",
										CapacityConsumed: 1,
									},
								},
								SupportedBillingModels: []config.KafkaBillingModel{
									{
										ID: "some-other-billing-model",
									},
									{
										ID: constants.BillingModelEnterprise.String(),
									},
									{
										ID: "some-other-billing-model-2",
									},
								},
							},
						},
					},
				},
			})).To(gomega.Equal(tt.want))
		})
	}
}
