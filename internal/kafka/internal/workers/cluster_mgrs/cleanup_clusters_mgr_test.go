package cluster_mgrs

import (
	"context"
	"errors"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafka_tls_certificate_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/onsi/gomega"

	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestCleanupClustersManager_processCleanupClusters(t *testing.T) {
	deprovisionCluster := api.Cluster{
		Status:               api.ClusterDeprovisioning,
		BaseKafkasDomainName: "some-domain-name.org",
	}

	type fields struct {
		clusterService                       services.ClusterService
		osdIDPKeycloakService                sso.OSDKeycloakService
		kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
			name: "should return an error if reconcileCleanupCluster fails during processing cleaned up clusters",
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
						return apiErrors.GeneralError("failed to deregister client in sso")
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to remove service account client in sso")
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
						return nil
					},
				},
			},
			wantErr: true,
		},
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
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
				clusterService:                       tt.fields.clusterService,
				osdIDPKeycloakService:                tt.fields.osdIDPKeycloakService,
				kasFleetshardOperatorAddon:           tt.fields.kasFleetshardOperatorAddon,
				dataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
				kafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-domain-name.org",
				},
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
		clusterService                       services.ClusterService
		osdIDPKeycloakService                sso.OSDKeycloakService
		kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "receives an error when certificate revocation fails",
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
						return errors.New("some errors")
					},
				},
			},
			arg: api.Cluster{
				ClusterID:            "cluster-id",
				BaseKafkasDomainName: "cluster-id.some-domain-name.org",
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successful deletion of an OSD cluster when there is no need to revoke certificate",
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: nil, // certificate revocation should not happen
				},
			},
			arg: api.Cluster{
				BaseKafkasDomainName: "some-domain-name.org",
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafka_tls_certificate_management.CertificateRevocationReason) error {
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:            "cluster-id",
				BaseKafkasDomainName: "cluster-id.some-domain-name.org",
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &CleanupClustersManager{
				clusterService:                       tt.fields.clusterService,
				osdIDPKeycloakService:                tt.fields.osdIDPKeycloakService,
				kasFleetshardOperatorAddon:           tt.fields.kasFleetshardOperatorAddon,
				dataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
				kafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-domain-name.org",
				},
			}
			g.Expect(c.reconcileCleanupCluster(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
