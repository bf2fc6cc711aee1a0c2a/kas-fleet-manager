package services

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	mockkafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func TestFirstReadyCluster_FindCluster(t *testing.T) {
	type fields struct {
		ClusterService         ClusterService
		Kafka                  *config.KafkaConfig
		DataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
	}{
		{
			name: "Find ready cluster",
			fields: fields{
				Kafka:                  config.NewKafkaConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			want:    &api.Cluster{},
			wantErr: false,
		},
		{
			name: "Cannot find ready cluster",
			fields: fields{
				Kafka:                  config.NewKafkaConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "find ready cluster with error",
			fields: fields{
				Kafka:                  config.NewKafkaConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (*api.Cluster, error) {
						return nil, errors.New("not found")
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := &FirstReadyCluster{
				ClusterService: tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.kafka)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindReadyCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestFirstScheduleWithinLimit_FindCluster(t *testing.T) {
	type fields struct {
		DataplaneClusterConfig *config.DataplaneClusterConfig
		ClusterService         ClusterService
		kafkaConfig            *config.KafkaConfig
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
	}{
		{
			name: "find an available schedule cluster and within limit",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 3}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) ([]ResKafkaInstanceCount, error) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					SizeId:       "x1",
					InstanceType: types.STANDARD.String(),
				},
			},
			want:    &api.Cluster{ClusterID: "test01"},
			wantErr: false,
		},
		{
			name: "Failed to find an available schedulable cluster as exceeds limit",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 1}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) ([]ResKafkaInstanceCount, error) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					SizeId:       "x1",
					InstanceType: types.STANDARD.String(),
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Find an available schedulable cluster after one exceeds limit",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig: config.NewClusterConfig(config.ClusterList{
						config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 1},
						config.ManualCluster{ClusterId: "test02", Schedulable: true, KafkaInstanceLimit: 3}})},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						res = append(res, &api.Cluster{ClusterID: "test02"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) ([]ResKafkaInstanceCount, error) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test02", Count: 1})
						return res2, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					SizeId:       "x1",
					InstanceType: types.STANDARD.String(),
				},
			},
			want:    &api.Cluster{ClusterID: "test02"},
			wantErr: false,
		},
		{
			name: "Failed to find an available cluster as non is schedulable",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: false, KafkaInstanceLimit: 1}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) ([]ResKafkaInstanceCount, error) {
						return nil, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType: types.STANDARD.String(),
					SizeId:       "x1",
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Failed to find an available cluster due to error",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return nil, errors.New("not found")
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) ([]ResKafkaInstanceCount, error) {
						return nil, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType: types.STANDARD.String(),
					SizeId:       "x1",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := &FirstSchedulableWithinLimit{
				DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
				ClusterService:         tt.fields.ClusterService,
				KafkaConfig:            tt.fields.kafkaConfig,
			}
			got, err := f.FindCluster(tt.args.kafka)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAvailableCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestFirstReadyWithCapacity_FindCluster(t *testing.T) {
	type fields struct {
		ClusterService ClusterService
		KafkaConfig    *config.KafkaConfig
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr error
	}{
		{
			name: "should return an error if getting clusters that matches the given criteria fails",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return nil, errors.New("failed to find clusters")
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(),
			},
			want: nil,
			wantErr: errors.Wrapf(errors.New("failed to find clusters"), fmt.Sprintf("failed to find all clusters with criteria '%v'", FindClusterCriteria{
				MultiAZ: mockkafkas.BuildKafkaRequest().MultiAZ,
				Status:  api.ClusterReady,
			})),
		},
		{
			name: "should return an error if getting streaming unit count per region and instance type fails",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return []*api.Cluster{
							{
								ClusterID:           mockkafkas.DefaultClusterID,
								DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
							},
						}, nil
					},
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (KafkaStreamingUnitCountPerClusterList, error) {
						return KafkaStreamingUnitCountPerClusterList{}, errors.New("failed to retrieve streaming unit count per region and instance type")
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(),
			},
			want: nil,
			wantErr: errors.Wrapf(errors.New("failed to retrieve streaming unit count per region and instance type"), fmt.Sprintf("failed to get count of streaming units by cluster and instance type for criteria '%v'", FindClusterCriteria{
				MultiAZ: mockkafkas.BuildKafkaRequest().MultiAZ,
				Status:  api.ClusterReady,
			})),
		},
		{
			name: "should return an error if getting the requested kafka instance size fails",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return []*api.Cluster{
							{
								ClusterID:           mockkafkas.DefaultClusterID,
								DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
							},
						}, nil
					},
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (KafkaStreamingUnitCountPerClusterList, error) {
						return KafkaStreamingUnitCountPerClusterList{
							{
								ClusterId:    mockkafkas.DefaultClusterID,
								InstanceType: types.STANDARD.String(),
								Count:        0,
							},
						}, nil
					},
				},
				KafkaConfig: config.NewKafkaConfig(),
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(
					mockkafkas.With(mockkafkas.ID, mockkafkas.DefaultKafkaID),
					mockkafkas.With(mockkafkas.INSTANCE_TYPE, "unsupported"),
					mockkafkas.With(mockkafkas.SIZE_ID, "unsupported"),
				),
			},
			want: nil,
			wantErr: errors.Wrapf(errors.New("unable to find kafka instance type for 'unsupported'"), fmt.Sprintf("failed to get kafka instance size for cluster with criteria '%v'", FindClusterCriteria{
				MultiAZ:               mockkafkas.BuildKafkaRequest().MultiAZ,
				Status:                api.ClusterReady,
				SupportedInstanceType: "unsupported",
			})),
		},
		{
			name: "should return nil if no clusters matches the given criteria",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(),
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "should return a cluster if it has remaining capacity",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return []*api.Cluster{
							{
								ClusterID:           mockkafkas.DefaultClusterID,
								ClusterType:         api.ManagedDataPlaneClusterType.String(),
								DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
							},
						}, nil
					},
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (KafkaStreamingUnitCountPerClusterList, error) {
						return KafkaStreamingUnitCountPerClusterList{
							{
								ClusterId:    mockkafkas.DefaultClusterID,
								ClusterType:  api.ManagedDataPlaneClusterType.String(),
								InstanceType: types.STANDARD.String(),
								Count:        0,
							},
						}, nil
					},
				},
				KafkaConfig: &config.KafkaConfig{
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
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(
					mockkafkas.With(mockkafkas.ID, mockkafkas.DefaultKafkaID),
					mockkafkas.With(mockkafkas.INSTANCE_TYPE, types.STANDARD.String()),
					mockkafkas.With(mockkafkas.SIZE_ID, "x1"),
				),
			},
			want: &api.Cluster{
				ClusterID:           mockkafkas.DefaultClusterID,
				ClusterType:         api.ManagedDataPlaneClusterType.String(),
				DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
			},
			wantErr: nil,
		},
		{
			name: "should return nil if cluster has no remaining capacity",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return []*api.Cluster{
							{
								ClusterID:           mockkafkas.DefaultClusterID,
								DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
							},
						}, nil
					},
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (KafkaStreamingUnitCountPerClusterList, error) {
						return KafkaStreamingUnitCountPerClusterList{
							{
								ClusterId:    mockkafkas.DefaultClusterID,
								InstanceType: types.STANDARD.String(),
								Count:        1,
							},
						}, nil
					},
				},
				KafkaConfig: &config.KafkaConfig{
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
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(
					mockkafkas.With(mockkafkas.ID, mockkafkas.DefaultKafkaID),
					mockkafkas.With(mockkafkas.INSTANCE_TYPE, types.STANDARD.String()),
					mockkafkas.With(mockkafkas.SIZE_ID, "x1"),
				),
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "should return nil if current and requested kafka goes over cluster capacity limit",
			fields: fields{
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, error) {
						return []*api.Cluster{
							{
								ClusterID:           mockkafkas.DefaultClusterID,
								DynamicCapacityInfo: api.JSON([]byte(`{"standard":{"max_nodes":1,"max_units":2,"remaining_units":1}}`)),
							},
						}, nil
					},
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (KafkaStreamingUnitCountPerClusterList, error) {
						return KafkaStreamingUnitCountPerClusterList{
							{
								ClusterId:    mockkafkas.DefaultClusterID,
								InstanceType: types.STANDARD.String(),
								Count:        1,
							},
						}, nil
					},
				},
				KafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: types.STANDARD.String(),
									Sizes: []config.KafkaInstanceSize{
										{
											Id:               "x1",
											CapacityConsumed: 2,
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: mockkafkas.BuildKafkaRequest(
					mockkafkas.With(mockkafkas.ID, mockkafkas.DefaultKafkaID),
					mockkafkas.With(mockkafkas.INSTANCE_TYPE, types.STANDARD.String()),
					mockkafkas.With(mockkafkas.SIZE_ID, "x1"),
				),
			},
			want:    nil,
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := &FirstReadyWithCapacity{
				ClusterService: tt.fields.ClusterService,
				KafkaConfig:    tt.fields.KafkaConfig,
			}

			got, err := f.FindCluster(tt.args.kafka)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr != nil))
			if tt.wantErr != nil {
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErr.Error()))
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
