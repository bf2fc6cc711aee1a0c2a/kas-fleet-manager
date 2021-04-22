package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"reflect"
	"testing"
)

func TestFirstReadyCluster_FindCluster(t *testing.T) {
	type fields struct {
		ConfigService  ConfigService
		ClusterService ClusterService
	}
	type args struct {
		kafka *api.KafkaRequest
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
				ConfigService: NewConfigService(config.ApplicationConfig{
					Kafka:            config.NewKafkaConfig(),
					OSDClusterConfig: config.NewOSDClusterConfig(),
				}),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return &api.Cluster{}, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    &api.Cluster{},
			wantErr: false,
		},
		{
			name: "Cannot find ready cluster",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					Kafka:            config.NewKafkaConfig(),
					OSDClusterConfig: config.NewOSDClusterConfig(),
				}),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "find ready cluster with error",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					Kafka:            config.NewKafkaConfig(),
					OSDClusterConfig: config.NewOSDClusterConfig(),
				}),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.NotFound("not found")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FirstReadyCluster{
				ConfigService:  tt.fields.ConfigService,
				ClusterService: tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.kafka)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindReadyCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindReadyCluster() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFirstScheduleWithinLimit_FindCluster(t *testing.T) {
	type fields struct {
		ConfigService  ConfigService
		ClusterService ClusterService
	}
	type args struct {
		kafka *api.KafkaRequest
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
				ConfigService: NewConfigService(config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
						ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 3}}),
					},
				}),
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    &api.Cluster{ClusterID: "test01"},
			wantErr: false,
		},
		{
			name: "Failed to find an available schedulable cluster as exceeds limit",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
						ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 1}}),
					},
				}),
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Find an available schedulable cluster after one exceeds limit",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
						ClusterConfig: config.NewClusterConfig(config.ClusterList{
							config.ManualCluster{ClusterId: "test01", Schedulable: true, KafkaInstanceLimit: 1},
							config.ManualCluster{ClusterId: "test02", Schedulable: true, KafkaInstanceLimit: 3}})},
				}),
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						res = append(res, &api.Cluster{ClusterID: "test02"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
						var res2 []ResKafkaInstanceCount
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test01", Count: 1})
						res2 = append(res2, ResKafkaInstanceCount{Clusterid: "test02", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    &api.Cluster{ClusterID: "test02"},
			wantErr: false,
		},
		{
			name: "Failed to find an available cluster as non is schedulable",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
						ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: false, KafkaInstanceLimit: 1}}),
					},
				}),
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Failed to find an available cluster due to error",
			fields: fields{
				ConfigService: NewConfigService(config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
					},
				}),
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.NotFound("not found")
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FirstSchedulableWithinLimit{
				ConfigService:  tt.fields.ConfigService,
				ClusterService: tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.kafka)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAvailableCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindAvailableCluster() got = %v, want %v", got, tt.want)
			}
		})
	}
}
