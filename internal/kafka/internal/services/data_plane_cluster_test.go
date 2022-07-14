package services

import (
	"context"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	"github.com/onsi/gomega"
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
					UpdateFunc: func(cluster api.Cluster) *errors.ServiceError {
						return nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
		},
		{
			name:          "An error is returned when an error occurs while trying to retrieve the cluter using its id",
			clusterID:     testClusterID,
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, &errors.ServiceError{}
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
			wantErr: true,
		},
		{
			name:          "Returns nil if cluster cannot process status reports",
			clusterID:     testClusterID,
			clusterStatus: nil,
			dataPlaneClusterServiceFactory: func() *dataPlaneClusterService {
				clusterService := &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID: clusterID,
							Status:    api.ClusterFailed,
						}, nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
			wantErr: false,
		},
		{
			name:      "An error is returned when the fleet shard operator is not ready",
			clusterID: testClusterID,
			clusterStatus: &dbapi.DataPlaneClusterStatus{
				Conditions: []dbapi.DataPlaneClusterStatusCondition{
					{
						Type:   "Ready",
						Status: "False",
					},
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
							Status:    api.ClusterWaitingForKasFleetShardOperator,
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					UpdateFunc: func(cluster api.Cluster) *errors.ServiceError {
						return nil
					},
				}
				return NewDataPlaneClusterService(sampleValidApplicationConfigForDataPlaneClusterTest(clusterService))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res, err := f.isFleetShardOperatorReady(tt.clusterStatus)
			if err != nil != tt.wantErr {
				t.Errorf("isFleetShardOperatorReady() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(res).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res := f.clusterCanProcessStatusReports(tt.apiCluster)
			g.Expect(res).To(gomega.Equal(tt.want))

		})
	}
}

func Test_dataPlaneClusterService_GetDataPlaneClusterConfig(t *testing.T) {
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
			name: "should succeed by returned complete spec object with observability and empty capacity config for a given cluster when autoscaling is not set",
			fields: fields{
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							DynamicCapacityInfo: api.JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
						}, nil
					},
				},
				ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{
					ObservabilityConfigRepo:        "test-repo",
					ObservabilityConfigChannel:     "test-channel",
					ObservabilityConfigAccessToken: "test-token",
					ObservabilityConfigTag:         "test-tag",
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
			want: &dbapi.DataPlaneClusterConfig{
				Observability: dbapi.DataPlaneClusterConfigObservability{
					AccessToken: "test-token",
					Channel:     "test-channel",
					Repository:  "test-repo",
					Tag:         "test-tag",
				},
				DynamicCapacityInfo: map[string]api.DynamicCapacityInfo{},
			},
		},
		{
			name: "should succeed by returned complete spec object with observability and capacity config for a given cluster when autoscaling is set",
			fields: fields{
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							DynamicCapacityInfo: api.JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
						}, nil
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
			want: &dbapi.DataPlaneClusterConfig{
				Observability: dbapi.DataPlaneClusterConfigObservability{
					AccessToken: "test-token",
					Channel:     "test-channel",
					Repository:  "test-repo",
					Tag:         "test-tag",
				},
				DynamicCapacityInfo: map[string]api.DynamicCapacityInfo{
					"key1": {
						MaxNodes:       1,
						RemainingUnits: 1,
						MaxUnits:       1,
					},
				},
			},
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
		{
			name: "nil is returned when the provided cluster cannot be found",
			fields: fields{
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			s := NewDataPlaneClusterService(dataPlaneClusterService{
				ClusterService:         tt.fields.clusterService,
				ObservabilityConfig:    tt.fields.ObservabilityConfiguration,
				DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
			})
			config, err := s.GetDataPlaneClusterConfig(context.TODO(), "test-cluster-id")
			if err != nil && !tt.wantErr {
				t.Fatalf("unexpected error %v", err)
			}
			g.Expect(config).To(gomega.Equal(tt.want))
		})
	}
}

func Test_DataPlaneCluster_setClusterStatus(t *testing.T) {
	type input struct {
		status                  *dbapi.DataPlaneClusterStatus
		cluster                 *api.Cluster
		dataPlaneClusterService *dataPlaneClusterService
		clusterService          *ClusterServiceMock
	}
	cases := []struct {
		name                         string
		inputFactory                 func() *input
		wantErr                      bool
		wantStatus                   api.ClusterStatus
		wantDynamicCapacityInfo      api.JSON
		wantAvailableStrimziVersions api.JSON
	}{
		{
			name: "set cluster status as ready as well as updates dynamic capacity info and available strimzi versions",
			inputFactory: func() *input {
				apiCluster := &api.Cluster{
					ClusterID:           testClusterID,
					MultiAZ:             true,
					Status:              api.ClusterWaitingForKasFleetShardOperator,
					DynamicCapacityInfo: api.JSON([]byte(`{"key":{"max_nodes": 90}}`)),
				}

				clusterService := &ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *errors.ServiceError {
						if cluster.ClusterID != apiCluster.ClusterID {
							return errors.GeneralError("unexpected test error")
						}
						return nil
					},
				}

				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				c.DataplaneClusterConfig.DataPlaneClusterScalingType = config.ManualScaling
				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
					clusterService:          clusterService,
				}
			},
			wantStatus:                   api.ClusterReady,
			wantDynamicCapacityInfo:      api.JSON([]byte(`{"key":{"max_nodes":90,"max_units":10,"remaining_units":2}}`)),
			wantAvailableStrimziVersions: api.JSON([]byte(`[{"version":"1.0.0","ready":true,"kafkaVersions":[{"version":"3.0.1"}],"kafkaIBPVersions":[{"version":"3.0.1"}]}]`)),
			wantErr:                      false,
		},
		{
			name: "return an error when updates in the database fails",
			inputFactory: func() *input {
				apiCluster := &api.Cluster{
					ClusterID:           testClusterID,
					MultiAZ:             true,
					Status:              api.ClusterWaitingForKasFleetShardOperator,
					DynamicCapacityInfo: api.JSON([]byte(`{"key":{"max_nodes": 90}}`)),
				}

				clusterService := &ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *errors.ServiceError {
						return errors.GeneralError("unexpected test error")
					},
				}

				testStatus := sampleValidBaseDataPlaneClusterStatusRequest()
				c := sampleValidApplicationConfigForDataPlaneClusterTest(clusterService)
				c.DataplaneClusterConfig.DataPlaneClusterScalingType = config.ManualScaling
				dataPlaneClusterService := NewDataPlaneClusterService(c)
				return &input{
					status:                  testStatus,
					cluster:                 apiCluster,
					dataPlaneClusterService: dataPlaneClusterService,
					clusterService:          clusterService,
				}
			},
			wantErr: true,
		},
	}

	for _, testcase := range cases {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			f := tt.inputFactory()
			g.Expect(f).ToNot(gomega.BeNil(), "dataPlaneClusterService is nil")

			res := f.dataPlaneClusterService.setClusterStatus(f.cluster, f.status)

			updateCalls := f.clusterService.calls.Update
			g.Expect(updateCalls).To(gomega.HaveLen(1))
			if res != nil != tt.wantErr {
				t.Errorf("setClusterStatus() got = %+v, expected %+v", res, tt.wantErr)
			}

			if !tt.wantErr {
				updatedCluster := updateCalls[0].Cluster
				g.Expect(updatedCluster.Status).To(gomega.Equal(tt.wantStatus))
				g.Expect(updatedCluster.DynamicCapacityInfo).To(gomega.Equal(tt.wantDynamicCapacityInfo))
				g.Expect(updatedCluster.AvailableStrimziVersions).To(gomega.Equal(tt.wantAvailableStrimziVersions))
			}

		})
	}
}

func sampleValidBaseDataPlaneClusterStatusRequest() *dbapi.DataPlaneClusterStatus {
	return &dbapi.DataPlaneClusterStatus{
		DynamicCapacityInfo: map[string]api.DynamicCapacityInfo{
			"key": {
				MaxUnits:       10,
				RemainingUnits: 2,
			},
		},
		AvailableStrimziVersions: []api.StrimziVersion{
			{
				Version: "1.0.0",
				Ready:   true,
				KafkaIBPVersions: []api.KafkaIBPVersion{
					{
						Version: "3.0.1",
					},
				},
				KafkaVersions: []api.KafkaVersion{
					{
						Version: "3.0.1",
					},
				},
			},
		},
		Conditions: []dbapi.DataPlaneClusterStatusCondition{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
}

func sampleValidApplicationConfigForDataPlaneClusterTest(clusterService ClusterService) dataPlaneClusterService {
	dataplaneClusterConfig := config.NewDataplaneClusterConfig()
	dataplaneClusterConfig.DataPlaneClusterScalingType = config.AutoScaling

	return dataPlaneClusterService{
		ClusterService:         clusterService,
		DataplaneClusterConfig: dataplaneClusterConfig,
	}
}
