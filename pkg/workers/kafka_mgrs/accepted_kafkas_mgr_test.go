package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestAcceptedKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService        services.KafkaService
		configService       services.ConfigService
		clusterPlmtStrategy services.ClusterPlacementStrategy
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
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return nil, errors.GeneralError("test")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &AcceptedKafkaManager{
				kafkaService:        tt.fields.kafkaService,
				configService:       tt.fields.configService,
				clusterPlmtStrategy: tt.fields.clusterPlmtStrategy,
			}
			if err := k.reconcileAcceptedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileAcceptedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
