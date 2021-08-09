package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestReadyKafkaManager_reconcileCanaryServiceAccount(t *testing.T) {
	serviceAccount := &api.ServiceAccount{
		ClientID:     "some-client-id",
		ClientSecret: "some-client-secret",
	}

	type fields struct {
		kafkaService    services.KafkaService
		keycloakService coreServices.KeycloakService
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
			name: "Successful reconcile kafka that have canary service account already created ",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: nil, // set to nil as it should not be called
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: nil, // set to nil as it should not be called,
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					CanaryServiceAccountClientID:     "some-client-id",
					CanaryServiceAccountClientSecret: "some-client-secret",
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when service account creation fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request coreServices.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, &errors.ServiceError{}
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request coreServices.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return serviceAccount, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &ReadyKafkaManager{
				kafkaService:    tt.fields.kafkaService,
				keycloakService: tt.fields.keycloakService,
			}

			if err := k.reconcileCanaryServiceAccount(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparingKafka() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				gomega.Expect(tt.args.kafka.CanaryServiceAccountClientID).NotTo(gomega.BeEmpty())
				gomega.Expect(tt.args.kafka.CanaryServiceAccountClientSecret).NotTo(gomega.BeEmpty())
			}
		})
	}
}
