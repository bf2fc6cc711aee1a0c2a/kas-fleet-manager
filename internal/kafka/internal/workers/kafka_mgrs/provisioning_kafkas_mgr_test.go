package kafka_mgrs

import (
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/onsi/gomega"
)

func TestProvisioningKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should throw an error if listing kafkas fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to list kafka requests")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if listing kafkas returns an empty list",
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
			name: "Should not throw an error when updating metrics for returned kafkas list",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusProvisioning.String()),
							),
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(len(NewProvisioningKafkaManager(tt.fields.kafkaService, w.Reconciler{}).Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}
