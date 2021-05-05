package workers

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestProvisionedKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService         services.KafkaService
		observatoriumService services.ObservatoriumService
		configService        services.ConfigService
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
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.NotFound("Not Found")
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
			name: "error when updating kafka status fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.NotFound("Not Found")
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
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{State: "ready"}, nil
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
			k := &ProvisioningKafkaManager{
				kafkaService:         tt.fields.kafkaService,
				observatoriumService: tt.fields.observatoriumService,
				configService:        tt.fields.configService,
			}
			if err := k.reconcileProvisioningKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileProvisioningKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
