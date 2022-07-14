package kafka_mgrs

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/onsi/gomega"

	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
)

func TestKafkaRoutesCNAMEManager_Reconcile(t *testing.T) {
	testChangeID := "1234"
	testChangeINSYNC := route53.ChangeStatusInsync

	type fields struct {
		kafkaService services.KafkaService
		kafkaConfig  *config.KafkaConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should succeed when there are no errors",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
							}),
						}, nil
					},
					ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
						return &route53.ChangeResourceRecordSetsOutput{
							ChangeInfo: &route53.ChangeInfo{
								Id:     &testChangeID,
								Status: &testChangeINSYNC,
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: false,
		},
		{
			name: "should succeed when there are no errors and RoutesCreationId is set for kafka",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
								kafkaRequest.RoutesCreationId = "test"
							}),
						}, nil
					},
					ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
						return &route53.ChangeResourceRecordSetsOutput{
							ChangeInfo: &route53.ChangeInfo{
								Id:     &testChangeID,
								Status: &testChangeINSYNC,
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetCNAMERecordStatusFunc: func(kafkaRequest *dbapi.KafkaRequest) (*services.CNameRecordStatus, error) {
						return &services.CNameRecordStatus{
							Status: &testChangeINSYNC,
						}, nil
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: false,
		},
		{
			name: "should fail when RoutesCreationId is set for kafka and GetCNAMERecordStatus fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
								kafkaRequest.RoutesCreationId = "test"
							}),
						}, nil
					},
					ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
						return &route53.ChangeResourceRecordSetsOutput{
							ChangeInfo: &route53.ChangeInfo{
								Id:     &testChangeID,
								Status: &testChangeINSYNC,
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetCNAMERecordStatusFunc: func(kafkaRequest *dbapi.KafkaRequest) (*services.CNameRecordStatus, error) {
						return nil, errors.GeneralError("failed to get cname record status")
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: true,
		},
		{
			name: "should skip route creation if external certs are disabled",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
							}),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: false},
			},
			wantErr: false,
		},
		{
			name: "should fail if kafka update fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
							}),
						}, nil
					},
					ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
						return &route53.ChangeResourceRecordSetsOutput{
							ChangeInfo: &route53.ChangeInfo{
								Id:     &testChangeID,
								Status: &testChangeINSYNC,
							},
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("failed to list kafkas")
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: true,
		},
		{
			name: "should return error when list kafkas failed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to list kafkas")
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: true,
		},
		{
			name: "should return error when creating CNAME failed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.RoutesCreated = false
							}),
						}, nil
					},
					ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to create CNAME")
					},
				},
				kafkaConfig: &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(len(NewKafkaCNAMEManager(test.fields.kafkaService,
				test.fields.kafkaConfig, w.Reconciler{}).Reconcile()) > 0).To(gomega.Equal(test.wantErr))
		})
	}
}
