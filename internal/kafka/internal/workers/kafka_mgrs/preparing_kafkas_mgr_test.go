package kafka_mgrs

import (
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	constants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestPreparingKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		wantErr             bool
		wantErrMsg          string
		expectedKafkaStatus constants.KafkaStatus
	}{
		{
			name: "Encounter a 5xx error Kafka preparation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(30)),
					},
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "",
			expectedKafkaStatus: constants.KafkaRequestStatusPreparing,
		},
		{
			name: "Encounter a 5xx error Kafka preparation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(-30)),
					},
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "simulate 5xx error",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter a Client error (4xx) in Kafka preparation",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.NotFound("simulate a 4xx error")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "simulate a 4xx error",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(30))}},
			},
			wantErr:    true,
			wantErrMsg: "",
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
			},
			wantErr:             true,
			wantErrMsg:          "ErrorFailedToCreateSSOClientReason",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
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
