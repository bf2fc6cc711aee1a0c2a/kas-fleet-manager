package kafka_mgrs

import (
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/onsi/gomega"

	mockClusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
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
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
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
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
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
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(mockKafkas.With(mocks.CLUSTER_ID, "some-cluster-id")),
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
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(
								mockKafkas.With(mockKafkas.STATUS, constants.KafkaRequestStatusAccepted.String()),
								mockKafkas.With(mocks.CLUSTER_ID, "some-cluster-id"),
							),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIDFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
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
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
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
		kafkaService             services.KafkaService
		clusterService           services.ClusterService
		clusterPlacementStrategy services.ClusterPlacementStrategy
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
		wantClusterID              string
	}{
		{
			name: "should return an error when finding cluster fails",
			fields: fields{
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
			wantErr:       true,
			wantClusterID: mockKafkas.DefaultClusterID,
		},
		{
			name: "shoud successful move the kafka onto preparing state",
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
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:                    false,
			wantStatus:                 constants.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: mockClusters.StrimziOperatorVersion,
			wantClusterID:              mockKafkas.DefaultClusterID,
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
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:                    true,
			wantStatus:                 constants.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: mockClusters.StrimziOperatorVersion,
			wantClusterID:              mockKafkas.DefaultClusterID,
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
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:                    false,
			wantStatus:                 constants.KafkaRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: mockClusters.StrimziOperatorVersion,
			wantClusterID:              mockKafkas.DefaultClusterID,
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
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now()),
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:       false,
			wantClusterID: mockKafkas.DefaultClusterID,
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
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now().Add(time.Duration(-constants.AcceptedKafkaMaxRetryDurationWhileWaitingForStrimziVersion))),
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:       false,
			wantStatus:    constants.KafkaRequestStatusFailed.String(),
			wantClusterID: mockKafkas.DefaultClusterID,
		},
		{
			name: "return an error when setting kafka status to failed if no strimzi operator version fails",
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
						return errors.GeneralError("error updating the status")
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now().Add(time.Duration(-constants.AcceptedKafkaMaxRetryDurationWhileWaitingForStrimziVersion))),
					mockKafkas.With(mockKafkas.CLUSTER_ID, mockKafkas.DefaultClusterID),
				),
			},
			wantErr:       true,
			wantStatus:    constants.KafkaRequestStatusFailed.String(),
			wantClusterID: mockKafkas.DefaultClusterID,
		},
		{
			name: "should return an error when cluster placement returns an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: nil, // make it as nil as it never be called
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, fmt.Errorf("some errors")
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: nil, // make it as nil as it never be called
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now()),
					mockKafkas.With(mockKafkas.CLUSTER_ID, ""), // make it empty to ensure cluster placement is called
				),
			},
			wantErr:       true,
			wantClusterID: "",
		},
		{
			name: "should keep kafka status as accepted if no available cluster found for assignement and one hour duration has not passed since kafka creation",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: nil, // make it as nil as it never be called
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: nil, // make it as nil as it never be called
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now()),
					mockKafkas.With(mockKafkas.CLUSTER_ID, ""), // make it empty to ensure cluster placement is called
				),
			},
			wantErr: false,
		},
		{
			name: "should assigns cluster id onto the kafka without one",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: nil, // make it as nil as it never be called
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockClusters.BuildCluster(func(cluster *api.Cluster) {
							cluster.ClusterID = "some-new-assigned-cluster-id"
						}), nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: nil, // make it as nil as it never be called
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now()),
					mockKafkas.With(mockKafkas.CLUSTER_ID, ""), // make it empty to ensure cluster placement is called
				),
			},
			wantErr:       false,
			wantClusterID: "some-new-assigned-cluster-id",
		},
		{
			name: "should mark the Kafka as failed after it has waited long without being assigned on a data plane cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: nil, // make it as nil as it never be called
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now().Add(time.Duration(-constants.AcceptedKafkaMaxRetryDurationWhileWaitingForClusterAssignment))), // assume that the kafka has been in waiting for more than an hour
					mockKafkas.With(mockKafkas.CLUSTER_ID, ""), // make it empty to ensure cluster placement is called
				),
			},
			wantErr:       false,
			wantClusterID: "",
			wantStatus:    constants.KafkaRequestStatusFailed.String(),
		},
		{
			name: "should return an error when marking the Kafka as failed returns an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: nil, // make it as nil as it never be called
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("some errors")
					},
				},
			},
			args: args{
				kafka: mockKafkas.BuildKafkaRequest(
					mockKafkas.WithCreatedAt(time.Now().Add(-61*time.Minute)), // assume that the kafka has been in waiting for more than an hour
					mockKafkas.With(mockKafkas.CLUSTER_ID, ""),                // make it empty to ensure cluster placement is called
				),
			},
			wantErr:       true,
			wantClusterID: "",
			wantStatus:    constants.KafkaRequestStatusFailed.String(),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &AcceptedKafkaManager{
				kafkaService:             tt.fields.kafkaService,
				clusterService:           tt.fields.clusterService,
				clusterPlacementStrategy: tt.fields.clusterPlacementStrategy,
				dataPlaneClusterConfig:   config.NewDataplaneClusterConfig(),
			}
			g.Expect(k.reconcileAcceptedKafka(tt.args.kafka) != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(tt.args.kafka.Status).To(gomega.Equal(tt.wantStatus))
			g.Expect(tt.args.kafka.DesiredStrimziVersion).To(gomega.Equal(tt.wantStrimziOperatorVersion))
			g.Expect(tt.args.kafka.ClusterID).To(gomega.Equal(tt.wantClusterID))
		})
	}
}
