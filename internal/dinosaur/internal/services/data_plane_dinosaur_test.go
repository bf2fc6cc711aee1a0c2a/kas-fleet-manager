package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func TestDataPlaneDinosaurService_UpdateDataPlaneDinosaurService(t *testing.T) {
	testErrorCondMessage := "test failed message"
	bootstrapServer := "test.dinosaur.example.com"
	ingress := fmt.Sprintf("elb.%s", bootstrapServer)
	invalidIngress := "elb.test1.dinosaur.example.com"
	tests := []struct {
		name            string
		clusterService  ClusterService
		dinosaurService func(c map[string]int) DinosaurService
		clusterId       string
		status          []*dbapi.DataPlaneDinosaurStatus
		wantErr         bool
		expectCounters  map[string]int
	}{
		{
			name: "should return error when cluster id is not valid",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return nil, nil
				},
			},
			dinosaurService: func(c map[string]int) DinosaurService {
				return &DinosaurServiceMock{}
			},
			clusterId: "test-cluster-id",
			status:    []*dbapi.DataPlaneDinosaurStatus{},
			wantErr:   true,
			expectCounters: map[string]int{
				"ready":    0,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			},
		},
		{
			name: "should success",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(c map[string]int) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusFailed) {
							if !strings.Contains(dinosaurRequest.FailedReason, testErrorCondMessage) {
								return errors.GeneralError("Test failure error. Expected FailedReason is empty")
							}
							c["failed"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusReady) {
							c["ready"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusDeleting) {
							c["deleting"]++
						} else {
							c["rejected"]++
						}
						return nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						if status == constants2.DinosaurRequestStatusReady {
							c["ready"]++
						} else if status == constants2.DinosaurRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.DinosaurRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:    "Ready",
							Status:  "False",
							Reason:  "Error",
							Message: testErrorCondMessage,
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Deleted",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Rejected",
						},
					},
				},
			},
			wantErr: false,
			expectCounters: map[string]int{
				"ready":    1,
				"failed":   1,
				"deleting": 1,
				"rejected": 1,
			},
		},
		{
			name: "should use routes in the requests in they are presented",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
				GetClusterDNSFunc: func(clusterID string) (string, *errors.ServiceError) {
					return bootstrapServer, nil
				},
			},
			dinosaurService: func(c map[string]int) DinosaurService {
				routesCreated := false
				expectedRoutes := []dbapi.DataPlaneDinosaurRoute{
					{
						Domain: bootstrapServer,
						Router: ingress,
					},
					{
						Domain: fmt.Sprintf("admin-api-%s", bootstrapServer),
						Router: ingress,
					},
					{
						Domain: fmt.Sprintf("broker-0-%s", bootstrapServer),
						Router: ingress,
					},
				}
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:           "test-cluster-id",
							Status:              constants2.DinosaurRequestStatusProvisioning.String(),
							BootstrapServerHost: bootstrapServer,
							RoutesCreated:       routesCreated,
						}, nil
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						routes, err := dinosaurRequest.GetRoutes()
						if err != nil || !reflect.DeepEqual(routes, expectedRoutes) {
							c["rejected"]++
						} else {
							routesCreated = true
						}
						if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusReady) {
							c["ready"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusDeleting) {
							c["deleting"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusFailed) {
							c["failed"]++
						}
						return nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						if status == constants2.DinosaurRequestStatusReady {
							c["ready"]++
						} else if status == constants2.DinosaurRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.DinosaurRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				// This will set "RoutesCreated" to true
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneDinosaurRouteRequest{
						{
							Name:   "bootstrap",
							Prefix: "",
							Router: ingress,
						},
						{
							Name:   "admin-api",
							Prefix: "admin-api",
							Router: ingress,
						},
						{
							Name:   "broker-0",
							Prefix: "broker-0",
							Router: ingress,
						},
					},
				},
				// This will then set the dinosaur instance to be ready
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneDinosaurRouteRequest{
						{
							Name:   "bootstrap",
							Prefix: "",
							Router: ingress,
						},
						{
							Name:   "admin-api",
							Prefix: "admin-api",
							Router: ingress,
						},
						{
							Name:   "broker-0",
							Prefix: "broker-0",
							Router: ingress,
						},
					},
				},
			},
			wantErr: false,
			expectCounters: map[string]int{
				"ready":    1,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			},
		},
		{
			name: "return error if router is not valid",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
				GetClusterDNSFunc: func(clusterID string) (string, *errors.ServiceError) {
					return bootstrapServer, nil
				},
			},
			dinosaurService: func(c map[string]int) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:           "test-cluster-id",
							Status:              constants2.DinosaurRequestStatusProvisioning.String(),
							BootstrapServerHost: bootstrapServer,
							RoutesCreated:       false,
						}, nil
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				// This will set "RoutesCreated" to true
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneDinosaurRouteRequest{
						{
							Name:   "bootstrap",
							Prefix: "",
							Router: invalidIngress,
						},
						{
							Name:   "admin-api",
							Prefix: "admin-api",
							Router: invalidIngress,
						},
						{
							Name:   "broker-0",
							Prefix: "broker-0",
							Router: invalidIngress,
						},
					},
				},
			},
			wantErr: true,
			expectCounters: map[string]int{
				"ready":    0,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			},
		},
		{
			name: "success when updates dinosaur status to ready and removes failed reason",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(c map[string]int) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
							FailedReason:  testErrorCondMessage,
						}, nil
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusFailed) {
							if !strings.Contains(dinosaurRequest.FailedReason, testErrorCondMessage) {
								return errors.GeneralError("Test failure error. Expected FailedReason is empty")
							}
							c["failed"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusReady) {
							c["ready"]++
						} else if dinosaurRequest.Status == string(constants2.DinosaurRequestStatusDeleting) {
							c["deleting"]++
						} else {
							c["rejected"]++
						}
						return nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						if status == constants2.DinosaurRequestStatusReady {
							c["ready"]++
						} else if status == constants2.DinosaurRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.DinosaurRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			},
			wantErr: false,
			expectCounters: map[string]int{
				"ready":    1,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := map[string]int{
				"ready":    0,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			}
			s := NewDataPlaneDinosaurService(tt.dinosaurService(counter), tt.clusterService, &config.DinosaurConfig{
				NumOfBrokers: 1,
			})
			err := s.UpdateDataPlaneDinosaurService(context.TODO(), tt.clusterId, tt.status)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(counter, tt.expectCounters) {
				t.Errorf("counters dont match. want: %v got: %v", tt.expectCounters, counter)
			}
		})
	}
}

func TestDataPlaneDinosaurService_UpdateVersions(t *testing.T) {
	type versions struct {
		actualDinosaurVersion    string
		actualStrimziVersion     string
		actualDinosaurIBPVersion string
		strimziUpgrading         bool
		dinosaurUpgrading        bool
		dinosaurIBPUpgrading     bool
	}

	tests := []struct {
		name             string
		clusterService   ClusterService
		dinosaurService  func(v *versions) DinosaurService
		clusterId        string
		status           []*dbapi.DataPlaneDinosaurStatus
		wantErr          bool
		expectedVersions versions
	}{
		{
			name: "should update versions",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(v *versions) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:                "test-cluster-id",
							Status:                   constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:                   []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated:            true,
							ActualDinosaurVersion:    "dinosaur-original-ver-0",
							ActualDinosaurIBPVersion: "dinosaur-ibp-original-ver-0",
							ActualStrimziVersion:     "strimzi-original-ver-0",
						}, nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualDinosaurVersion = dinosaurRequest.ActualDinosaurVersion
						v.actualDinosaurIBPVersion = dinosaurRequest.ActualDinosaurIBPVersion
						v.actualStrimziVersion = dinosaurRequest.ActualStrimziVersion
						v.strimziUpgrading = dinosaurRequest.StrimziUpgrading
						v.dinosaurUpgrading = dinosaurRequest.DinosaurUpgrading
						v.dinosaurIBPUpgrading = dinosaurRequest.DinosaurIBPUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
							Reason: "StrimziUpdating",
						},
					},
					DinosaurVersion:    "dinosaur-1",
					StrimziVersion:     "strimzi-1",
					DinosaurIBPVersion: "dinosaur-ibp-3",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualDinosaurVersion:    "dinosaur-1",
				actualStrimziVersion:     "strimzi-1",
				actualDinosaurIBPVersion: "dinosaur-ibp-3",
				strimziUpgrading:         true,
				dinosaurUpgrading:        false,
				dinosaurIBPUpgrading:     false,
			},
		},
		{
			name: "when the condition does not contain a reason then all upgrading fields should be set to false",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(v *versions) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:                "test-cluster-id",
							Status:                   constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:                   []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated:            true,
							ActualDinosaurVersion:    "dinosaur-1",
							ActualStrimziVersion:     "strimzi-1",
							ActualDinosaurIBPVersion: "dinosaur-ibp-1",
							StrimziUpgrading:         true,
							DinosaurUpgrading:        true,
							DinosaurIBPUpgrading:     true,
						}, nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualDinosaurVersion = dinosaurRequest.ActualDinosaurVersion
						v.actualDinosaurIBPVersion = dinosaurRequest.ActualDinosaurIBPVersion
						v.actualStrimziVersion = dinosaurRequest.ActualStrimziVersion
						v.strimziUpgrading = dinosaurRequest.StrimziUpgrading
						v.dinosaurUpgrading = dinosaurRequest.DinosaurUpgrading
						v.dinosaurIBPUpgrading = dinosaurRequest.DinosaurIBPUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					DinosaurVersion:    "dinosaur-1",
					DinosaurIBPVersion: "dinosaur-ibp-1",
					StrimziVersion:     "strimzi-1",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualDinosaurVersion:    "dinosaur-1",
				actualDinosaurIBPVersion: "dinosaur-ibp-1",
				actualStrimziVersion:     "strimzi-1",
				strimziUpgrading:         false,
				dinosaurUpgrading:        false,
				dinosaurIBPUpgrading:     false,
			},
		},
		{
			name: "when received condition is upgrading dinosaur then it is set to true if it wasn't",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(v *versions) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualDinosaurVersion = dinosaurRequest.ActualDinosaurVersion
						v.actualDinosaurIBPVersion = dinosaurRequest.ActualDinosaurIBPVersion
						v.actualStrimziVersion = dinosaurRequest.ActualStrimziVersion
						v.strimziUpgrading = dinosaurRequest.StrimziUpgrading
						v.dinosaurUpgrading = dinosaurRequest.DinosaurUpgrading
						v.dinosaurIBPUpgrading = dinosaurRequest.DinosaurIBPUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
							Reason: "DinosaurUpdating",
						},
					},
					DinosaurVersion:    "dinosaur-1",
					StrimziVersion:     "strimzi-1",
					DinosaurIBPVersion: "dinosaur-ibp-3",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualDinosaurVersion:    "dinosaur-1",
				actualStrimziVersion:     "strimzi-1",
				actualDinosaurIBPVersion: "dinosaur-ibp-3",
				strimziUpgrading:         false,
				dinosaurUpgrading:        true,
				dinosaurIBPUpgrading:     false,
			},
		},
		{
			name: "when received condition is upgrading dinosaur ibp then it is set to true if it wasn't",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			dinosaurService: func(v *versions) DinosaurService {
				return &DinosaurServiceMock{
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.DinosaurRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdatesFunc: func(dinosaurRequest *dbapi.DinosaurRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualDinosaurVersion = dinosaurRequest.ActualDinosaurVersion
						v.actualDinosaurIBPVersion = dinosaurRequest.ActualDinosaurIBPVersion
						v.actualStrimziVersion = dinosaurRequest.ActualStrimziVersion
						v.strimziUpgrading = dinosaurRequest.StrimziUpgrading
						v.dinosaurUpgrading = dinosaurRequest.DinosaurUpgrading
						v.dinosaurIBPUpgrading = dinosaurRequest.DinosaurIBPUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneDinosaurStatus{
				{
					Conditions: []dbapi.DataPlaneDinosaurStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
							Reason: "DinosaurIbpUpdating",
						},
					},
					DinosaurVersion:    "dinosaur-1",
					StrimziVersion:     "strimzi-1",
					DinosaurIBPVersion: "dinosaur-ibp-3",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualDinosaurVersion:    "dinosaur-1",
				actualStrimziVersion:     "strimzi-1",
				actualDinosaurIBPVersion: "dinosaur-ibp-3",
				strimziUpgrading:         false,
				dinosaurUpgrading:        false,
				dinosaurIBPUpgrading:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := versions{}
			s := NewDataPlaneDinosaurService(tt.dinosaurService(&v), tt.clusterService, &config.DinosaurConfig{
				NumOfBrokers: 1,
			})
			err := s.UpdateDataPlaneDinosaurService(context.TODO(), tt.clusterId, tt.status)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(v, tt.expectedVersions) {
				t.Errorf("versions dont match. want: %v got: %v", tt.expectedVersions, v)
			}
		})
	}
}
