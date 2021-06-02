package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestAcceptedKafkaManager(t *testing.T) {
	type fields struct {
		kafkaService        services.KafkaService
		configService       services.ConfigService
		clusterPlmtStrategy services.ClusterPlacementStrategy
		quotaService        services.QuotaService
	}
	type args struct {
		kafka *api.KafkaRequest
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
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return nil, errors.GeneralError("test")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *api.KafkaRequest) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *api.KafkaRequest) (string, *errors.ServiceError) {
						return "some-subscription", nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr:    true,
			wantStatus: constants.KafkaRequestStatusPreparing.String(),
		},
		{
			name: "set kafka status to failed when quota is insufficient",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: &config.KafkaConfig{
						Quota: config.NewKafkaQuotaConfig(),
					},
				}),
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *api.KafkaRequest) (string, *errors.ServiceError) {
						return "", errors.InsufficientQuotaError("quota insufficient")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantStatus: constants.KafkaRequestStatusFailed.String(),
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: &config.KafkaConfig{
						Quota: config.NewKafkaQuotaConfig(),
					},
				}),
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *api.KafkaRequest) (string, *errors.ServiceError) {
						return "sub-scription", nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantStatus: constants.KafkaRequestStatusPreparing.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &AcceptedKafkaManager{
				kafkaService:        tt.fields.kafkaService,
				configService:       tt.fields.configService,
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
