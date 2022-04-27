package services

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	. "github.com/onsi/gomega"
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
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
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
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
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
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.NotFound("not found")
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FirstReadyCluster{
				ClusterService: tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.kafka)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindReadyCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			Expect(got).To(Equal(tt.want))
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
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
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
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.NotFound("not found")
					},
					FindKafkaInstanceCountFunc: func(clusterIds []string) (res []ResKafkaInstanceCount, error *errors.ServiceError) {
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			Expect(got).To(Equal(tt.want))
		})
	}
}
