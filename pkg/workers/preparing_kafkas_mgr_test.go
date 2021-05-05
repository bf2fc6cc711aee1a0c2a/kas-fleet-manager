package workers

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
		name       string
		fields     fields
		args       args
		wantErrMsg string
		wantErr    bool
	}{
		{
			name: "Encounter an SSO Client internal error in Kafka creation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
			},
			wantErr:    true,
			wantErrMsg: "ErrorFailedToCreateSSOClientReason",
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
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
			name: "Encounter an Client error (4xx) in Kafka creation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.NotFound("simulate an 4xx error on purpose")
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
			},
			wantErr:    true,
			wantErrMsg: "simulate an 4xx error on purpose",
		},
		{
			name: "Successfully reconciled the prepared Kafka instance",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
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
			if err := k.reconcilePreparedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
			gomega.Expect(tt.args.kafka.FailedReason).Should(gomega.Equal(tt.wantErrMsg))

		})
	}
}
