package cluster_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func Test_noopDynamicScaleUpProcessor_ShouldScaleUp(t *testing.T) {
	g := gomega.NewWithT(t)

	p := noopDynamicScaleUpProcessor{}
	shouldScaleUp, err := p.ShouldScaleUp()
	g.Expect(shouldScaleUp).To(gomega.BeFalse())
	g.Expect(err).NotTo(gomega.HaveOccurred())
}
func Test_noopDynamicScaleUpProcessor_ScaleUp(t *testing.T) {
	g := gomega.NewWithT(t)

	p := noopDynamicScaleUpProcessor{}
	err := p.ScaleUp()
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func Test_standardDynamicScaleUpProcessor_ShouldScaleUp(t *testing.T) {
	type fields struct {
		locator                                      supportedInstanceTypeLocator
		kafkaStreamingUnitCountPerClusterListFactory func() services.KafkaStreamingUnitCountPerClusterList
		supportedKafkaInstanceTypesConfigFactory     func() *config.SupportedKafkaInstanceTypesConfig
		instanceTypeConfig                           *config.InstanceTypeConfig
	}

	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "When the region limit has been reached then no scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					Limit: &[]int{5}[0],
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "When there is an ongoing scale up action then no scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         1,
						MaxUnits:      1,
						Status:        api.ClusterProvisioning.String(),
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				instanceTypeConfig: &config.InstanceTypeConfig{},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "When the biggest kafka size of the provided instance type does not fit in any cluster then scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							config.KafkaInstanceType{
								Id: "t1",
								Sizes: []config.KafkaInstanceSize{
									config.KafkaInstanceSize{
										Id:               "s1",
										CapacityConsumed: 1,
									},
									config.KafkaInstanceSize{
										Id:               "s2",
										CapacityConsumed: 2,
									},
									config.KafkaInstanceSize{
										Id:               "s3",
										CapacityConsumed: 4,
									},
								},
							},
						},
					}
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "When there is not enough capacity slack then scale up is performed (assuming the biggest kafka instance size can fit in at least one cluster)",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					MinAvailableCapacitySlackStreamingUnits: 5,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "When the configured capacity slack is 0 then no scale up is performed (assuming the biggest kafka instance size can fit in at least one cluster)",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					return services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							CloudProvider: locator.provider,
							Region:        locator.region,
							InstanceType:  locator.instanceTypeName,
							Count:         8,
							MaxUnits:      10,
							Status:        api.ClusterReady.String(),
							ClusterType:   locator.clusterType,
						},
					}
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					MinAvailableCapacitySlackStreamingUnits: 0,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "When there is enough capacity slack and enough capacity to fit the biggest kafka size in at least one cluster then scale up is not performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					MinAvailableCapacitySlackStreamingUnits: 1,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "If the region limit has been reached no scale up is performed even if the biggest kafka size of the provided instance type does not fit in any cluster then scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							config.KafkaInstanceType{
								Id: "t1",
								Sizes: []config.KafkaInstanceSize{
									config.KafkaInstanceSize{
										Id:               "s1",
										CapacityConsumed: 1,
									},
									config.KafkaInstanceSize{
										Id:               "s2",
										CapacityConsumed: 2,
									},
									config.KafkaInstanceSize{
										Id:               "s3",
										CapacityConsumed: 4,
									},
								},
							},
						},
					}
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					Limit: &[]int{5}[0],
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "If there is an ongoing scale up action no scale up is performed even if the biggest kafka size of the provided instance type does not fit in any cluster then scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							config.KafkaInstanceType{
								Id: "t1",
								Sizes: []config.KafkaInstanceSize{
									config.KafkaInstanceSize{
										Id:               "s1",
										CapacityConsumed: 1,
									},
									config.KafkaInstanceSize{
										Id:               "s2",
										CapacityConsumed: 2,
									},
									config.KafkaInstanceSize{
										Id:               "s3",
										CapacityConsumed: 4,
									},
								},
							},
						},
					}
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         1,
						MaxUnits:      1,
						Status:        api.ClusterProvisioning.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				instanceTypeConfig: &config.InstanceTypeConfig{},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Even when there is enough capacity slack, if the biggest kafka instance size cannot fit in any cluster a scale up is performed",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							config.KafkaInstanceType{
								Id: "t1",
								Sizes: []config.KafkaInstanceSize{
									config.KafkaInstanceSize{
										Id:               "s1",
										CapacityConsumed: 1,
									},
									config.KafkaInstanceSize{
										Id:               "s2",
										CapacityConsumed: 2,
									},
									config.KafkaInstanceSize{
										Id:               "s3",
										CapacityConsumed: 4,
									},
								},
							},
						},
					}
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					MinAvailableCapacitySlackStreamingUnits: 1,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "When there is an error with the summary calculator an error is returned",
			fields: fields{
				locator: supportedInstanceTypeLocator{
					provider:         "p1",
					region:           "r1",
					instanceTypeName: "unexistingInstanceType",
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				instanceTypeConfig: &config.InstanceTypeConfig{
					MinAvailableCapacitySlackStreamingUnits: 1,
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			standardDynamicScaleUpProcessor := standardDynamicScaleUpProcessor{
				locator:                               tt.fields.locator,
				instanceTypeConfig:                    tt.fields.instanceTypeConfig,
				kafkaStreamingUnitCountPerClusterList: tt.fields.kafkaStreamingUnitCountPerClusterListFactory(),
				supportedKafkaInstanceTypesConfig:     tt.fields.supportedKafkaInstanceTypesConfigFactory(),
			}

			res, err := standardDynamicScaleUpProcessor.ShouldScaleUp()

			g.Expect(res).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

		})
	}

}

func Test_standardDynamicScaleUpProcessor_enoughCapacitySlackInRegion(t *testing.T) {
	type fields struct {
		standardDynamicScaleUpProcessor *standardDynamicScaleUpProcessor
	}

	type args struct {
		summary instanceTypeConsumptionSummary
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "When the supported instance type has more free streaming units than the defined slack capacity there is enough capcity slack in the region",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						MinAvailableCapacitySlackStreamingUnits: 3,
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					freeStreamingUnits: 5,
				},
			},
			want: true,
		},
		{
			name: "When the supported instance type has the same free streaming units than the defined slack capacity there is enough capcity slack in the region",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						MinAvailableCapacitySlackStreamingUnits: 5,
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					freeStreamingUnits: 5,
				},
			},
			want: true,
		},
		{
			name: "When the supported instance type has less free streaming units than the defined slack capacity there is not enough capcity slack in the region",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						MinAvailableCapacitySlackStreamingUnits: 5,
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					freeStreamingUnits: 3,
				},
			},
			want: false,
		},
		{
			name: "When the defined slack capacity is 0 there is always capacity slack in the region",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						MinAvailableCapacitySlackStreamingUnits: 0,
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					freeStreamingUnits: 0,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.standardDynamicScaleUpProcessor.enoughCapacitySlackInRegion(tt.args.summary)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_standardDynamicScaleUpProcessor_regionLimitReached(t *testing.T) {
	type fields struct {
		standardDynamicScaleUpProcessor *standardDynamicScaleUpProcessor
	}

	type args struct {
		summary instanceTypeConsumptionSummary
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "When the supported instance type consumed streaming units has reached the region limit true is returned",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						Limit: &[]int{6}[0],
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					consumedStreamingUnits: 6,
				},
			},
			want: true,
		},
		{
			name: "When the supported instance type has consumed more streaming units than the region limit then the region limit has been reached",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						Limit: &[]int{6}[0],
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					consumedStreamingUnits: 7,
				},
			},
			want: true,
		},
		{
			name: "When there is no region limit configured then the region limit is never reached",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						Limit: nil,
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					consumedStreamingUnits: 1000,
				},
			},
			want: false,
		},
		{
			name: "When the region limit is 0 then the limit is always reached",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					instanceTypeConfig: &config.InstanceTypeConfig{
						Limit: &[]int{0}[0],
					},
				},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					consumedStreamingUnits: 0,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.standardDynamicScaleUpProcessor.regionLimitReached(tt.args.summary)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_standardDynamicScaleUpProcessor_freeCapacityForBiggestInstanceSize(t *testing.T) {
	type fields struct {
		standardDynamicScaleUpProcessor *standardDynamicScaleUpProcessor
	}

	type args struct {
		summary instanceTypeConsumptionSummary
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "When the received summary has biggestInstanceSizeCapacityAvailable set to true then that value is returned",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					biggestInstanceSizeCapacityAvailable: true,
				},
			},
			want: true,
		},
		{
			name: "When the received summary has biggestInstanceSizeCapacityAvailable set to false then that value is returned",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					biggestInstanceSizeCapacityAvailable: false,
				},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.standardDynamicScaleUpProcessor.freeCapacityForBiggestInstanceSize(tt.args.summary)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_standardDynamicScaleUpProcessor_ongoingScaleUpAction(t *testing.T) {
	type fields struct {
		standardDynamicScaleUpProcessor *standardDynamicScaleUpProcessor
	}

	type args struct {
		summary instanceTypeConsumptionSummary
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "When the received summary has ongoingScaleUpAction set to true then that value is returned",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					ongoingScaleUpAction: true,
				},
			},
			want: true,
		},
		{
			name: "When the received summary has ongoingScaleUpAction set to false then that value is returned",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{},
			},
			args: args{
				summary: instanceTypeConsumptionSummary{
					ongoingScaleUpAction: false,
				},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.standardDynamicScaleUpProcessor.ongoingScaleUpAction(tt.args.summary)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_standardDynamicScaleUpProcessor_ScaleUp(t *testing.T) {
	type fields struct {
		standardDynamicScaleUpProcessor *standardDynamicScaleUpProcessor
	}
	testStandardIntanceTypeLocator := supportedInstanceTypeLocator{
		provider:         "p1",
		region:           "r1",
		instanceTypeName: api.StandardTypeSupport.String(),
	}
	testDeveloperInstanceTypeLocator := supportedInstanceTypeLocator{
		provider:         "p1",
		region:           "r1",
		instanceTypeName: api.DeveloperTypeSupport.String(),
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "When ScaleUp is called for a new cluster of supported standard instance type it is registered with the appropriate attributes",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					locator:                               testStandardIntanceTypeLocator,
					instanceTypeConfig:                    &config.InstanceTypeConfig{},
					kafkaStreamingUnitCountPerClusterList: newTestHelperBaseKafkaStreamingUnitCountPerClusterList(),
					supportedKafkaInstanceTypesConfig:     newTestHelperBaseSupportedKafkaInstanceTypesConfig(),
					clusterService: &services.ClusterServiceMock{
						RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *apiErrors.ServiceError {
							return nil
						},
					},
					dryRun: false,
				},
			},
			want: &api.Cluster{
				CloudProvider:         testStandardIntanceTypeLocator.provider,
				Region:                testStandardIntanceTypeLocator.region,
				SupportedInstanceType: testStandardIntanceTypeLocator.instanceTypeName,
				Status:                api.ClusterAccepted,
				ProviderType:          api.ClusterProviderOCM,
				ClusterType:           api.ManagedDataPlaneClusterType.String(),
				MultiAZ:               true,
			},
			wantErr: false,
		},
		{
			name: "When ScaleUp is called for a new cluster of supported developer instance type it is registerd with the appropriate attributes",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					locator:                               testDeveloperInstanceTypeLocator,
					instanceTypeConfig:                    &config.InstanceTypeConfig{},
					kafkaStreamingUnitCountPerClusterList: newTestHelperBaseKafkaStreamingUnitCountPerClusterList(),
					supportedKafkaInstanceTypesConfig:     newTestHelperBaseSupportedKafkaInstanceTypesConfig(),
					clusterService: &services.ClusterServiceMock{
						RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *apiErrors.ServiceError {
							return nil
						},
					},
					dryRun: false,
				},
			},
			want: &api.Cluster{
				CloudProvider:         testDeveloperInstanceTypeLocator.provider,
				Region:                testDeveloperInstanceTypeLocator.region,
				SupportedInstanceType: testDeveloperInstanceTypeLocator.instanceTypeName,
				Status:                api.ClusterAccepted,
				ProviderType:          api.ClusterProviderOCM,
				ClusterType:           api.ManagedDataPlaneClusterType.String(),
				MultiAZ:               false,
			},
			wantErr: false,
		},
		{
			name: "When ScaleUp is called with a processor with dry run enabled then no cluster is registered",
			fields: fields{
				standardDynamicScaleUpProcessor: &standardDynamicScaleUpProcessor{
					locator:                               newTestHelperBaseSupportedInstanceTypeLocator(),
					instanceTypeConfig:                    &config.InstanceTypeConfig{},
					kafkaStreamingUnitCountPerClusterList: newTestHelperBaseKafkaStreamingUnitCountPerClusterList(),
					supportedKafkaInstanceTypesConfig:     newTestHelperBaseSupportedKafkaInstanceTypesConfig(),
					clusterService: &services.ClusterServiceMock{
						RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *apiErrors.ServiceError {
							return nil
						},
					},
					dryRun: true,
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := tt.fields.standardDynamicScaleUpProcessor.ScaleUp()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			// We test whether the cluster creation has been registered depending
			// on the dryRun value
			clusterServiceMock, ok := tt.fields.standardDynamicScaleUpProcessor.clusterService.(*services.ClusterServiceMock)
			g.Expect(ok).To(gomega.BeTrue())
			registerClusterJobCalls := clusterServiceMock.RegisterClusterJobCalls()
			registerClusterJobCalled := len(registerClusterJobCalls) > 0
			registerClusterJobCallExpected := !tt.fields.standardDynamicScaleUpProcessor.dryRun
			g.Expect(registerClusterJobCalled).To(gomega.Equal(registerClusterJobCallExpected))
			if registerClusterJobCallExpected {
				g.Expect(registerClusterJobCalls).To(gomega.HaveLen(1))
				registerClusterJobcall := registerClusterJobCalls[0]
				g.Expect(registerClusterJobcall.ClusterRequest).To(gomega.Equal(tt.want))
			}
		})
	}
}

func newTestHelperBaseSupportedKafkaInstanceTypesConfig() *config.SupportedKafkaInstanceTypesConfig {
	return &config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			config.KafkaInstanceType{
				Id: "t1",
				Sizes: []config.KafkaInstanceSize{
					config.KafkaInstanceSize{
						Id:               "s1",
						CapacityConsumed: 1,
					},
					config.KafkaInstanceSize{
						Id:               "s2",
						CapacityConsumed: 2,
					},
				},
			},
		},
	}
}

func newTestHelperBaseSupportedInstanceTypeLocator() supportedInstanceTypeLocator {
	return supportedInstanceTypeLocator{
		provider:         "p1",
		region:           "r1",
		instanceTypeName: "t1",
		clusterType:      "c1",
	}
}
func newTestHelperBaseKafkaStreamingUnitCountPerClusterList() services.KafkaStreamingUnitCountPerClusterList {
	locator := newTestHelperBaseSupportedInstanceTypeLocator()
	return services.KafkaStreamingUnitCountPerClusterList{
		services.KafkaStreamingUnitCountPerCluster{
			CloudProvider: locator.provider,
			Region:        locator.region,
			InstanceType:  locator.instanceTypeName,
			Count:         1,
			MaxUnits:      2,
			Status:        api.ClusterReady.String(),
			ClusterType:   locator.clusterType,
		},
		services.KafkaStreamingUnitCountPerCluster{
			CloudProvider: locator.provider,
			Region:        locator.region,
			InstanceType:  locator.instanceTypeName,
			Count:         4,
			MaxUnits:      6,
			Status:        api.ClusterReady.String(),
			ClusterType:   locator.clusterType,
		},
		// should be ignored in capacity calculations
		services.KafkaStreamingUnitCountPerCluster{
			CloudProvider: locator.provider,
			Region:        locator.region,
			InstanceType:  locator.instanceTypeName,
			Count:         4,
			MaxUnits:      6,
			Status:        api.ClusterReady.String(),
			ClusterType:   api.EnterpriseDataPlaneClusterType.String(),
		},
	}
}

func Test_supportedInstanceTypeLocator_Equal(t *testing.T) {
	type fields struct {
		locator supportedInstanceTypeLocator
	}

	type args struct {
		locator supportedInstanceTypeLocator
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "locators with same provider, region, instance type name and are equal",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			args: args{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			want: true,
		},
		{
			name: "locators with a different provider are not equal",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			args: args{
				locator: supportedInstanceTypeLocator{
					provider:         "p2",
					region:           "r1",
					instanceTypeName: "t1",
					clusterType:      "c1",
				},
			},
			want: false,
		},
		{
			name: "locators with a different region are not equal",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			args: args{
				locator: supportedInstanceTypeLocator{
					provider:         "p1",
					region:           "r2",
					instanceTypeName: "t1",
					clusterType:      "c1",
				},
			},
			want: false,
		},
		{
			name: "locators with a different instance type name are not equal",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			args: args{
				locator: supportedInstanceTypeLocator{
					provider:         "p1",
					region:           "r1",
					instanceTypeName: "t2",
					clusterType:      "c1",
				},
			},
			want: false,
		},
		{
			name: "locators with a different cluster type name are not equal",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
			},
			args: args{
				locator: supportedInstanceTypeLocator{
					provider:         "p1",
					region:           "r1",
					instanceTypeName: "t1",
					clusterType:      "c2",
				},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.locator.Equal(tt.args.locator)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_instanceTypeConsumptionSummaryCalculator_Calculate(t *testing.T) {
	type fields struct {
		locator                                      supportedInstanceTypeLocator
		kafkaStreamingUnitCountPerClusterListFactory func() services.KafkaStreamingUnitCountPerClusterList
		supportedKafkaInstanceTypesConfigFactory     func() *config.SupportedKafkaInstanceTypesConfig
	}

	tests := []struct {
		name    string
		fields  fields
		want    instanceTypeConsumptionSummary
		wantErr bool
	}{
		{
			name: "Calculate correctly computes the result",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "Cluster information that does not match the provided locator's region is ignored",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					clusterInfoForAnotherLocator := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: "p1",
						Region:        "r2",
						InstanceType:  "t1",
						Count:         4,
						MaxUnits:      6,
						Status:        api.ClusterAccepted.String(),
					}

					res = append(res, clusterInfoForAnotherLocator)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "Cluster information that does not match the provided locator's provider is ignored",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					clusterInfoForAnotherLocator := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: "p2",
						Region:        "r1",
						InstanceType:  "t1",
						Count:         4,
						MaxUnits:      6,
						Status:        api.ClusterAccepted.String(),
					}

					res = append(res, clusterInfoForAnotherLocator)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "Cluster information that does not match the provided locator's instance type name is ignored",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					clusterInfoForAnotherLocator := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: "p1",
						Region:        "r1",
						InstanceType:  "t2",
						Count:         4,
						MaxUnits:      6,
						Status:        api.ClusterAccepted.String(),
					}

					res = append(res, clusterInfoForAnotherLocator)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in accepted state ongoing scale up is true",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      0,
						Status:        api.ClusterAccepted.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 true,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in waiting for kasfleetshard state ongoing scale up is true",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      0,
						Status:        api.ClusterWaitingForKasFleetShardOperator.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 true,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in provisioning state ongoing scale up is true",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      0,
						Status:        api.ClusterProvisioning.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 true,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in provisioned state ongoing scale up is true",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingScaledInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      0,
						Status:        api.ClusterProvisioned.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingScaledInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 true,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When there is no cluster where the biggest size of the locator's instance type can fit biggestInstanceSizeCapacityAvailable is false",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					return newTestHelperBaseKafkaStreamingUnitCountPerClusterList()
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return &config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							config.KafkaInstanceType{
								Id: "t1",
								Sizes: []config.KafkaInstanceSize{
									config.KafkaInstanceSize{
										Id:               "s1",
										CapacityConsumed: 1,
									},
									config.KafkaInstanceSize{
										Id:               "s2",
										CapacityConsumed: 2,
									},
									config.KafkaInstanceSize{
										Id:               "s3",
										CapacityConsumed: 4,
									},
								},
							},
						},
					}

				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: false,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in deprovisioning state its max units are not taken into account",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingDeprovisionedInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      30,
						Status:        api.ClusterDeprovisioning.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingDeprovisionedInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When one of the clusters that match the locator is in cleanup state its max units are not taken into account",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingDeprovisionedInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      30,
						Status:        api.ClusterCleanup.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingDeprovisionedInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    8,
				freeStreamingUnits:                   3,
				consumedStreamingUnits:               5,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: true,
			},
			wantErr: false,
		},
		{
			name: "When the provided locator's instance type is not found in the supported providers configuration an error is returned",
			fields: fields{
				locator: supportedInstanceTypeLocator{
					provider:         "p1",
					region:           "r1",
					instanceTypeName: "unexistinginstancetype",
				},
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					res := []services.KafkaStreamingUnitCountPerCluster(newTestHelperBaseKafkaStreamingUnitCountPerClusterList())
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					clusterBeingDeprovisionedInfo := services.KafkaStreamingUnitCountPerCluster{
						CloudProvider: locator.provider,
						Region:        locator.region,
						InstanceType:  locator.instanceTypeName,
						Count:         0,
						MaxUnits:      30,
						Status:        api.ClusterCleanup.String(),
						ClusterType:   locator.clusterType,
					}
					res = append(res, clusterBeingDeprovisionedInfo)
					return res
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			wantErr: true,
		},
		{
			name: "When the clusters are in deprovision or cleanup state, we do not consider them when evaluating for the availability of biggest instance size capacity",
			fields: fields{
				locator: newTestHelperBaseSupportedInstanceTypeLocator(),
				kafkaStreamingUnitCountPerClusterListFactory: func() services.KafkaStreamingUnitCountPerClusterList {
					locator := newTestHelperBaseSupportedInstanceTypeLocator()
					return services.KafkaStreamingUnitCountPerClusterList{
						services.KafkaStreamingUnitCountPerCluster{
							CloudProvider: locator.provider,
							Region:        locator.region,
							InstanceType:  locator.instanceTypeName,
							Count:         0,
							MaxUnits:      30,
							Status:        api.ClusterCleanup.String(),
							ClusterType:   locator.clusterType,
						},
						services.KafkaStreamingUnitCountPerCluster{
							CloudProvider: locator.provider,
							Region:        locator.region,
							InstanceType:  locator.instanceTypeName,
							Count:         0,
							MaxUnits:      30,
							Status:        api.ClusterDeprovisioning.String(),
							ClusterType:   locator.clusterType,
						},
					}
				},
				supportedKafkaInstanceTypesConfigFactory: func() *config.SupportedKafkaInstanceTypesConfig {
					return newTestHelperBaseSupportedKafkaInstanceTypesConfig()
				},
			},
			want: instanceTypeConsumptionSummary{
				maxStreamingUnits:                    0,
				freeStreamingUnits:                   0,
				consumedStreamingUnits:               0,
				ongoingScaleUpAction:                 false,
				biggestInstanceSizeCapacityAvailable: false,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			summaryCalculator := instanceTypeConsumptionSummaryCalculator{
				locator:                               tt.fields.locator,
				kafkaStreamingUnitCountPerClusterList: tt.fields.kafkaStreamingUnitCountPerClusterListFactory(),
				supportedKafkaInstanceTypesConfig:     tt.fields.supportedKafkaInstanceTypesConfigFactory(),
			}
			res, err := summaryCalculator.Calculate()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
