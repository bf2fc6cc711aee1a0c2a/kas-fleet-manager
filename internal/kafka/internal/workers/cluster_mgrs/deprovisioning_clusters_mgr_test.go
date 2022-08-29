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
	autoScalingDataPlaneConfig := &config.DataplaneClusterConfig{
		DataPlaneClusterScalingType: config.AutoScaling,
	}
	type fields struct {
		clusterService         services.ClusterService
		DataplaneClusterConfig *config.DataplaneClusterConfig
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
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return nil, errors.New("failed to find cluster")
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "should update the status back to ready when no sibling cluster found",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
		{
			name: "receives an error when delete OCM cluster fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &api.Cluster{ClusterID: "dummy cluster"}, nil
					},
					UpdateStatusFunc: nil,
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to remove cluster")
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "successful deletion of an OSD cluster when auto configuration is enabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &api.Cluster{ClusterID: "dummy cluster"}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
		{
			name: "successful deletion of an OSD cluster when manual configuration is enabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: nil, // should not be called
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
		},
		{
			name: "receives an error when the update status fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return fmt.Errorf("Some errors")
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
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
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.Errorf("this should not be called")
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &DeprovisioningClustersManager{
				clusterService:         tt.fields.clusterService,
				dataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
			}

			g.Expect(c.reconcileDeprovisioningCluster(&tt.arg) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestDeprovisioningClustersManager_processDeprovisioningClusters(t *testing.T) {
	deprovisionCluster := api.Cluster{
		Status: api.ClusterDeprovisioning,
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
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &deprovisionCluster, nil
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
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &deprovisionCluster, nil
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
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &DeprovisioningClustersManager{
				clusterService:         tt.fields.clusterService,
				dataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
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
