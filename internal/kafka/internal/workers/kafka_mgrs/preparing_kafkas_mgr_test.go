package kafka_mgrs

import (
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestPreparingKafkaManager(t *testing.T) {
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
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(30)),
					},
					Status: string(constants2.KafkaRequestStatusPreparing),
				},
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
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(-30)),
					},
					Status: string(constants2.KafkaRequestStatusPreparing),
				},
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
				kafka: &dbapi.KafkaRequest{
					Status: string(constants2.KafkaRequestStatusPreparing),
				},
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
				kafka: &dbapi.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(30))}},
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
				kafka: &dbapi.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &PreparingKafkaManager{
				kafkaService: tt.fields.kafkaService,
			}

			if err := k.reconcilePreparingKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparingKafka() error = %v, wantErr %v", err, tt.wantErr)
			}

			gomega.Expect(tt.expectedKafkaStatus.String()).Should(gomega.Equal(tt.args.kafka.Status))
			gomega.Expect(tt.args.kafka.FailedReason).Should(gomega.Equal(tt.wantErrMsg))
		})
	}
}
