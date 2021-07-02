package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDeletingKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
		quotaService services.QuotaService
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful reconcile",
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeleteFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
		},
		{
			name: "failed reconcile",
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeleteFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &DeletingKafkaManager{
				kafkaService: tt.fields.kafkaService,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}
			if err := k.reconcileDeletingKafkas(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileDeletingKafkas() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
