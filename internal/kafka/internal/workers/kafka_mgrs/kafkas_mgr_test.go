package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	dpMock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
)

func TestKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService            services.KafkaService
		dataplaneClusterConfig  config.DataplaneClusterConfig
		cloudProviders          config.ProviderConfig
		accessControlListConfig *acl.AccessControlListConfig
		kafkaConfig             config.KafkaConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if setKafkaStatusCountMetric returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return nil, errors.GeneralError("failed to count kafkas by status")
					},
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig:             *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if setClusterStatusCapacityMetrics returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return nil, errors.GeneralError("failed to count kafkas by region and instance type")
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig:             *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if DeprovisionExpiredKafkas returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return errors.GeneralError("failed to deprovision expired kafkas")
					},
				},
				dataplaneClusterConfig: *config.NewDataplaneClusterConfig(),
				accessControlListConfig: &acl.AccessControlListConfig{
					EnableDenyList: true,
				},
				kafkaConfig: *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:            tt.fields.kafkaService,
				dataplaneClusterConfig:  &tt.fields.dataplaneClusterConfig,
				accessControlListConfig: tt.fields.accessControlListConfig,
				cloudProviders:          &tt.fields.cloudProviders,
				kafkaConfig:             &tt.fields.kafkaConfig,
			}

			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_reconcileDeniedKafkaOwners(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		deniedAccounts acl.DeniedUsers
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "do not reconcile when denied accounts list is empty",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil, // set to nil as it should not be called
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{},
			},
			wantErr: false,
		},
		{
			name: "should receive error when update in deprovisioning in database returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when update in deprovisioning in database succeed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService: tt.fields.kafkaService,
			}
			g.Expect(k.reconcileDeniedKafkaOwners(tt.args.deniedAccounts) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_setKafkaStatusCountMetric(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if CountByStatus fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return nil, errors.GeneralError("failed to count kafkas by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully set kafka status count metrics",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{
							{
								Status: constants.KafkaRequestStatusAccepted,
								Count:  2,
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := NewKafkaManager(tt.fields.kafkaService, nil, nil, nil, nil, workers.Reconciler{})

			g.Expect(k.setKafkaStatusCountMetric() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_setClusterStatusCapacityMetrics(t *testing.T) {
	type fields struct {
		kafkaService           services.KafkaService
		dataplaneClusterConfig config.DataplaneClusterConfig
		cloudProviders         config.ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if CountStreamingUnitByRegionAndInstanceType fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return nil, errors.GeneralError("failed to count kafkas")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if calculateCapacityByRegionAndInstanceTypeForManualClusters fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("invalid"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
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
			name: "should return an error if calculateAvailableAndMaxCapacityForDynamicScaling fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
								MaxUnits:      10,
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:                   "us-east-1",
										Default:                true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{},
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
			name: "should successfully assign metrics for manual clusters",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard,developer"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should successfully assign metrics for autoscaling mode",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountStreamingUnitByRegionAndInstanceTypeFunc: func() ([]services.KafkaStreamingUnitCountPerRegion, error) {
						return []services.KafkaStreamingUnitCountPerRegion{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			g.Expect(k.setClusterStatusCapacityMetrics() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

var (
	cloudProviderStandardLimit = 5
)

func TestKafkaManager_calculateCapacityByRegionAndInstanceTypeForManualClusters(t *testing.T) {
	type fields struct {
		kafkaService                 services.KafkaService
		dataplaneClusterConfig       config.DataplaneClusterConfig
		cloudProviders               config.ProviderConfig
		streamingUnitsCountPerRegion []services.KafkaStreamingUnitCountPerRegion
	}

	type counts struct {
		available int32
		max       int32
	}

	tests := []struct {
		name     string
		fields   fields
		expected map[string]map[string]counts
	}{
		{
			name: "expected available capacity with no cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: []services.KafkaStreamingUnitCountPerRegion{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         5,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used, 3 developer instances used
			// ==> expect 5 more standard instances to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 5,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with no cloud provider limit and existing developer instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard,developer"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: nil},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: []services.KafkaStreamingUnitCountPerRegion{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         5,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "cluster-id",
						Count:         3,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used
			// 3 developer instances used
			// ==> expect 2 more standard instances to be available
			// ==> expect 2 more developer instances to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 2,
						max:       10,
					},
					"developer": counts{
						available: 2,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: []services.KafkaStreamingUnitCountPerRegion{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// ==> expect 1 more standard instance to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       5,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing mixed instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("developer,standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: []services.KafkaStreamingUnitCountPerRegion{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// 4 developer instances used
			// ==> expect 1 more standard instance to be available
			// ==> expect 2 more developer instance to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       5,
					},
					"developer": counts{
						available: 2,
						max:       10,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			results, err := k.calculateCapacityByRegionAndInstanceTypeForManualClusters(tt.fields.streamingUnitsCountPerRegion)
			g.Expect(err).To(gomega.BeNil())

			for _, result := range results {
				count := tt.expected[result.Region][result.InstanceType]
				g.Expect(result.Count).To(gomega.Equal(count.available))
				g.Expect(result.MaxUnits).To(gomega.Equal(count.max))
			}
		})
	}
}

func TestKafkaManager_calculateAvailableAndMaxCapacityForDynamicScaling(t *testing.T) {
	type fields struct {
		kafkaService                services.KafkaService
		dataplaneClusterConfig      config.DataplaneClusterConfig
		cloudProviders              config.ProviderConfig
		streamingUnitCountPerRegion services.KafkaStreamingUnitCountPerRegion
	}

	type counts struct {
		available int32
		max       int32
	}

	tests := []struct {
		name     string
		fields   fields
		expected map[string]map[string]counts
	}{
		{
			name: "expected available capacity with no cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitCountPerRegion: services.KafkaStreamingUnitCountPerRegion{

					Region:        "us-east-1",
					InstanceType:  "standard",
					ClusterId:     "cluster-id",
					Count:         9,
					MaxUnits:      10,
					CloudProvider: "aws",
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 9 instances used,
			// ==> expect 1 available and max 10
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitCountPerRegion: services.KafkaStreamingUnitCountPerRegion{

					Region:        "us-east-1",
					InstanceType:  "standard",
					ClusterId:     "cluster-id",
					Count:         5,
					MaxUnits:      9,
					CloudProvider: "aws",
				},
			},
			// 9 total capacity
			// 5 cloud provider limits
			// 5 instances used,
			// ==> expect 0 available and max 5
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 0,
						max:       5,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			result, err := k.calculateAvailableAndMaxCapacityForDynamicScaling(tt.fields.streamingUnitCountPerRegion)
			g.Expect(err).To(gomega.BeNil())

			count := tt.expected[result.Region][result.InstanceType]
			g.Expect(result.Count).To(gomega.Equal(count.available))
			g.Expect(result.MaxUnits).To(gomega.Equal(count.max))
		})
	}
}
