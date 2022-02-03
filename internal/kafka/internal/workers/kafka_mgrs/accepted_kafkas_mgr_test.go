package kafka_mgrs

import (
	"encoding/json"
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	"github.com/onsi/gomega"
)

func TestAcceptedKafkaManager(t *testing.T) {
	testConfig := config.NewDataplaneClusterConfig()

	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"
	availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
				{
					Version: "2.8.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
				{
					Version: "2.8",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	noAvailableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   false,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
				{
					Version: "2.8.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
				{
					Version: "2.8",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	mockCluster := &api.Cluster{
		Meta: api.Meta{
			ID:        "id",
			CreatedAt: time.Now(),
		},
		ClusterID:                "cluster-id",
		MultiAZ:                  true,
		Region:                   "us-east-1",
		Status:                   "ready",
		AvailableStrimziVersions: availableStrimziVersions,
	}

	mockClusterWithoutAvailableStrimziVersion := *mockCluster
	mockClusterWithoutAvailableStrimziVersion.AvailableStrimziVersions = noAvailableStrimziVersions

	type fields struct {
		kafkaService           services.KafkaService
		quotaService           services.QuotaService
		dataPlaneClusterConfig *config.DataplaneClusterConfig
		clusterService         services.ClusterService
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name                       string
		fields                     fields
		args                       args
		wantErr                    bool
		wantStatus                 string
		wantStrimziOperatorVersion string
	}{
		{
			name: "should return an error when finding cluster fails",
			fields: fields{
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "", nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
					ClusterID: mockCluster.ClusterID,
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error when kafka service update fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockCluster, nil
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
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
					ClusterID: mockCluster.ClusterID,
				},
			},
			wantErr:                    true,
			wantStatus:                 constants2.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: strimziOperatorVersion,
		},
		{
			name: "should get desired strimzi version from cluster if the StrimziOperatorVersion is not set in the data plane config",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockCluster, nil
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
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
					ClusterID: mockCluster.ClusterID,
				},
			},
			wantErr:                    false,
			wantStatus:                 constants2.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: strimziOperatorVersion,
		},
		{
			name: "should keep kafka status as accepted if no strimzi operator version is available when retry period has not expired",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &mockClusterWithoutAvailableStrimziVersion, nil
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
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
					ClusterID: mockCluster.ClusterID,
				},
			},
			wantErr: false,
		},
		{
			name: "should set kafka status to failed if no strimzi operator version is available after retry period has expired",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &mockClusterWithoutAvailableStrimziVersion, nil
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
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Duration(-constants2.AcceptedKafkaMaxRetryDuration)),
					},
					ClusterID: mockCluster.ClusterID,
				},
			},
			wantErr:    true,
			wantStatus: constants2.KafkaRequestStatusFailed.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &AcceptedKafkaManager{
				kafkaService:   tt.fields.kafkaService,
				clusterService: tt.fields.clusterService,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
				dataPlaneClusterConfig: tt.fields.dataPlaneClusterConfig,
			}
			err := k.reconcileAcceptedKafka(tt.args.kafka)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(tt.args.kafka.Status).To(gomega.Equal(tt.wantStatus))
			gomega.Expect(tt.args.kafka.DesiredStrimziVersion).To(gomega.Equal(tt.wantStrimziOperatorVersion))
			gomega.Expect(tt.args.kafka.ClusterID).ToNot(gomega.Equal(""))
		})
	}
}
