package kafka_mgrs

import (
	"fmt"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"
)

// AcceptedKafkaManager represents a kafka manager that periodically reconciles accepted kafka requests.
type AcceptedKafkaManager struct {
	workers.BaseWorker
	kafkaService             services.KafkaService
	dataPlaneClusterConfig   *config.DataplaneClusterConfig
	clusterPlacementStrategy services.ClusterPlacementStrategy
	clusterService           services.ClusterService
}

// NewAcceptedKafkaManager creates a new kafka manager to reconcile accepted kafkas.
func NewAcceptedKafkaManager(kafkaService services.KafkaService, clusterPlacementStrategy services.ClusterPlacementStrategy, dataPlaneClusterConfig *config.DataplaneClusterConfig, clusterService services.ClusterService, reconciler workers.Reconciler) *AcceptedKafkaManager {
	return &AcceptedKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "accepted_kafka",
			Reconciler: reconciler,
		},
		kafkaService:             kafkaService,
		clusterPlacementStrategy: clusterPlacementStrategy,
		dataPlaneClusterConfig:   dataPlaneClusterConfig,
		clusterService:           clusterService,
	}
}

// Start initializes the kafka manager to reconcile accepted kafka requests.
func (k *AcceptedKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling accepted kafka requests to stop.
func (k *AcceptedKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *AcceptedKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling accepted kafkas")
	var encounteredErrors []error

	// handle accepted kafkas
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted kafkas"))
	} else {
		glog.Infof("accepted kafkas count = %d", len(acceptedKafkas))
	}

	for _, kafka := range acceptedKafkas {
		glog.V(10).Infof("accepted kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusAccepted, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcileAcceptedKafka(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted kafka %s", kafka.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedKafkaManager) reconcileAcceptedKafka(kafka *dbapi.KafkaRequest) error {
	var cluster *api.Cluster
	kafkaAlreadyAssignedInADataPlaneCluster := kafka.ClusterID != ""
	if !kafkaAlreadyAssignedInADataPlaneCluster {
		assignedCluster, err := k.clusterPlacementStrategy.FindCluster(kafka)
		if err != nil {
			return errors.Wrapf(err, "failed to find cluster for kafka request %s", kafka.ID)
		}
		if assignedCluster == nil {
			return k.markTheUnassignedKafkaAsFailedOrAllowRetryClusterPlacementReconciliation(kafka)
		}
		kafka.ClusterID = assignedCluster.ClusterID
		cluster = assignedCluster
	} else {
		foundCluster, e := k.clusterService.FindClusterByID(kafka.ClusterID)
		if foundCluster == nil || e != nil {
			return errors.Wrapf(e, "failed to find cluster with '%s' for kafka request '%s'", kafka.ClusterID, kafka.ID)
		}
		cluster = foundCluster
	}

	latestReadyStrimziVersion, err := cluster.GetLatestAvailableAndReadyStrimziVersion()
	if err != nil {
		return errors.Wrapf(err, "Failed to get latest available ready strimzi version for cluster %s", cluster.ClusterID)
	}

	noLatestReadyStrimziVersionFound := latestReadyStrimziVersion == nil
	if noLatestReadyStrimziVersionFound {
		return k.markTheAssignedKafkaAsFailedOrAllowRetryStrimziVersionPickingReconciliation(kafka)
	}

	versionAssignementError := k.assignDesiredKafkaVersions(kafka, latestReadyStrimziVersion)
	if versionAssignementError != nil {
		return versionAssignementError
	}

	glog.Infof("Kafka instance with id %s is assigned to cluster with id %s", kafka.ID, kafka.ClusterID)
	kafka.Status = constants.KafkaRequestStatusPreparing.String()
	if err2 := k.kafkaService.Update(kafka); err2 != nil {
		return errors.Wrapf(err2, "failed to update kafka %s with cluster details", kafka.ID)
	}
	return nil
}

func (k *AcceptedKafkaManager) markTheUnassignedKafkaAsFailedOrAllowRetryClusterPlacementReconciliation(kafka *dbapi.KafkaRequest) error {
	durationSinceCreation := time.Since(kafka.CreatedAt)
	logger.Logger.Warningf("No available cluster found for Kafka %s instance of size %s in region %s and cloud provider %s", kafka.InstanceType, kafka.SizeId, kafka.Region, kafka.CloudProvider)
	if durationSinceCreation < constants.AcceptedKafkaMaxRetryDurationWhileWaitingForClusterAssignment {
		return nil
	}
	kafka.Status = constants.KafkaRequestStatusFailed.String()
	kafka.FailedReason = fmt.Sprintf("Region %s in cloud provider %s cannot accept %s Kafka of size %s at the moment.", kafka.Region, kafka.CloudProvider, kafka.InstanceType, kafka.SizeId)
	if err2 := k.kafkaService.Update(kafka); err2 != nil {
		return errors.Wrapf(err2, "failed to update failed kafka %s", kafka.ID)
	}
	metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, durationSinceCreation)
	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	return nil
}

func (k *AcceptedKafkaManager) markTheAssignedKafkaAsFailedOrAllowRetryStrimziVersionPickingReconciliation(kafka *dbapi.KafkaRequest) error {
	durationSinceCreation := time.Since(kafka.CreatedAt)
	// Strimzi version may not be available at the start (i.e. during upgrade of Strimzi operator).
	// We need to allow the reconciler to retry getting and setting of the desired strimzi version for a Kafka request
	// until the max retry duration is reached before updating its status to 'failed'.
	if durationSinceCreation < constants.AcceptedKafkaMaxRetryDurationWhileWaitingForStrimziVersion {
		glog.V(10).Infof("No available and ready strimzi version found for Kafka '%s' in Cluster ID '%s'", kafka.ID, kafka.ClusterID)
		return nil
	}
	kafka.Status = constants.KafkaRequestStatusFailed.String()
	kafka.FailedReason = "Failed to get desired Strimzi version"
	if err := k.kafkaService.Update(kafka); err != nil {
		return errors.Wrapf(err, "failed to update failed kafka %s", kafka.ID)
	}
	metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, durationSinceCreation)
	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	return nil
}

func (*AcceptedKafkaManager) assignDesiredKafkaVersions(kafka *dbapi.KafkaRequest, latestReadyStrimziVersion *api.StrimziVersion) error {
	kafka.DesiredStrimziVersion = latestReadyStrimziVersion.Version
	desiredKafkaVersion := latestReadyStrimziVersion.GetLatestKafkaVersion()
	if desiredKafkaVersion == nil {
		return errors.New(fmt.Sprintf("failed to get Kafka version %s", kafka.ID))
	}
	kafka.DesiredKafkaVersion = desiredKafkaVersion.Version

	desiredKafkaIBPVersion := latestReadyStrimziVersion.GetLatestKafkaIBPVersion()
	if desiredKafkaIBPVersion == nil {
		return errors.New(fmt.Sprintf("failed to get Kafka IBP version %s", kafka.ID))
	}
	kafka.DesiredKafkaIBPVersion = desiredKafkaIBPVersion.Version
	return nil
}
