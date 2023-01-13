package cluster_mgrs

import (
	"errors"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

func Test_standardDynamicScaleDownProcessor_ShouldScaleDown(t *testing.T) {
	type fields struct {
		standardDynamicScaleDownProcessor *standardDynamicScaleDownProcessor
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    bool
	}{
		{
			name: "should not scale down if at least one streaming unit count is non zero for the given cluster indexes",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
							Count:  0,
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
							Count:  3,
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3},
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "should scale down if all streaming unit count are zero for the given cluster indexes and the no supported instance types in the region",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					regionsSupportedInstanceType: config.InstanceTypeMap{}, // an empty supported instance type
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
							Count:  0,
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
							Count:  0,
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3},
				},
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "should scale down if all streaming unit count are zero for the given cluster indexes and the instance type is not part of supported instance types in the region",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"some-instance-type": config.InstanceTypeConfig{},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:       api.ClusterReady.String(),
							Count:        0,
							InstanceType: "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:       api.ClusterReady.String(),
							Count:        0,
							InstanceType: "instance-type-2",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3},
				},
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "should return an error when evaluating if there is a need to scale up after cluster removal returns an error",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"instance-type-1": config.InstanceTypeConfig{},
						"instance-type-2": config.InstanceTypeConfig{},
					},
					supportedKafkaInstanceTypesConfig: &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id:    "some-instance-type",
								Sizes: []config.KafkaInstanceSize{},
							},
						},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:       api.ClusterReady.String(),
							Count:        0,
							InstanceType: "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:       api.ClusterReady.String(),
							Count:        0,
							InstanceType: "instance-type-2",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3},
				},
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "should not scale down if after the removal of the cluster there will be a need to scale up because largest size cannot fit in the cluster",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"instance-type-1": config.InstanceTypeConfig{},
						"instance-type-2": config.InstanceTypeConfig{},
					},
					supportedKafkaInstanceTypesConfig: &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "instance-type-2",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x4",
										CapacityConsumed: 4,
										QuotaConsumed:    4,
									},
								},
							},
							{
								Id: "instance-type-1",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x4",
										CapacityConsumed: 4,
										QuotaConsumed:    4,
									},
								},
							},
						},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							Count:         0,
							Region:        "r",
							CloudProvider: "c",
							MaxUnits:      2,
							InstanceType:  "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							Count:         0,
							MaxUnits:      3,
							Region:        "r",
							CloudProvider: "c",
							InstanceType:  "instance-type-2",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3},
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "should not scale down if after the removal of the cluster there will be a need to scale up because region slack won't be honoured",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					clusterID: "cluster-1",
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"instance-type-1": config.InstanceTypeConfig{
							MinAvailableCapacitySlackStreamingUnits: 30,
						},
					},
					supportedKafkaInstanceTypesConfig: &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "instance-type-1",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x1",
										CapacityConsumed: 1,
										QuotaConsumed:    1,
									},
								},
							},
						},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							Count:         0,
							Region:        "region",
							CloudProvider: "cp",
							ClusterId:     "cluster-1",
							MaxUnits:      2,
							InstanceType:  "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							Count:         0,
							InstanceType:  "instance-type-1",
							ClusterId:     "cluster-2",
							Region:        "region",
							MaxUnits:      29,
							CloudProvider: "cp",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1},
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "should not scale down if after the removal of the cluster there will be a need to scale up because there is no sibling cluster in the region",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					clusterID: "cluster-1",
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"instance-type-1": config.InstanceTypeConfig{
							MinAvailableCapacitySlackStreamingUnits: 1,
						},
					},
					supportedKafkaInstanceTypesConfig: &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "instance-type-1",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x1",
										CapacityConsumed: 1,
										QuotaConsumed:    1,
									},
								},
							},
						},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							Count:         0,
							Region:        "region",
							CloudProvider: "cp",
							ClusterId:     "cluster-1",
							MaxUnits:      2,
							InstanceType:  "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						// even though this cluster is in the same region as "cluster-1", it is not ready yet so it is not considered as a sibling cluster
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterProvisioned.String(),
							Count:         0,
							Region:        "region",
							CloudProvider: "cp",
							ClusterId:     "cluster-2",
							MaxUnits:      0,
							InstanceType:  "instance-type-1",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1},
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "should scale down if after the removal of the cluster there wont be a need to scale up",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					clusterID: "cluster-1",
					regionsSupportedInstanceType: config.InstanceTypeMap{
						"instance-type-1": config.InstanceTypeConfig{
							MinAvailableCapacitySlackStreamingUnits: 1,
						},
					},
					supportedKafkaInstanceTypesConfig: &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "instance-type-1",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:               "x1",
										CapacityConsumed: 1,
										QuotaConsumed:    1,
									},
								},
							},
						},
					},
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status:      api.ClusterReady.String(),
							ClusterType: api.ManagedDataPlaneClusterType.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							ClusterType:   api.ManagedDataPlaneClusterType.String(),
							Count:         0,
							Region:        "region",
							CloudProvider: "cp",
							ClusterId:     "cluster-1",
							MaxUnits:      2,
							InstanceType:  "instance-type-1",
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status:        api.ClusterReady.String(),
							ClusterType:   api.ManagedDataPlaneClusterType.String(),
							Count:         0,
							Region:        "region",
							CloudProvider: "cp",
							ClusterId:     "cluster-2",
							MaxUnits:      2,
							InstanceType:  "instance-type-1",
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1},
				},
			},
			wantErr: false,
			want:    true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			shouldScaleDown, err := tt.fields.standardDynamicScaleDownProcessor.ShouldScaleDown()
			if tt.wantErr {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(shouldScaleDown).To(gomega.BeFalse())
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(shouldScaleDown).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_standardDynamicScaleDownProcessor_ScaleDown(t *testing.T) {
	type fields struct {
		standardDynamicScaleDownProcessor *standardDynamicScaleDownProcessor
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    services.KafkaStreamingUnitCountPerClusterList
	}{
		{
			name: "When dryRun is true, no error should be returned and clusterService should never be called",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					clusterService: &services.ClusterServiceMock{
						UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
							return errors.New("should never be called")
						},
					},
					dryRun: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Should expect an error to be returned when cluster service status update returns an error",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{},
					clusterService: &services.ClusterServiceMock{
						UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
							return errors.New("some error")
						},
					},
					dryRun: false,
				},
			},
			wantErr: true,
		},
		{
			name: "Should successful set the status of the cluster to deprovisioning",
			fields: fields{
				standardDynamicScaleDownProcessor: &standardDynamicScaleDownProcessor{
					clusterID: "some-cluster-id",
					kafkaStreamingUnitCountPerClusterList: services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
						services.KafkaStreamingUnitCountPerCluster{
							Status: api.ClusterReady.String(),
						},
					},
					indexesOfStreamingUnitForSameClusterID: []int{1, 3}, // the second and the fourth streaming unit in kafkaStreamingUnitCountPerClusterList should see their status changing to deprovisioning
					clusterService: &services.ClusterServiceMock{
						UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
							return nil
						},
					},
					dryRun: false,
				},
			},
			wantErr: false,
			want: services.KafkaStreamingUnitCountPerClusterList{
				services.KafkaStreamingUnitCountPerCluster{
					Status: api.ClusterReady.String(),
				},
				services.KafkaStreamingUnitCountPerCluster{
					Status: api.ClusterDeprovisioning.String(), // status changed to deprovisioning
				},
				services.KafkaStreamingUnitCountPerCluster{
					Status: api.ClusterReady.String(),
				},
				services.KafkaStreamingUnitCountPerCluster{
					Status: api.ClusterDeprovisioning.String(), // status changed to deprovisioning
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := tt.fields.standardDynamicScaleDownProcessor.ScaleDown()
			if tt.wantErr {
				g.Expect(err).To(gomega.HaveOccurred())
			} else {
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(tt.fields.standardDynamicScaleDownProcessor.kafkaStreamingUnitCountPerClusterList).To(gomega.Equal(tt.want))
				clusterServiceMock, ok := tt.fields.standardDynamicScaleDownProcessor.clusterService.(*services.ClusterServiceMock)
				g.Expect(ok).To(gomega.BeTrue())
				updateStatusCalls := clusterServiceMock.UpdateStatusCalls()
				updateStatusCallsCount := len(updateStatusCalls)
				g.Expect(updateStatusCallsCount > 0).To(gomega.Equal(!tt.fields.standardDynamicScaleDownProcessor.dryRun)) // should only be called if not dryRun
				if updateStatusCallsCount > 0 {
					g.Expect(updateStatusCallsCount == 1).To(gomega.BeTrue()) // check that it update status is only called once
					// now check that the arguments matches what we expect
					g.Expect(updateStatusCalls[0].Status).To(gomega.Equal(api.ClusterDeprovisioning))
					g.Expect(updateStatusCalls[0].Cluster).To(gomega.Equal(api.Cluster{
						ClusterID: tt.fields.standardDynamicScaleDownProcessor.clusterID,
					}))
				}
			}
		})
	}
}

func Test_DynamicScaleDownManager_Reconcile(t *testing.T) {
	type fields struct {
		dataplaneClusterConfig *config.DataplaneClusterConfig
		clusterProvidersConfig *config.ProviderConfig
		kafkaConfig            *config.KafkaConfig
		clusterService         services.ClusterService
	}

	tests := []struct {
		name                      string
		fields                    fields
		wantErr                   bool
		wantUpdateStatusCallCount int
	}{
		{
			name: "Should not reconcile when scaling mode is not autoscaling",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
				},
				clusterService:         &services.ClusterServiceMock{},
				kafkaConfig:            nil, // should never be needed
				clusterProvidersConfig: nil, // should never be needed
			},
			wantErr: false,
		},
		{
			name: "Should expect an error when clusterService.FindStreamingUnitCountByClusterAndInstanceType returns an error",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return nil, errors.New("some errors")
					},
					UpdateStatusFunc: nil, // should never be called
				},
				kafkaConfig:            nil, // should never be needed
				clusterProvidersConfig: nil, // should never be needed
			},
			wantErr: true,
		},
		{
			name: "Should not expect an error when clusterService.FindStreamingUnitCountByClusterAndInstanceType returns empty",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return nil, nil
					},
					UpdateStatusFunc: nil, // should never be called,
				},
				kafkaConfig:            nil, // should never be needed
				clusterProvidersConfig: nil, // should never be needed
			},
			wantErr: false,
		},
		{
			name: "Should never call clusterService.UpdateStatus when there is no need to scale down",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:       0,
								ClusterType: api.ManagedDataPlaneClusterType.String(),
								Status:      api.ClusterAccepted.String(),
							},
						}, nil
					},
					UpdateStatusFunc: nil, // should never be called,
				},
				kafkaConfig:            nil, // should never be needed
				clusterProvidersConfig: nil, // should never be needed
			},
			wantErr: false,
		},
		{
			name: "Should not check for scale up evaluation after cluster removal when cloud provider is not provided",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:         0,
								ClusterId:     "2",
								Status:        api.ClusterReady.String(),
								CloudProvider: "cp",
								Region:        "r",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{},
					},
				},
			},
			wantErr:                   false,
			wantUpdateStatusCallCount: 1,
		},
		{
			name: "Should not check for scale up evaluation after cluster removal when region limits are not provided",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:         0,
								ClusterId:     "2",
								Status:        api.ClusterReady.String(),
								CloudProvider: "cp",
								Region:        "r",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "cp",
								Regions: config.RegionList{},
							},
						},
					},
				},
			},
			wantErr:                   false,
			wantUpdateStatusCallCount: 1,
		},
		{
			name: "Should never call clusterService.UpdateStatus when there is no need to scale down",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:       0,
								ClusterType: api.ManagedDataPlaneClusterType.String(),
								Status:      api.ClusterAccepted.String(),
							},
						}, nil
					},
					UpdateStatusFunc: nil, // should never be called,
				},
				kafkaConfig:            nil, // should never be needed
				clusterProvidersConfig: nil, // should never be needed
			},
			wantErr: false,
		},
		{
			name: "Should call clusterService.UpdateStatus only once for each occurrence of the same cluster that needs scale down",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Status:      api.ClusterReady.String(),
								ClusterType: api.ManagedDataPlaneClusterType.String(),
								ClusterId:   "other-cluster-id",
								Count:       4,
								MaxUnits:    4,
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-3",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-2",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-3",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-2",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
								{
									Id: "instance-type-2",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
								{
									Id: "instance-type-3",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
							},
						},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "cp",
								Regions: config.RegionList{
									config.Region{
										Name: "region",
										SupportedInstanceTypes: config.InstanceTypeMap{
											"instance-type-1": config.InstanceTypeConfig{
												Limit:                                   nil,
												MinAvailableCapacitySlackStreamingUnits: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr:                   false,
			wantUpdateStatusCallCount: 1,
		},
		{
			name: "Should call clusterService.UpdateStatus as many times as there are clusters to deprovision",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Status:      api.ClusterReady.String(),
								ClusterType: api.ManagedDataPlaneClusterType.String(),
								ClusterId:   "other-cluster-id",
								Count:       4,
								MaxUnits:    4,
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-3",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-3",
								MaxUnits:      2,
								InstanceType:  "instance-type-3",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-2",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
								{
									Id: "instance-type-2",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
								{
									Id: "instance-type-3",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
							},
						},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "cp",
								Regions: config.RegionList{
									config.Region{
										Name: "region",
										SupportedInstanceTypes: config.InstanceTypeMap{
											"instance-type-1": config.InstanceTypeConfig{
												Limit:                                   nil,
												MinAvailableCapacitySlackStreamingUnits: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr:                   false,
			wantUpdateStatusCallCount: 2,
		},
		{
			name: "Should return an error when evaluating scale down returns an error",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-1",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
							services.KafkaStreamingUnitCountPerCluster{
								Status:        api.ClusterReady.String(),
								ClusterType:   api.ManagedDataPlaneClusterType.String(),
								Count:         0,
								Region:        "region",
								CloudProvider: "cp",
								ClusterId:     "cluster-2",
								MaxUnits:      2,
								InstanceType:  "instance-type-1",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.New("some errors")
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-2",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 1,
										},
									},
								},
							},
						},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "cp",
								Regions: config.RegionList{
									config.Region{
										Name: "region",
										SupportedInstanceTypes: config.InstanceTypeMap{
											"instance-type-1": config.InstanceTypeConfig{
												Limit:                                   nil,
												MinAvailableCapacitySlackStreamingUnits: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should expect an error when clusterService.UpdateStatus returns an error",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: true,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:         0,
								ClusterId:     "1",
								Status:        api.ClusterReady.String(),
								CloudProvider: "cp",
								Region:        "r",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.New("some errors")
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "cp",
								Regions: config.RegionList{},
							},
						},
					},
				},
			},
			wantErr:                   true,
			wantUpdateStatusCallCount: 1,
		},
		{
			name: "Should never call clusterService.UpdateStatus when dry run",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						EnableDynamicScaleDownManagerScaleDownTrigger: false,
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							services.KafkaStreamingUnitCountPerCluster{
								Count:         0,
								ClusterId:     "1",
								Status:        api.ClusterReady.String(),
								CloudProvider: "cp",
								Region:        "r",
							},
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.New("some errors")
					},
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{},
					},
				},
				clusterProvidersConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "cp",
								Regions: config.RegionList{},
							},
						},
					},
				},
			},
			wantErr:                   false,
			wantUpdateStatusCallCount: 0,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			mgr := DynamicScaleDownManager{
				dataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				clusterProvidersConfig: tt.fields.clusterProvidersConfig,
				kafkaConfig:            tt.fields.kafkaConfig,
				clusterService:         tt.fields.clusterService,
			}

			errs := mgr.Reconcile()
			g.Expect(len(errs) > 0).To(gomega.Equal(tt.wantErr))

			clusterServiceMock, ok := tt.fields.clusterService.(*services.ClusterServiceMock)
			g.Expect(ok).To(gomega.BeTrue())
			updateStatusCalls := clusterServiceMock.UpdateStatusCalls()
			updateStatusCallsCount := len(updateStatusCalls)
			g.Expect(updateStatusCallsCount).To(gomega.Equal(tt.wantUpdateStatusCallCount))
			if updateStatusCallsCount > 0 {
				g.Expect(updateStatusCalls[0].Status).To(gomega.Equal(api.ClusterDeprovisioning))
			}

			findStreamingUnitCountByClusterAndInstanceTypeCalls := clusterServiceMock.FindStreamingUnitCountByClusterAndInstanceTypeCalls()
			findStreamingUnitCountByClusterAndInstanceTypeCallsCount := len(findStreamingUnitCountByClusterAndInstanceTypeCalls)
			if tt.fields.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
				g.Expect(findStreamingUnitCountByClusterAndInstanceTypeCallsCount).To(gomega.Equal(1))
			} else {
				g.Expect(findStreamingUnitCountByClusterAndInstanceTypeCallsCount).To(gomega.Equal(0))
			}
		})
	}
}
