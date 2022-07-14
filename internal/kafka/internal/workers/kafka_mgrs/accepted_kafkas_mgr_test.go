package kafka_mgrs

import (
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/onsi/gomega"

	mockClusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
)

func TestAcceptedKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		quotaService   services.QuotaService
		clusterService services.ClusterService
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
			name: "Should call reconcileAcceptedKafka and fail if an error is returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(),
						}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to find cluster")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should call reconcileAcceptedKafka and dont fail if an error is not returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.With(mockKafkas.STATUS, constants2.KafkaRequestStatusAccepted.String()),
							),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.AvailableStrimziVersions = mockClusters.AvailableStrimziVersions
						}), nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "sub-scription", nil
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
			k := NewAcceptedKafkaManager(
				tt.fields.kafkaService,
				&services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
				services.NewClusterPlacementStrategy(tt.fields.clusterService, config.NewDataplaneClusterConfig(), &config.KafkaConfig{}),
				config.NewDataplaneClusterConfig(),
				tt.fields.clusterService,
				w.Reconciler{})
			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestAcceptedKafkaManager_reconcileAcceptedKafka(t *testing.T) {

	type fields struct {
		kafkaService   services.KafkaService
		quotaService   services.QuotaService
		clusterService services.ClusterService
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
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr: true,
		},
		{
			name: "should return an error when kafka service update fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.AvailableStrimziVersions = mockClusters.AvailableStrimziVersions
						}), nil
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
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:                    true,
			wantStatus:                 constants2.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: mockClusters.StrimziOperatorVersion,
		},
		{
			name: "should get desired strimzi version from cluster if the StrimziOperatorVersion is not set in the data plane config",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.AvailableStrimziVersions = mockClusters.AvailableStrimziVersions
						}), nil
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
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:                    false,
			wantStatus:                 constants2.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: mockClusters.StrimziOperatorVersion,
		},
		{
			name: "should keep kafka status as accepted if no strimzi operator version is available when retry period has not expired",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.AvailableStrimziVersions = mockClusters.NoAvailableStrimziVersions
						}), nil
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
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now()),
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr: false,
		},
		{
			name: "should set kafka status to failed if no strimzi operator version is available after retry period has expired",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.AvailableStrimziVersions = mockClusters.NoAvailableStrimziVersions
						}), nil
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
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now().Add(time.Duration(-constants2.AcceptedKafkaMaxRetryDuration))),
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:    true,
			wantStatus: constants2.KafkaRequestStatusFailed.String(),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &AcceptedKafkaManager{
				kafkaService:   tt.fields.kafkaService,
				clusterService: tt.fields.clusterService,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			}
			g.Expect(k.reconcileAcceptedKafka(tt.args.kafka) != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(tt.args.kafka.Status).To(gomega.Equal(tt.wantStatus))
			g.Expect(tt.args.kafka.DesiredStrimziVersion).To(gomega.Equal(tt.wantStrimziOperatorVersion))
			g.Expect(tt.args.kafka.ClusterID).ToNot(gomega.Equal(""))
		})
	}
}
