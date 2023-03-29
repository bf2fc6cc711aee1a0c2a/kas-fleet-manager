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
		cluster                api.Cluster
		fleetShardParams       services.ParameterList
		consumedUnitsInCluster int32
		kafkaConfig            *config.KafkaConfig
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
					CloudProvider:                 "aws",
					Region:                        "us-east-1",
					MultiAZ:                       true,
					DynamicCapacityInfo:           api.JSON(`{"standard":{"max_nodes":8,"max_units":5,"remaining_units":3}}`),
				},
				fleetShardParams: services.ParameterList{
					{
						Id:    paramId,
						Value: paraValue,
					},
				},
				consumedUnitsInCluster: 2,
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								config.KafkaInstanceType{
									Id: types.STANDARD.String(),
									SupportedBillingModels: []config.KafkaBillingModel{
										config.KafkaBillingModel{
											ID: constants.BillingModelEnterprise.String(),
										},
									},
									Sizes: []config.KafkaInstanceSize{
										config.KafkaInstanceSize{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
							},
						},
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
				CloudProvider: "aws",
				Region:        "us-east-1",
				MultiAz:       true,
				Kind:          "Cluster",
				Href:          fmt.Sprintf("/api/kafkas_mgmt/v1/clusters/%s", clusterId),
				SupportedInstanceTypes: public.SupportedKafkaInstanceTypesList{
					InstanceTypes: []public.SupportedKafkaInstanceType{
						public.SupportedKafkaInstanceType{
							Id: types.STANDARD.String(),
							SupportedBillingModels: []public.SupportedKafkaBillingModel{
								public.SupportedKafkaBillingModel{
									Id: constants.BillingModelEnterprise.String(),
								},
							},
							Sizes: []public.SupportedKafkaSize{
								public.SupportedKafkaSize{
									Id:               "x1",
									CapacityConsumed: 1,
								},
							},
						},
					},
				},
				CapacityInformation: public.EnterpriseClusterAllOfCapacityInformation{
					KafkaMachinePoolNodeCount:    8,
					ConsumedKafkaStreamingUnits:  2,
					MaximumKafkaStreamingUnits:   5,
					RemainingKafkaStreamingUnits: 3,
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentEnterpriseClusterRegistrationResponse(tt.args.cluster, tt.args.consumedUnitsInCluster, tt.args.kafkaConfig, tt.args.fleetShardParams)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_PresentEnterpriseCluster(t *testing.T) {
	type args struct {
		cluster     api.Cluster
		kafkaConfig *config.KafkaConfig
	}

	validKafkaConfig := &config.KafkaConfig{
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
	}

	kafkaConfigWithMissingStandardInstanceType := &config.KafkaConfig{
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
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
	}

	kafkaConfigWithMissingEnterpriseBillingModel := &config.KafkaConfig{
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
						},
						SupportedBillingModels: []config.KafkaBillingModel{
							{
								ID: "some-other-billing-model",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		args    args
		want    public.EnterpriseCluster
		wantErr bool
	}{
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
					DynamicCapacityInfo:           api.JSON(`{"standard":{"max_nodes":15,"max_units":5,"remaining_units":3}}`),
				},
				kafkaConfig: validKafkaConfig,
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
								{
									Id:               "x4",
									CapacityConsumed: 4,
								},
								{
									Id:               "x5",
									CapacityConsumed: 5,
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
			name: "should successfully convert api.Cluster to EnterpriseCluster with capacity information and a full list of supported sizes",
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
				kafkaConfig: validKafkaConfig,
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
								{
									Id:               "x4",
									CapacityConsumed: 4,
								},
								{
									Id:               "x5",
									CapacityConsumed: 5,
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
			name: "should return an error when Kafka config is missing standard instance type",
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
				kafkaConfig: kafkaConfigWithMissingStandardInstanceType,
			},
			want:    public.EnterpriseCluster{},
			wantErr: true,
		},
		{
			name: "should return an error when standard instance type is missing the enterprise billing model",
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
				kafkaConfig: kafkaConfigWithMissingEnterpriseBillingModel,
			},
			want:    public.EnterpriseCluster{},
			wantErr: true,
		},
		{
			name: "should return an error when cluster is missing capacity information",
			args: args{
				cluster: api.Cluster{
					ClusterID:                     clusterId,
					Status:                        status,
					AccessKafkasViaPrivateNetwork: false,
					CloudProvider:                 "azure",
					Region:                        "af-east",
					MultiAZ:                       true,
				},
				kafkaConfig: validKafkaConfig,
			},
			want:    public.EnterpriseCluster{},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			presentedCluster, err := PresentEnterpriseCluster(tt.args.cluster, 2, tt.args.kafkaConfig)
			g.Expect(presentedCluster).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
