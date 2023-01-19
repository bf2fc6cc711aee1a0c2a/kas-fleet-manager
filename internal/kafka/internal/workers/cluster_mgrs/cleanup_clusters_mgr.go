package cluster_mgrs

import (
	"context"

	fleeterrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafka_tls_certificate_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"

	"github.com/pkg/errors"
)

const (
	cleanupClustersWorkerType = "cleanup_clusters"
)

// CleanupClustersManager represents a worker that periodically reconciles data plane clusters in cleanup state.
type CleanupClustersManager struct {
	workers.BaseWorker
	clusterService                       services.ClusterService
	kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
	osdIDPKeycloakService                sso.OsdKeycloakService
	dataplaneClusterConfig               *config.DataplaneClusterConfig
	kafkaConfig                          *config.KafkaConfig
	kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
}

// NewCleanupClustersManager creates a new worker that reconciles data plane clusters in cleanup state.
func NewCleanupClustersManager(reconciler workers.Reconciler,
	clusterService services.ClusterService,
	kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon,
	osdIDPKeycloakService sso.OsdKeycloakService, dataplaneClusterConfig *config.DataplaneClusterConfig, kafkaConfig *config.KafkaConfig,
	kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService) *CleanupClustersManager {
	return &CleanupClustersManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: cleanupClustersWorkerType,
			Reconciler: reconciler,
		},
		clusterService:                       clusterService,
		kasFleetshardOperatorAddon:           kasFleetshardOperatorAddon,
		osdIDPKeycloakService:                osdIDPKeycloakService,
		dataplaneClusterConfig:               dataplaneClusterConfig,
		kafkaConfig:                          kafkaConfig,
		kafkaTLSCertificateManagementService: kafkaTLSCertificateManagementService,
	}
}

// Start initializes the worker to reconcile data plane clusters in cleanup state.
func (m *CleanupClustersManager) Start() {
	m.StartWorker(m)
}

// Stop causes the process for reconciling data plane clusters in cleanup state to stop.
func (m *CleanupClustersManager) Stop() {
	m.StopWorker(m)
}

func (m *CleanupClustersManager) Reconcile() []error {
	glog.Infoln("reconciling clusters")

	var errList fleeterrors.ErrorList
	err := m.processCleanupClusters()
	if err != nil {
		errList.AddErrors(err)
	}

	return errList.ToErrorSlice()
}

func (m *CleanupClustersManager) processCleanupClusters() error {
	var errList fleeterrors.ErrorList

	cleanupClusters, serviceErr := m.clusterService.ListByStatus(api.ClusterCleanup)
	if serviceErr != nil {
		errList.AddErrors(errors.Wrap(serviceErr, "failed to list of cleanup clusters"))
		return errList
	}

	glog.Infof("cleanup clusters count = %d", len(cleanupClusters))

	for _, cluster := range cleanupClusters {
		glog.V(10).Infof("cleanup cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterCleanup)
		if err := m.reconcileCleanupCluster(cluster); err != nil {
			errList.AddErrors(errors.Wrapf(err, "failed to reconcile cleanup cluster %s", cluster.ClusterID))
		}
	}

	if errList.IsEmpty() {
		return nil
	}

	return errList
}

func (m *CleanupClustersManager) reconcileCleanupCluster(cluster api.Cluster) error {
	if m.dataplaneClusterConfig.EnableKafkaSreIdentityProviderConfiguration {
		glog.Infof("Removing Dataplane cluster %s IDP client", cluster.ClusterID)
		keycloakDeregistrationErr := m.osdIDPKeycloakService.DeRegisterClientInSSO(cluster.ID)
		if keycloakDeregistrationErr != nil {
			return errors.Wrapf(keycloakDeregistrationErr, "failed to removed Dataplane cluster %s IDP client", cluster.ClusterID)
		}
	}
	glog.Infof("Removing Dataplane cluster %s fleetshard service account", cluster.ClusterID)
	serviceAccountRemovalErr := m.kasFleetshardOperatorAddon.RemoveServiceAccount(cluster)
	if serviceAccountRemovalErr != nil {
		return errors.Wrapf(serviceAccountRemovalErr, "failed to removed Dataplane cluster %s fleetshard service account", cluster.ClusterID)
	}

	// only revoke non shared wildcard certificate.
	// I.e we do not need to revoke a shared wildcard certificate for base Kafka domain (*.<kafka-domain-name>) for data plane clusters that existed before ADR-89 https://github.com/bf2fc6cc711aee1a0c2a/architecture/blob/main/_adr/89/index.adoc
	if cluster.BaseKafkasDomainName != "" && cluster.BaseKafkasDomainName != m.kafkaConfig.KafkaDomainName {
		certRevocationErr := m.kafkaTLSCertificateManagementService.RevokeCertificate(context.Background(), cluster.BaseKafkasDomainName, kafka_tls_certificate_management.CessationOfOperation)
		if certRevocationErr != nil {
			return certRevocationErr
		}
	}

	glog.Infof("Soft deleting the Dataplane cluster %s from the database", cluster.ClusterID)
	deleteError := m.clusterService.DeleteByClusterID(cluster.ClusterID)
	if deleteError != nil {
		return errors.Wrapf(deleteError, "failed to soft delete Dataplane cluster %s from the database", cluster.ClusterID)
	}
	return nil
}
