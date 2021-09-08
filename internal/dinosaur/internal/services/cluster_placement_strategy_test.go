package services

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func TestFirstReadyCluster_FindCluster(t *testing.T) {
	type fields struct {
		ClusterService         ClusterService
		Dinosaur               *config.DinosaurConfig
		DataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		dinosaur *dbapi.DinosaurRequest
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
				Dinosaur:               config.NewDinosaurConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return &api.Cluster{}, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    &api.Cluster{},
			wantErr: false,
		},
		{
			name: "Cannot find ready cluster",
			fields: fields{
				Dinosaur:               config.NewDinosaurConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "find ready cluster with error",
			fields: fields{
				Dinosaur:               config.NewDinosaurConfig(),
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
				ClusterService: &ClusterServiceMock{
					FindClusterFunc: func(criteria FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.NotFound("not found")
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FirstReadyCluster{
				ClusterService: tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.dinosaur)
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
		DataplaneClusterConfig *config.DataplaneClusterConfig
		ClusterService         ClusterService
	}
	type args struct {
		dinosaur *dbapi.DinosaurRequest
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
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, DinosaurInstanceLimit: 3}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindDinosaurInstanceCountFunc: func(clusterIds []string) (res []ResDinosaurInstanceCount, error *errors.ServiceError) {
						var res2 []ResDinosaurInstanceCount
						res2 = append(res2, ResDinosaurInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    &api.Cluster{ClusterID: "test01"},
			wantErr: false,
		},
		{
			name: "Failed to find an available schedulable cluster as exceeds limit",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: true, DinosaurInstanceLimit: 1}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindDinosaurInstanceCountFunc: func(clusterIds []string) (res []ResDinosaurInstanceCount, error *errors.ServiceError) {
						var res2 []ResDinosaurInstanceCount
						res2 = append(res2, ResDinosaurInstanceCount{Clusterid: "test01", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
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
						config.ManualCluster{ClusterId: "test01", Schedulable: true, DinosaurInstanceLimit: 1},
						config.ManualCluster{ClusterId: "test02", Schedulable: true, DinosaurInstanceLimit: 3}})},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						res = append(res, &api.Cluster{ClusterID: "test02"})
						return res, nil
					},
					FindDinosaurInstanceCountFunc: func(clusterIds []string) (res []ResDinosaurInstanceCount, error *errors.ServiceError) {
						var res2 []ResDinosaurInstanceCount
						res2 = append(res2, ResDinosaurInstanceCount{Clusterid: "test01", Count: 1})
						res2 = append(res2, ResDinosaurInstanceCount{Clusterid: "test02", Count: 1})
						return res2, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    &api.Cluster{ClusterID: "test02"},
			wantErr: false,
		},
		{
			name: "Failed to find an available cluster as non is schedulable",
			fields: fields{
				DataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: "manual",
					ClusterConfig:               config.NewClusterConfig(config.ClusterList{config.ManualCluster{ClusterId: "test01", Schedulable: false, DinosaurInstanceLimit: 1}}),
				},
				ClusterService: &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) (cluster []*api.Cluster, serviceError *errors.ServiceError) {
						var res []*api.Cluster
						res = append(res, &api.Cluster{ClusterID: "test01"})
						return res, nil
					},
					FindDinosaurInstanceCountFunc: func(clusterIds []string) (res []ResDinosaurInstanceCount, error *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
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
					FindDinosaurInstanceCountFunc: func(clusterIds []string) (res []ResDinosaurInstanceCount, error *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FirstSchedulableWithinLimit{
				DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
				ClusterService:         tt.fields.ClusterService,
			}
			got, err := f.FindCluster(tt.args.dinosaur)
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
