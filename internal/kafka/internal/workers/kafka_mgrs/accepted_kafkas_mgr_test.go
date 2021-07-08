package kafka_mgrs

import (
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"testing"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestAcceptedKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService        services.KafkaService
		clusterPlmtStrategy services.ClusterPlacementStrategy
		quotaService        services.QuotaService
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantStatus string
	}{
		{
			name: "error when finding cluster fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "some-subscription", nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantErr:    true,
			wantStatus: constants2.KafkaRequestStatusPreparing.String(),
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{}, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "sub-scription", nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{},
			},
			wantStatus: constants2.KafkaRequestStatusPreparing.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &AcceptedKafkaManager{
				kafkaService:        tt.fields.kafkaService,
				clusterPlmtStrategy: tt.fields.clusterPlmtStrategy,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}
			err := k.reconcileAcceptedKafka(tt.args.kafka)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(tt.args.kafka.Status).To(gomega.Equal(tt.wantStatus))
		})
	}
}
