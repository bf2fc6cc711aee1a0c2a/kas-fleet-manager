package kafka_mgrs

import (
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"

	"github.com/onsi/gomega"

	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
)

func TestPreparingKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
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
			name: "Should successfully call reconcilePreparingKafka and return no error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusPreparing.String()),
							),
						}, nil
					},
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should call reconcilePreparingKafka and fail if an error is returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusPreparing.String()),
							),
						}, nil
					},
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("fail to prepare kafka request")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("fail to update kafka request")
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
			g.Expect(len(NewPreparingKafkaManager(tt.fields.kafkaService, w.Reconciler{}).Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestPreparingKafkaManager_reconcilePreparingKafkas(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		wantErr             bool
		wantErrMsg          string
		expectedKafkaStatus constants2.KafkaStatus
	}{
		{
			name: "Encounter a 5xx error Kafka preparation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.CreatedAt = time.Now().Add(time.Minute * time.Duration(30))
					kafkaRequest.Status = constants2.KafkaRequestStatusPreparing.String()
				}),
			},
			wantErr:             true,
			wantErrMsg:          "",
			expectedKafkaStatus: constants2.KafkaRequestStatusPreparing,
		},
		{
			name: "Encounter a 5xx error Kafka preparation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.CreatedAt = time.Now().Add(time.Minute * time.Duration(-30))
					kafkaRequest.Status = constants2.KafkaRequestStatusPreparing.String()
				}),
			},
			wantErr:             true,
			wantErrMsg:          "simulate 5xx error",
			expectedKafkaStatus: constants2.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter a Client error (4xx) in Kafka preparation",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.NotFound("simulate a 4xx error")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusPreparing.String()),
				),
			},
			wantErr:             true,
			wantErrMsg:          "simulate a 4xx error",
			expectedKafkaStatus: constants2.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.CreatedAt = time.Now().Add(time.Minute * time.Duration(30))
					kafkaRequest.Status = ""
				}),
			},
			wantErr:    true,
			wantErrMsg: "",
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.CreatedAt = time.Now().Add(time.Minute * time.Duration(-30))
					kafkaRequest.Status = ""
				}),
			},
			wantErr:             true,
			wantErrMsg:          "ErrorFailedToCreateSSOClientReason",
			expectedKafkaStatus: constants2.KafkaRequestStatusFailed,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &PreparingKafkaManager{
				kafkaService: tt.fields.kafkaService,
			}

			g.Expect(k.reconcilePreparingKafka(tt.args.kafka) != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(tt.expectedKafkaStatus.String()).Should(gomega.Equal(tt.args.kafka.Status))
			g.Expect(tt.args.kafka.FailedReason).Should(gomega.Equal(tt.wantErrMsg))
		})
	}
}
