package services

import (
	"context"
	"testing"

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

	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
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
			g.Expect(res).To(Equal(tt.want))
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

	g := NewWithT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			f := tt.dataPlaneClusterServiceFactory()
			if f == nil {
				t.Fatalf("dataPlaneClusterService is nil")
			}

			res := f.clusterCanProcessStatusReports(tt.apiCluster)
			g.Expect(res).To(Equal(tt.want))

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
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

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
			g.Expect(config).To(Equal(tt.want))
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
			name: "set cluster status as ready",
			inputFactory: func() (*input, *api.ClusterStatus) {
				apiCluster := &api.Cluster{
					ClusterID: testClusterID,
					MultiAZ:   true,
					Status:    api.ClusterWaitingForKasFleetShardOperator,
				}
				var spyReceivedUpdateStatus *api.ClusterStatus = new(api.ClusterStatus)

				clusterService := &ClusterServiceMock{
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
	}

	g := NewWithT(t)

	for _, testcase := range cases {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			f, spyReceivedStatus := tt.inputFactory()
			g.Expect(f).ToNot(BeNil(), "dataPlaneClusterService is nil")

			res := f.dataPlaneClusterService.setClusterStatus(f.cluster, f.status)
			if res != nil != tt.wantErr {
				t.Errorf("setClusterStatus() got = %+v, expected %+v", res, tt.wantErr)
			}
			g.Expect(spyReceivedStatus).ToNot(BeNil())
			g.Expect(*spyReceivedStatus).To(Equal(tt.want))
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
