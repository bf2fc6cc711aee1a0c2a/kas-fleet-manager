package cluster_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/onsi/gomega"

	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestCleanupClustersManager_processCleanupClusters(t *testing.T) {
	deprovisionCluster := api.Cluster{
		Status: api.ClusterDeprovisioning,
	}

	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      sso.OSDKeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		dataplaneClusterConfig     *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
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
			c := &CleanupClustersManager{
				clusterService:             tt.fields.clusterService,
				osdIDPKeycloakService:      tt.fields.osdIDPKeycloakService,
				kasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				dataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
			}

			err := c.processCleanupClusters()
			if !tt.wantErr {
				g.Expect(err).ToNot(gomega.HaveOccurred())
			} else {
				g.Expect(err).To(gomega.HaveOccurred())
			}
		})
	}
}

func TestCleanupClustersManager_reconcileCleanupCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      sso.OSDKeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		dataplaneClusterConfig     *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "receives an error when remove kas-fleetshard-operator service account fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "receives an error when soft delete cluster from database fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "shouldn't attempt to delete OSD IDP client when Kafka SRE reconciliation is disabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: false,
				},
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: nil, // should never be called
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "successful deletion of an OSD cluster",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
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
			c := &CleanupClustersManager{
				clusterService:             tt.fields.clusterService,
				osdIDPKeycloakService:      tt.fields.osdIDPKeycloakService,
				kasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				dataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
			}
			g.Expect(c.reconcileCleanupCluster(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestCleanupClustersManager_reconcileEnterpriseCleanupCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      sso.OSDKeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		dataplaneClusterConfig     *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "successful deletion of an enterprise cluster",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					HardDeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should fail when enterprise cluster can't be deleted",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					HardDeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return apiErrors.GeneralError("error hard deleting enterprise cluster")
					},
				},
				osdIDPKeycloakService: &sso.OSDKeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &CleanupClustersManager{
				clusterService:             tt.fields.clusterService,
				osdIDPKeycloakService:      tt.fields.osdIDPKeycloakService,
				kasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				dataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
			}
			g.Expect(c.reconcileCleanupEnterpriseCluster(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
