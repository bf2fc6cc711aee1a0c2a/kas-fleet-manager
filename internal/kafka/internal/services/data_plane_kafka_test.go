package services

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDataPlaneKafkaService_UpdateDataPlaneKafkaService(t *testing.T) {
	testErrorCondMessage := "test failed message"
	bootstrapServer := "test.kafka.example.com"
	ingress := fmt.Sprintf("elb.%s", bootstrapServer)
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
							Status:        constants.KafkaRequestStatusProvisioning.String(),
							Routes:        []byte("[{'domain':'test.example.com', 'router':'test.example.com'}]"),
							RoutesCreated: true,
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						if kafkaRequest.Status == string(constants.KafkaRequestStatusFailed) {
							if !strings.Contains(kafkaRequest.FailedReason, testErrorCondMessage) {
								return errors.GeneralError("Test failure error. Expected FailedReason is empty")
							}
							c["failed"]++

						} else {
							c["rejected"]++
						}
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants.KafkaRequestStatusFailed {
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
			name: "should create default routes if not presented in the request",
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
						Domain: fmt.Sprintf("admin-server-%s", bootstrapServer),
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
							Status:              constants.KafkaRequestStatusProvisioning.String(),
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
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants.KafkaRequestStatusFailed {
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
				// This will set the routes and "RoutesCreated" to be ready
				{
					Conditions: []dbapi.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
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
			name: "should use routes in the requests in they are presented",
			clusterService: &ClusterServiceMock{
				FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
					return &api.Cluster{}, nil
				},
			},
			kafkaService: func(c map[string]int) KafkaService {
				routesCreated := false
				host := strings.SplitN(bootstrapServer, ".", 2)[1]
				expectedRoutes := []dbapi.DataPlaneKafkaRoute{
					{
						Domain: bootstrapServer,
						Router: ingress,
					},
					{
						Domain: fmt.Sprintf("admin-api.%s", host),
						Router: ingress,
					},
					{
						Domain: fmt.Sprintf("broker-0.%s", host),
						Router: ingress,
					},
				}
				return &KafkaServiceMock{
					GetByIdFunc: func(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							ClusterID:           "test-cluster-id",
							Status:              constants.KafkaRequestStatusProvisioning.String(),
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
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants.KafkaRequestStatusDeleting {
							c["deleting"]++
						} else if status == constants.KafkaRequestStatusFailed {
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
					Routes: []dbapi.DataPlaneKafkaRoute{
						{
							Domain: "bootstrap",
							Router: ingress,
						},
						{
							Domain: "admin-api",
							Router: ingress,
						},
						{
							Domain: "broker-0",
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
					Routes: []dbapi.DataPlaneKafkaRoute{
						{
							Domain: "bootstrap",
							Router: ingress,
						},
						{
							Domain: "admin-api",
							Router: ingress,
						},
						{
							Domain: "broker-0",
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
