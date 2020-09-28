package workers

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// build a test addonInstallation
func buildAddonInstallation(id string, state clustersmgmtv1.AddOnInstallationState) *clustersmgmtv1.AddOnInstallation {
	managedKafkaAddonBuilder := clustersmgmtv1.NewAddOnInstallation()
	if id != "" {
		managedKafkaAddonBuilder.ID(id)
	}
	if state != "" {
		managedKafkaAddonBuilder.State(state)
	}

	// Not possible to return an error when no cluster or addon information is being set
	managedKafkaAddon, _ := managedKafkaAddonBuilder.Build()
	return managedKafkaAddon
}

func TestClusterManager_reconcileClusterStatus(t *testing.T) {
	type fields struct {
		ocmClient      ocm.Client
		clusterService services.ClusterService
		timer          *time.Timer
	}
	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when getting cluster status from ocm fails",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						return nil, errors.New("test")
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "error when updating status in database fails",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						clusterStatus, err := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateReady).Build()
						if err != nil {
							panic(err)
						}
						return clusterStatus, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(id string, status api.ClusterStatus) error {
						return errors.New("test")
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterProvisioning,
				},
			},
			wantErr: true,
		},
		{
			name: "database update not invoked when update not needed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						clusterStatus, err := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateInstalling).Build()
						if err != nil {
							panic(err)
						}
						return clusterStatus, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(id string, status api.ClusterStatus) error {
						// this should never be invoked as the cluster state is already accurate
						return errors.New("test")
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterProvisioning,
				},
			},
			want: &api.Cluster{
				ClusterID: "test",
				Status:    api.ClusterProvisioning,
			},
		},
		{
			name: "pending state is set when internal cluster status is empty",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						clusterStatus, err := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStatePending).Build()
						if err != nil {
							panic(err)
						}
						return clusterStatus, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(id string, status api.ClusterStatus) error {
						return nil
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    "",
				},
			},
			want: &api.Cluster{
				ClusterID: "test",
				Status:    api.ClusterProvisioning,
			},
		},
		{
			name: "state is failed when underlying ocm cluster failed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						clusterStatus, err := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateError).Build()
						if err != nil {
							panic(err)
						}
						return clusterStatus, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(id string, status api.ClusterStatus) error {
						return nil
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterProvisioning,
				},
			},
			want: &api.Cluster{
				ClusterID: "test",
				Status:    api.ClusterFailed,
			},
		},
		{
			name: "successful reconcile",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterStatusFunc: func(id string) (status *clustersmgmtv1.ClusterStatus, e error) {
						clusterStatus, err := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateReady).Build()
						if err != nil {
							panic(err)
						}
						return clusterStatus, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(id string, status api.ClusterStatus) error {
						return nil
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterProvisioning,
				},
			},
			want: &api.Cluster{
				ClusterID: "test",
				Status:    api.ClusterProvisioned,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ocmClient:      tt.fields.ocmClient,
				clusterService: tt.fields.clusterService,
				timer:          tt.fields.timer,
			}
			got, err := c.reconcileClusterStatus(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reconcileClusterStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusterManager_reconcileStrimziOperator(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
		timer     *time.Timer
	}
	type args struct {
		addOnInstallationState clustersmgmtv1.AddOnInstallationState
	}
	tests := []struct {
		name    string
		fields  fields
		want    *clustersmgmtv1.AddOnInstallation
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when getting managed kafka addon from ocm fails",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						return nil, errors.New("error when getting managed kafka addon from ocm")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty state returned when managed kafka addon not found",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon := &clustersmgmtv1.AddOnInstallation{}
						return managedKafkaAddon, nil
					},
					CreateManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon := &clustersmgmtv1.AddOnInstallation{}
						return managedKafkaAddon, nil
					},
				},
			},
			want:    &clustersmgmtv1.AddOnInstallation{},
			wantErr: false,
		},
		{
			name: "empty state returned when managed kafka addon is found but with no state",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon, err := clustersmgmtv1.NewAddOnInstallation().ID(api.ManagedKafkaAddonID).Build()
						if err != nil {
							panic(err)
						}
						return managedKafkaAddon, nil
					},
				},
			},
			want: buildAddonInstallation(api.ManagedKafkaAddonID, ""),
		},
		{
			name: "failed state returned when managed kafka addon is found but with a AddOnInstallationStateFailed state",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon, err := clustersmgmtv1.NewAddOnInstallation().ID(api.ManagedKafkaAddonID).State(clustersmgmtv1.AddOnInstallationStateFailed).Build()
						if err != nil {
							panic(err)
						}
						return managedKafkaAddon, nil
					},
				},
			},
			want: buildAddonInstallation(api.ManagedKafkaAddonID, clustersmgmtv1.AddOnInstallationStateFailed),
		},
		{
			name: "ready state returned when managed kafka addon is found but with a AddOnInstallationStateReady state",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon, err := clustersmgmtv1.NewAddOnInstallation().ID(api.ManagedKafkaAddonID).State(clustersmgmtv1.AddOnInstallationStateReady).Build()
						if err != nil {
							panic(err)
						}
						return managedKafkaAddon, nil
					},
				},
			},
			want: buildAddonInstallation(api.ManagedKafkaAddonID, clustersmgmtv1.AddOnInstallationStateReady),
		},
		{
			name: "error when creating managed kafka addon from ocm fails",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						managedKafkaAddon := &clustersmgmtv1.AddOnInstallation{}
						return managedKafkaAddon, nil
					},
					CreateManagedKafkaAddonFunc: func(id string) (status *clustersmgmtv1.AddOnInstallation, e error) {
						return nil, errors.New("error when creating managed kafka addon from ocm")
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ocmClient: tt.fields.ocmClient,
				timer:     tt.fields.timer,
			}
			got, err := c.reconcileStrimziOperator("clusterId")
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileStrimziOperator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reconcileStrimziOperator() got = %v, want %v", got, tt.want)
			}
		})
	}
}
