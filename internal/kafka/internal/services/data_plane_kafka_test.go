package services

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDataPlaneKafkaService_UpdateDataPlaneKafkaService(t *testing.T) {
	testErrorCondMessage := "test failed message"
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
							ClusterID: "test-cluster-id",
							Status:    constants.KafkaRequestStatusProvisioning.String(),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := map[string]int{
				"ready":    0,
				"failed":   0,
				"deleting": 0,
				"rejected": 0,
			}
			s := NewDataPlaneKafkaService(tt.kafkaService(counter), tt.clusterService)
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
