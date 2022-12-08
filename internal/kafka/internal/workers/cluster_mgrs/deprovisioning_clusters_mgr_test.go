package cluster_mgrs

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"

	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestDeprovisioningClustersManager_reconcileDeprovisioningCluster(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "should receive error when FindCluster to retrieve sibling cluster returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to remove cluster")
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called
				},
			},
			wantErr: true,
		},
		{
			name: "should update the status back to ready when the cluster is not empty",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return &api.Cluster{}, nil // cluster is not empty its status will be brought back to ready
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "receives an error when delete OCM cluster fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: nil,
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to remove cluster")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successful deletion of an OSD cluster when auto configuration is enabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "receives an error when the update status back to ready fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return fmt.Errorf("Some errors")
					},
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return &api.Cluster{}, nil // cluster is not empty its status will be brought back to ready
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "does not update cluster status when cluster has not been fully deleted from ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.Errorf("this should not be called")
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &DeprovisioningClustersManager{
				clusterService: tt.fields.clusterService,
			}

			err := c.reconcileDeprovisioningCluster(&tt.arg)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestDeprovisioningClustersManager_processDeprovisioningClusters(t *testing.T) {
	deprovisionCluster := api.Cluster{
		Status: api.ClusterDeprovisioning,
	}
	// enterprise cluster should never be in this status, however
	// it must be tested that deprovisioning of such cluster should be skipped
	enterpriseDeprovisionCluster := api.Cluster{
		Status:      api.ClusterDeprovisioning,
		ClusterType: api.Enterprise.String(),
	}
	autoScalingDataPlaneConfig := &config.DataplaneClusterConfig{
		DataPlaneClusterScalingType: config.AutoScaling,
	}
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileDeprovisioningCluster fails during processing deprovisioned clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to delete cluster")
					},
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					FindNonEmptyClusterByIDFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
		{
			name: "should skip deletion for enterprise clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseDeprovisionCluster,
						}, nil
					},
					// FindNonEmptyClusterByID, Delete and UpdateStatus functions should never be called for enterprise cluster
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &DeprovisioningClustersManager{
				clusterService: tt.fields.clusterService,
			}

			err := c.processDeprovisioningClusters()
			if !tt.wantErr {
				g.Expect(err).ToNot(gomega.HaveOccurred())
			} else {
				g.Expect(err).To(gomega.HaveOccurred())
			}
		})
	}
}
