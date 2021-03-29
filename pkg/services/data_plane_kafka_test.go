package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDataPlaneKafkaService_UpdateDataPlaneKafkaService(t *testing.T) {
	tests := []struct {
		name           string
		clusterService ClusterService
		kafkaService   func(c map[string]int) KafkaService
		clusterId      string
		status         []*api.DataPlaneKafkaStatus
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
			status:    []*api.DataPlaneKafkaStatus{},
			wantErr:   true,
			expectCounters: map[string]int{
				"ready":    0,
				"failed":   0,
				"deleted":  0,
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
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{
							ClusterID: "test-cluster-id",
							Status:    constants.KafkaRequestStatusProvisioning.String(),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						c["rejected"]++
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						if status == constants.KafkaRequestStatusReady {
							c["ready"]++
						} else if status == constants.KafkaRequestStatusDeleted {
							c["deleted"]++
						} else if status == constants.KafkaRequestStatusFailed {
							c["failed"]++
						}
						return true, nil
					},
					DeleteFunc: func(in1 *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				}
			},
			clusterId: "test-cluster-id",
			status: []*api.DataPlaneKafkaStatus{
				{
					Conditions: []api.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
				{
					Conditions: []api.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Installing",
						},
					},
				},
				{
					Conditions: []api.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Error",
						},
					},
				},
				{
					Conditions: []api.DataPlaneKafkaStatusCondition{
						{
							Type:   "Ready",
							Status: "False",
							Reason: "Deleted",
						},
					},
				},
				{
					Conditions: []api.DataPlaneKafkaStatusCondition{
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
				"deleted":  1,
				"rejected": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := map[string]int{
				"ready":    0,
				"failed":   0,
				"deleted":  0,
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
