package services

import (
	"context"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	. "github.com/onsi/gomega"
)

func Test_DataPlaneCluster_UpdateDataPlaneClusterStatus(t *testing.T) {
	testClusterID := "test-cluster-id"
	tests := []struct {
		name                           string
		clusterID                      string
		clusterStatus                  *dbapi.DataPlaneClusterStatus
		dataPlaneClusterServiceFactory func() *dataPlaneClusterService
		wantErr                        bool
	}{
		{
			name:          "An error is returned when a non-existent ClusterID is passed",
			clusterID:     testClusterID,
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
			wantErr: true,
		},
		{
			name:      "It succeeds when there are no issues",
			clusterID: testClusterID,
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{
					{
						Type:   "Ready",
						Status: "True",
					},
				},
				NodeInfo: dbapi.DataPlaneClusterStatusNodeInfo{
					Current: 6,
				},
			},
			wantErr: false,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID: clusterID,
							Status:    api.ClusterReady,
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					GetComputeNodesFunc: func(clusterID string) (*types.ComputeNodesInfo, *errors.ServiceError) {
						return &types.ComputeNodesInfo{
							Actual:  6,
							Desired: 6,
						}, nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataPlaneClusterService := tt.dataPlaneClusterServiceFactory()
			svcErr := dataPlaneClusterService.UpdateDataPlaneClusterStatus(context.Background(), tt.clusterID, tt.clusterStatus)
			gotErr := svcErr != nil
			if gotErr != tt.wantErr {
				t.Errorf("UpdateDataPlaneClusterStatus() error = %v, wantErr = %v", svcErr, tt.wantErr)
			}
		})
	}
}

func Test_DataPlaneCluster_updateDataPlaneClusterNodes(t *testing.T) {
	testClusterID := "test-cluster-id"

	type input struct {
		status                  *dbapi.DataPlaneClusterStatus
		cluster                 *api.Cluster
		dataPlaneClusterService *dataPlaneClusterService
	}
	cases := []struct {
		name           string
		inputFactory   func() *input
		expectedResult int
		wantErr        bool
	}{
		{
			name: "when scale-up thresholds are crossed number of compute nodes is increased",
			inputFactory: func() *input {
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 6,
			wantErr:        false,
		},
		{
			name: "when a single scale-up threshold is crossed number of compute nodes is increased",
			inputFactory: func() *input {
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.Remaining.Connections = 10000000000
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 6,
			wantErr:        false,
		},
		{
			name: "when scale-up threshold is crossed but scale-up nodes would be higher than restricted celing then no scaling is performed",
			inputFactory: func() *input {
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 5 // We test restricted ceiling rounding here
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 3,
			wantErr:        false,
		},
		{
			name: "when all scale-down threshold is crossed number of compute nodes is decreased",
			inputFactory: func() *input {
				kafkaConfig := sampleValidApplicationConfigForDataPlaneClusterTest(nil).KafkaConfig
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 6
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				// We set remaining to a value much higher than resizeInfo.value which to
				// simulate a scale-down is needed, as scale-down thresholds are
				// calculated from resizeInfo.Delta value
				testStatus.ResizeInfo.Delta.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 10
				testStatus.ResizeInfo.Delta.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 10
				testStatus.Remaining.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 1000
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 1000
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
				}

				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 3,
			wantErr:        false,
		},
		{
			name: "when not all scale-down threshold are crossed number of compute nodes is not decreased",
			inputFactory: func() *input {
				kafkaConfig := sampleValidApplicationConfigForDataPlaneClusterTest(nil).KafkaConfig
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 6
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.ResizeInfo.Delta.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 10
				testStatus.ResizeInfo.Delta.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 10
				// We simulate connections scale-down threshold not being crossed
				// and partitions scale-down threshold being crossed
				testStatus.Remaining.Connections = testStatus.ResizeInfo.Delta.Connections - 1
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 1000
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 6,
			wantErr:        false,
		},
		{
			name: "when scale-down threshold is crossed but scaled-down nodes would be less than workloadMin then no scaling is performed",
			inputFactory: func() *input {
				kafkaConfig := sampleValidApplicationConfigForDataPlaneClusterTest(nil).KafkaConfig
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 6
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 6
				// We set remaining to a value much higher than resizeInfo.value which to
				// simulate a scale-down is needed, as scale-down thresholds are
				// calculated from resizeInfo.Delta value
				testStatus.ResizeInfo.Delta.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 10
				testStatus.ResizeInfo.Delta.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 10
				testStatus.Remaining.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 1000
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 1000
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 6,
			wantErr:        false,
		},
		{
			name: "when scale-down threshold is crossed but scaled-down nodes would be less than restricted floor then no scaling is performed",
			inputFactory: func() *input {
				kafkaConfig := sampleValidApplicationConfigForDataPlaneClusterTest(nil).KafkaConfig
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 6
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.NodeInfo.Floor = 5 // We test the rounding of restricted floor here
				// We set remaining to a value much higher than resizeInfo.value which to
				// simulate a scale-down is needed, as scale-down thresholds are
				// calculated from resizeInfo.Delta value
				testStatus.ResizeInfo.Delta.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 10
				testStatus.ResizeInfo.Delta.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 10
				testStatus.Remaining.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 1000
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 1000
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 6,
			wantErr:        false,
		},
		{
			name: "when no scale-up or scale-down thresholds are crossed no scaling is performed",
			inputFactory: func() *input {
				kafkaConfig := sampleValidApplicationConfigForDataPlaneClusterTest(nil).KafkaConfig
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				testStatus.NodeInfo.Current = 12
				testStatus.NodeInfo.Ceiling = 30
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.NodeInfo.Floor = 3

				// We set remaining higher than a single kafka instance capacity to not
				// trigger scale-up and we set it less than delta values to not force a
				// scale-down
				testStatus.Remaining.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 2
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 2
				testStatus.ResizeInfo.Delta.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections * 10
				testStatus.ResizeInfo.Delta.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions * 10

				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						return nil, nil
					},
				}
				dataPlaneClusterService := NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}
			},
			expectedResult: 12,
			wantErr:        false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.inputFactory()
			if input == nil {
				t.Fatalf("invalid input")
			}

			dataPlaneClusterService := input.dataPlaneClusterService
			nodesAfterScaling, err := dataPlaneClusterService.updateDataPlaneClusterNodes(input.cluster, input.status)

			if err != nil != tt.wantErr {
				t.Errorf("updateDataPlaneClusterNodes() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			Expect(nodesAfterScaling).To(Equal(tt.expectedResult))
		})
	}
}

func Test_DataPlaneCluster_computeNodeScalingActionInProgress(t *testing.T) {
	testClusterID := "test-cluster-id"
	tests := []struct {
		name                           string
		clusterStatus                  *dbapi.DataPlaneClusterStatus
		dataPlaneClusterServiceFactory func() *dataPlaneClusterService
		wantErr                        bool
		want                           bool
	}{
		{
			name:          "When desired compute nodes equals existing compute nodes no scaling action is in progress",
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					GetComputeNodesFunc: func(clusterID string) (*types.ComputeNodesInfo, *errors.ServiceError) {
						return &types.ComputeNodesInfo{
							Actual:  6,
							Desired: 6,
						}, nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

			},
			want:    false,
			wantErr: false,
		},
		{
			name:          "When desired compute nodes does not equal existing compute nodes scaling action is in progress",
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					GetComputeNodesFunc: func(clusterID string) (*types.ComputeNodesInfo, *errors.ServiceError) {
						return &types.ComputeNodesInfo{
							Actual:  6,
							Desired: 8,
						}, nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

			},
			want:    true,
			wantErr: false,
		},
		{
			name:          "When some node information is missing an error is returned",
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					GetComputeNodesFunc: func(clusterID string) (*types.ComputeNodesInfo, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get compute nodes info")
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))

			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			testAPICluster := &api.Cluster{
				ClusterID: testClusterID,
			}
			res, err := f.computeNodeScalingActionInProgress(testAPICluster, nil)
			if err != nil != tt.wantErr {
				t.Errorf("computeNodeScalingActionInProgress() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			Expect(res).To(Equal(tt.want))
		})
	}
}

func Test_DataPlaneCluster_isFleetShardOperatorReady(t *testing.T) {
	tests := []struct {
		name                           string
		clusterStatus                  *dbapi.DataPlaneClusterStatus
		dataPlaneClusterServiceFactory func() *dataPlaneClusterService
		wantErr                        bool
		want                           bool
	}{
		{
			name: "When KAS Fleet operator reports ready condition set to true the fleet shard operator is considered ready",
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{
					{
						Type:   "Ready",
						Status: "True",
					},
				},
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			wantErr: false,
			want:    true,
		},
		{
			name: "When KAS Fleet operator reports ready condition set to false the fleet shard operator is considered not ready",
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{
					{
						Type:   "Ready",
						Status: "False",
					},
				},
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			wantErr: false,
			want:    false,
		},
		{
			name: "When KAS Fleet operator reports doesn't report a Ready condition the fleet shard operator is considered not ready",
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{},
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			wantErr: false,
			want:    false,
		},
		{
			name: "When KAS Fleet operator reports reports a Ready condition with an unknown value an error is returned",
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{
					{
						Type:   "Ready",
						Status: "InventedValue",
					},
				},
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			wantErr: true,
			want:    false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res, err := f.isFleetShardOperatorReady(tt.clusterStatus)
			if err != nil != tt.wantErr {
				t.Errorf("isFleetShardOperatorReady() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			Expect(res).To(Equal(tt.want))
		})
	}
}

func Test_DataPlaneCluster_clusterCanProcessStatusReports(t *testing.T) {
	tests := []struct {
		name                           string
		apiCluster                     *api.Cluster
		dataPlaneClusterServiceFactory func() *dataPlaneClusterService
		want                           bool
	}{
		{
			name: "When cluster is ready then status reports can be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterReady,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: true,
		},
		{
			name: "When cluster is full then status reports can be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterFull,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: true,
		},
		{
			name: "When cluster is waiting for KAS Fleet Shard operator then status reports can be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterWaitingForKasFleetShardOperator,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: true,
		},
		{
			name: "When cluster is in state scaling up then status reports cannot be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterComputeNodeScalingUp,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: true,
		},
		{
			name: "When cluster is in state provisioning then status reports cannot be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterProvisioning,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: false,
		},
		{
			name: "When cluster is in state failed then status reports cannot be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterFailed,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: false,
		},
		{
			name: "When cluster is in state accepted then status reports cannot be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterAccepted,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: false,
		},
		{
			name: "When cluster is in state provisioned then status reports cannot be processed",
			apiCluster: &api.Cluster{
				Status: api.ClusterProvisioned,
			},
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(nil))

			},
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res := f.clusterCanProcessStatusReports(tt.apiCluster)
			Expect(res).To(Equal(tt.want))

		})
	}
}

func TestNewDataPlaneClusterService_GetDataPlaneClusterConfig(t *testing.T) {
	type fields struct {
		clusterService             ClusterService
		ObservabilityConfiguration *observatorium.ObservabilityConfiguration
		DataplaneClusterConfig     *config.DataplaneClusterConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    *dbapi.DataPlaneClusterConfig
	}{
		{
			name: "should success",
			fields: fields{
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{}, nil
					},
				},
				ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{
					ObservabilityConfigRepo:        "test-repo",
					ObservabilityConfigChannel:     "test-channel",
					ObservabilityConfigAccessToken: "test-token",
					ObservabilityConfigTag:         "test-tag",
				},
				DataplaneClusterConfig: sampleValidApplicationConfigForDataPlaneClusterTest(nil).DataplaneClusterConfig,
			},
			wantErr: false,
			want: &dbapi.DataPlaneClusterConfig{Observability: dbapi.DataPlaneClusterConfigObservability{
				AccessToken: "test-token",
				Channel:     "test-channel",
				Repository:  "test-repo",
				Tag:         "test-tag",
			}},
		},
		{
			name: "should fail",
			fields: fields{
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.NotFound("not found")
					},
				},
				ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{
					ObservabilityConfigRepo:        "test-repo",
					ObservabilityConfigChannel:     "test-channel",
					ObservabilityConfigAccessToken: "test-token",
					ObservabilityConfigTag:         "test-tag",
				},
				DataplaneClusterConfig: sampleValidApplicationConfigForDataPlaneClusterTest(nil).DataplaneClusterConfig,
			},
			wantErr: true,
			want:    nil,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDataPlaneClusterService(dataPlaneClusterService{
				ClusterService:         tt.fields.clusterService,
				ObservabilityConfig:    tt.fields.ObservabilityConfiguration,
				DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
			})
			config, err := s.GetDataPlaneClusterConfig(context.TODO(), "test-cluster-id")
			if err != nil && !tt.wantErr {
				t.Fatalf("unexpected error %v", err)
			}
			Expect(config).To(Equal(tt.want))
		})
	}
}

func Test_DataPlaneCluster_setClusterStatus(t *testing.T) {
	type input struct {
		status                  *dbapi.DataPlaneClusterStatus
		cluster                 *api.Cluster
		dataPlaneClusterService *dataPlaneClusterService
	}
	cases := []struct {
		name         string
		inputFactory func() (*input, *api.ClusterStatus)
		wantErr      bool
		want         api.ClusterStatus
	}{
		{
			name: "when there is capacity remaining and cluster is not ready then it is set as ready",
			inputFactory: func() (*input, *api.ClusterStatus) {
				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterFull,
				}
				var spyReceivedUpdateStatus *api.ClusterStatus = new(api.ClusterStatus)
				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if cluster.ClusterID != apiCluster.ClusterID {
							return errors.GeneralError("unexpected test error")
						}
						*spyReceivedUpdateStatus = status
						return nil
					},
				}
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				kafkaConfig := c.KafkaConfig
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.Remaining.Connections = kafkaConfig.KafkaCapacity.TotalMaxConnections + 1
				testStatus.Remaining.Partitions = kafkaConfig.KafkaCapacity.MaxPartitions + 1

				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}, spyReceivedUpdateStatus
			},
			want:    api.ClusterReady,
			wantErr: false,
		},
		{
			name: "when there is no capacity remaining and current number of nodes is less than restricted ceiling then state is set as scaling in progress",
			inputFactory: func() (*input, *api.ClusterStatus) {
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				var spyReceivedUpdateStatus *api.ClusterStatus = new(api.ClusterStatus)

				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if cluster.ClusterID != apiCluster.ClusterID {
							return errors.GeneralError("unexpected test error")
						}
						*spyReceivedUpdateStatus = status
						return nil
					},
				}

				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.Remaining.Connections = 0
				testStatus.Remaining.Partitions = 0

				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}, spyReceivedUpdateStatus
			},
			want:    api.ClusterComputeNodeScalingUp,
			wantErr: false,
		},
		{
			name: "when there is no capacity remaining and dynamic scaling is not enabled then state is set as ready",
			inputFactory: func() (*input, *api.ClusterStatus) {
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterWaitingForKasFleetShardOperator,
				}
				var spyReceivedUpdateStatus *api.ClusterStatus = new(api.ClusterStatus)

				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if cluster.ClusterID != apiCluster.ClusterID {
							return errors.GeneralError("unexpected test error")
						}
						*spyReceivedUpdateStatus = status
						return nil
					},
				}

				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				c.DataplaneClusterConfig.DataPlaneClusterScalingType = config.ManualScaling
				testStatus.NodeInfo.Current = 3
				testStatus.NodeInfo.Ceiling = 10000
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.Remaining.Connections = 0
				testStatus.Remaining.Partitions = 0
				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}, spyReceivedUpdateStatus
			},
			want:    api.ClusterReady,
			wantErr: false,
		},
		{
			name: "when there is no capacity remaining and current number of nodes is higher or equal than restricted ceiling then state is set to full",
			inputFactory: func() (*input, *api.ClusterStatus) {
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterReady,
				}
				var spyReceivedUpdateStatus *api.ClusterStatus = new(api.ClusterStatus)

				clusterService := &ClusterServiceMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*types.ClusterSpec, *errors.ServiceError) {
						if clusterID != apiCluster.ClusterID {
							return nil, errors.GeneralError("unexpected test error")
						}
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if cluster.ClusterID != apiCluster.ClusterID {
							return errors.GeneralError("unexpected test error")
						}
						*spyReceivedUpdateStatus = status
						return nil
					},
				}

				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				testStatus.NodeInfo.Current = 10
				testStatus.NodeInfo.Ceiling = 11
				testStatus.NodeInfo.CurrentWorkLoadMinimum = 3
				testStatus.Remaining.Connections = 0
				testStatus.Remaining.Partitions = 0
				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
				}, spyReceivedUpdateStatus
			},
			want:    api.ClusterFull,
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f, spyReceivedStatus := tt.inputFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res := f.dataPlaneClusterService.setClusterStatus(f.cluster, f.status)
			if res != nil != tt.wantErr {
				t.Errorf("setClusterStatus() got = %+v, expected %+v", res, tt.wantErr)
			}
			if spyReceivedStatus == nil {
				t.Fatalf("spyStatus is nil")
			}
			Expect(*spyReceivedStatus).To(Equal(tt.want))
		})
	}
}

func sampleValidBaseDataPlaneClusterStatusRequest() *dbapi.DataPlaneClusterStatus {
	return &dbapi.DataPlaneClusterStatus{
		Conditions: []dbapi.DataPlaneClusterStatusCondition{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		NodeInfo: dbapi.DataPlaneClusterStatusNodeInfo{
			Ceiling:                0,
			Floor:                  0,
			Current:                0,
			CurrentWorkLoadMinimum: 0,
		},
		Remaining: dbapi.DataPlaneClusterStatusCapacity{
			Connections:             0,
			Partitions:              0,
			IngressThroughputPerSec: "",
			EgressThroughputPerSec:  "",
			DataRetentionSize:       "",
		},
		ResizeInfo: dbapi.DataPlaneClusterStatusResizeInfo{
			NodeDelta: multiAZClusterNodeScalingMultiple,
			Delta: dbapi.DataPlaneClusterStatusCapacity{
				Connections:             0,
				Partitions:              0,
				IngressThroughputPerSec: "",
				EgressThroughputPerSec:  "",
				DataRetentionSize:       "",
			},
		},
	}
}

func sampleValidApplicationConfigForDataPlaneClusterTest(clusterService ClusterService) dataPlaneClusterService {
	dataplaneClusterConfig := config.NewDataplaneClusterConfig()
	dataplaneClusterConfig.DataPlaneClusterScalingType = config.AutoScaling

	return dataPlaneClusterService{
		ClusterService: clusterService,
		KafkaConfig: &config.KafkaConfig{
			KafkaCapacity: config.KafkaCapacityConfig{
				MaxPartitions:       100,
				TotalMaxConnections: 100,
			},
		},
		DataplaneClusterConfig: dataplaneClusterConfig,
	}
}
