package workers

import (
	"testing"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	constants "gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

func TestKafkaManager_reconcileProvisionedKafka(t *testing.T) {
	type fields struct {
		ocmClient      ocm.Client
		clusterService services.ClusterService
		kafkaService   services.KafkaService
		timer          *time.Timer
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when creating kafka fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
					GetFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when updating kafka status fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) *errors.ServiceError {
						return errors.GeneralError("test")
					},
					GetFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka being provisioned does not exist in the DB before creation",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) *errors.ServiceError {
						return nil
					},
					GetFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, errors.NotFound("Not Found")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) *errors.ServiceError {
						return nil
					},
					GetFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:      tt.fields.ocmClient,
				clusterService: tt.fields.clusterService,
				kafkaService:   tt.fields.kafkaService,
				timer:          tt.fields.timer,
			}
			if err := k.reconcileProvisionedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileProvisionedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKafkaManager_reconcileAcceptedKafka(t *testing.T) {
	type fields struct {
		ocmClient      ocm.Client
		clusterService services.ClusterService
		kafkaService   services.KafkaService
		timer          *time.Timer
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when finding cluster fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka service update fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (cluster *api.Cluster, serviceError *errors.ServiceError) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:      tt.fields.ocmClient,
				clusterService: tt.fields.clusterService,
				kafkaService:   tt.fields.kafkaService,
				timer:          tt.fields.timer,
			}
			if err := k.reconcileAcceptedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileAcceptedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
