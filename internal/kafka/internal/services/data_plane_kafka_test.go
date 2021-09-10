package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDataPlaneKafkaService_UpdateDataPlaneKafkaService(t *testing.T) {
	testErrorCondMessage := "test failed message"
	bootstrapServer := "test.kafka.example.com"
	ingress := fmt.Sprintf("elb.%s", bootstrapServer)
	invalidIngress := "elb.test1.kafka.example.com"
	tests := []struct {
		name           string
		clusterService ClusterService
		kafkaService   func(c map[string]int) KafkaService
		clusterId      string
		status         []*dbapi.DataPlaneKafkaStatus
		wantErr        bool
		expectCounters map[string]int
	}{
		{
			name: "should return error when cluster id is not valid",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return nil, nil
				},
			},
			kafkaService: func(c map[string]int) KafkaService {
				return &KafkaServiceMock{}
			},
			clusterId: "test-cluster-id",
			status:    []*dbapi.DataPlaneKafkaStatus{},
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
			kafkaService: func(c map[string]int) KafkaService {
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.KafkaRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						if kafkaRequest.Status == string(constants2.KafkaRequestStatusFailed) {
							if !strings.Contains(kafkaRequest.FailedReason, testErrorCondMessage) {
								return errors.GeneralError("Test failure error. Expected FailedReason is empty")
							}
							c["failed"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusReady) {
							c["ready"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusDeleting) {
							c["deleting"]++
						} else {
							c["rejected"]++
						}
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants2.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants2.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.KafkaRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:    "Ready",
							Status:  "False",
							Reason:  "Error",
							Message: testErrorCondMessage,
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Deleted",
						},
					},
				},
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
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
			kafkaService: func(c map[string]int) KafkaService {
				routesCreated := false
				expectedRoutes := []dbapi.DataPlaneKafkaRoute{
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
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:           "test-cluster-id",
							Status:              constants2.KafkaRequestStatusProvisioning.String(),
							BootstrapServerHost: bootstrapServer,
							RoutesCreated:       routesCreated,
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						routes, err := kafkaRequest.GetRoutes()
						if err != nil || !reflect.DeepEqual(routes, expectedRoutes) {
							c["rejected"]++
						} else {
							routesCreated = true
						}
						if kafkaRequest.Status == string(constants2.KafkaRequestStatusReady) {
							c["ready"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusDeleting) {
							c["deleting"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusFailed) {
							c["failed"]++
						}
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants2.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants2.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.KafkaRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				// This will set "RoutesCreated" to true
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneKafkaRouteRequest{
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
				// This will then set the kafka instance to be ready
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneKafkaRouteRequest{
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
			kafkaService: func(c map[string]int) KafkaService {
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:           "test-cluster-id",
							Status:              constants2.KafkaRequestStatusProvisioning.String(),
							BootstrapServerHost: bootstrapServer,
							RoutesCreated:       false,
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				// This will set "RoutesCreated" to true
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Routes: []dbapi.DataPlaneKafkaRouteRequest{
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
			name: "success when updates kafka status to ready and removes failed reason",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			kafkaService: func(c map[string]int) KafkaService {
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.KafkaRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
							FailedReason:  testErrorCondMessage,
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						if kafkaRequest.Status == string(constants2.KafkaRequestStatusFailed) {
							if !strings.Contains(kafkaRequest.FailedReason, testErrorCondMessage) {
								return errors.GeneralError("Test failure error. Expected FailedReason is empty")
							}
							c["failed"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusReady) {
							c["ready"]++
						} else if kafkaRequest.Status == string(constants2.KafkaRequestStatusDeleting) {
							c["deleting"]++
						} else {
							c["rejected"]++
						}
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						v, ok := values["status"]
						if ok {
							statusValue := v.(string)
							c[statusValue]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants2.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants2.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants2.KafkaRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
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
			s := NewDataPlaneKafkaService(tt.kafkaService(counter), tt.clusterService, &config.KafkaConfig{
				NumOfBrokers: 1,
			})
			err := s.UpdateDataPlaneKafkaService(context.TODO(), tt.clusterId, tt.status)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(counter, tt.expectCounters) {
				t.Errorf("counters dont match. want: %v got: %v", tt.expectCounters, counter)
			}
		})
	}
}

func TestDataPlaneKafkaService_UpdateVersions(t *testing.T) {
	type versions struct {
		actualKafkaVersion   string
		actualStrimziVersion string
		strimziUpgrading     bool
	}

	tests := []struct {
		name             string
		clusterService   ClusterService
		kafkaService     func(v *versions) KafkaService
		clusterId        string
		status           []*dbapi.DataPlaneKafkaStatus
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
			kafkaService: func(v *versions) KafkaService {
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:     "test-cluster-id",
							Status:        constants2.KafkaRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualKafkaVersion = kafkaRequest.ActualKafkaVersion
						v.actualStrimziVersion = kafkaRequest.ActualStrimziVersion
						v.strimziUpgrading = kafkaRequest.StrimziUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
							Reason: "StrimziUpdating",
						},
					},
					KafkaVersion:   "kafka-1",
					StrimziVersion: "strimzi-1",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualKafkaVersion:   "kafka-1",
				actualStrimziVersion: "strimzi-1",
				strimziUpgrading:     true,
			},
		},
		{
			name: "should set strimzi_upgrading to false",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			kafkaService: func(v *versions) KafkaService {
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:            "test-cluster-id",
							Status:               constants2.KafkaRequestStatusProvisioning.String(),
							Routes:               []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated:        true,
							ActualKafkaVersion:   "kafka-1",
							ActualStrimziVersion: "strimzi-1",
							StrimziUpgrading:     true,
						}, nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, fields map[string]interface{}) *errors.ServiceError {
						v.actualKafkaVersion = kafkaRequest.ActualKafkaVersion
						v.actualStrimziVersion = kafkaRequest.ActualStrimziVersion
						v.strimziUpgrading = kafkaRequest.StrimziUpgrading
						return nil
					},
					UpdateStatusFunc: func(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					DeleteFunc: func(in1 *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*dbapi.DataPlaneKafkaStatus{
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					KafkaVersion:   "kafka-1",
					StrimziVersion: "strimzi-1",
				},
			},
			wantErr: false,
			expectedVersions: versions{
				actualKafkaVersion:   "kafka-1",
				actualStrimziVersion: "strimzi-1",
				strimziUpgrading:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := versions{}
			s := NewDataPlaneKafkaService(tt.kafkaService(&v), tt.clusterService, &config.KafkaConfig{
				NumOfBrokers: 1,
			})
			err := s.UpdateDataPlaneKafkaService(context.TODO(), tt.clusterId, tt.status)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(v, tt.expectedVersions) {
				t.Errorf("versions dont match. want: %v got: %v", tt.expectedVersions, v)
			}
		})
	}
}
