package kafka_mgrs

import (
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/onsi/gomega"
)

func TestDeletingKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		quotaService   services.QuotaService
		keycloakConfig *keycloak.KeycloakConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should fail if listing kafkas in the reconciler fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("fail to list kafka requests")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should not fail if listing kafkas returns an empty list",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should call reconcileDeletingKafkas and fail if an error is returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.WithPredefinedTestValues(),
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusDeleting.String()),
							),
						}, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return errors.GeneralError("failed to delete quota")
					},
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username string, externalId string, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
				},
				keycloakConfig: &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
				},
			},
			wantErr: true,
		},
		{
			name: "Should call reconcileDeletingKafkas and not fail if no error is returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.WithPredefinedTestValues(),
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusDeleting.String()),
							),
							mockKafkas.BuildKafkaRequest(
								mockKafkas.WithPredefinedTestValues(),
							),
						}, nil
					},
					DeleteFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username string, externalId string, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
				},
				keycloakConfig: enabledAuthKeycloakConfig,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := NewDeletingKafkaManager(tt.fields.kafkaService,
				tt.fields.keycloakConfig,
				&services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
				w.Reconciler{})
			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestDeletingKafkaManager_reconcileDeletingKafkas(t *testing.T) {
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
			name: "should fail if deleting kafka fails",
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeleteFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("failed to delete kafka request")
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
		{
			name: "should fail if deleting quota fails",
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
						return errors.GeneralError("failed to delete quota")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &DeletingKafkaManager{
				kafkaService: tt.fields.kafkaService,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}
			g.Expect(k.reconcileDeletingKafkas(tt.args.kafka) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
