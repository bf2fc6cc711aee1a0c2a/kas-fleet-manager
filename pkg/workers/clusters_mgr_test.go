package workers

import (
	"testing"
	"time"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

func TestClusterManager_reconcile(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
		timer          *time.Timer
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Sample test case",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CreateFunc: func(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *errors.ServiceError) {
						return &clustersmgmtv1.Cluster{}, nil
					},
				},
				timer: time.NewTimer(30 * time.Second),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				clusterService: tt.fields.clusterService,
				timer:          tt.fields.timer,
			}
			c.reconcile()
		})
	}
}
