package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mockServiceAccounts "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/service_accounts"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
)

var (
	enabledAuthKeycloakConfig = &keycloak.KeycloakConfig{
		EnableAuthenticationOnKafka: true,
	}
)

func TestReadyKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService    services.KafkaService
		keycloakService sso.KeycloakService
		keycloakConfig  *keycloak.KeycloakConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should skip reconciliation without error if EnableAuthenticationOnKafka is not set in the keycloak config",
			fields: fields{
				keycloakConfig: &keycloak.KeycloakConfig{},
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if listing kafkas fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to list kafka requests")
					},
				},
				keycloakConfig: enabledAuthKeycloakConfig,
			},
			wantErr: true,
		},
		{
			name: "Should succeed if no kafkas are returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{}, nil
					},
				},
				keycloakConfig: enabledAuthKeycloakConfig,
			},
			wantErr: false,
		},
		{
			name: "Should throw an error if reconciling canary service account fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetKafkaClientSecretFunc: func(clientId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to create service account")
					},
				},
				keycloakConfig: enabledAuthKeycloakConfig,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := NewReadyKafkaManager(tt.fields.kafkaService, tt.fields.keycloakService, tt.fields.keycloakConfig, w.Reconciler{})

			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestReadyKafkaManager_reconcileCanaryServiceAccount(t *testing.T) {
	type fields struct {
		kafkaService    services.KafkaService
		keycloakService sso.KeycloakService
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
			name: "Successful reconcile kafka that have canary service account already created",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: nil, // set to nil as it should not be called
				},
				keycloakService: &sso.KeycloakServiceMock{
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
				keycloakService: &sso.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
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
			name: "Successful reconcile execution",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return mockServiceAccounts.BuildApiServiceAccount(nil), nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr: false,
		},
		{
			name: "Should fail if kafka update fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("kafka update failed")
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return mockServiceAccounts.BuildApiServiceAccount(nil), nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &ReadyKafkaManager{
				kafkaService:    tt.fields.kafkaService,
				keycloakService: tt.fields.keycloakService,
			}

			g.Expect(k.reconcileCanaryServiceAccount(tt.args.kafka) != nil).To(gomega.Equal(tt.wantErr))

			if !tt.wantErr {
				g.Expect(tt.args.kafka.CanaryServiceAccountClientID).NotTo(gomega.BeEmpty())
				g.Expect(tt.args.kafka.CanaryServiceAccountClientSecret).NotTo(gomega.BeEmpty())
			}
		})
	}
}
