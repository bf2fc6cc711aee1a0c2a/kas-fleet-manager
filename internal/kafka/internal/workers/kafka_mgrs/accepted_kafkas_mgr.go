package kafka_mgrs

import (
	"fmt"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"
)

// AcceptedKafkaManager represents a kafka manager that periodically reconciles accepted kafka requests.
type AcceptedKafkaManager struct {
	workers.BaseWorker
	kafkaService           services.KafkaService
	quotaServiceFactory    services.QuotaServiceFactory
	dataPlaneClusterConfig *config.DataplaneClusterConfig
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	clusterService         services.ClusterService
}

// NewAcceptedKafkaManager creates a new kafka manager to reconcile accepted kafkas.
func NewAcceptedKafkaManager(kafkaService services.KafkaService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, dataPlaneClusterConfig *config.DataplaneClusterConfig, clusterService services.ClusterService, reconciler workers.Reconciler) *AcceptedKafkaManager {
	return &AcceptedKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "accepted_kafka",
			Reconciler: reconciler,
		},
		kafkaService:           kafkaService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
		clusterService:         clusterService,
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
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted kafkas"))
	} else {
		glog.Infof("accepted kafkas count = %d", len(acceptedKafkas))
	}

	for _, kafka := range acceptedKafkas {
		glog.V(10).Infof("accepted kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusAccepted, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcileAcceptedKafka(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted kafka %s", kafka.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedKafkaManager) reconcileAcceptedKafka(kafka *dbapi.KafkaRequest) error {
	cluster, e := k.clusterService.FindClusterByID(kafka.ClusterID)
	if cluster == nil || e != nil {
		return errors.Wrapf(e, "failed to find cluster with '%s' for kafka request '%s'", kafka.ClusterID, kafka.ID)
	}

	latestReadyStrimziVersion, err := cluster.GetLatestAvailableAndReadyStrimziVersion()
	if err != nil || latestReadyStrimziVersion == nil {
		// Strimzi version may not be available at the start (i.e. during upgrade of Strimzi operator).
		// We need to allow the reconciler to retry getting and setting of the desired strimzi version for a Kafka request
		// until the max retry duration is reached before updating its status to 'failed'.
		durationSinceCreation := time.Since(kafka.CreatedAt)
		if durationSinceCreation < constants2.AcceptedKafkaMaxRetryDuration {
			glog.V(10).Infof("No available and ready strimzi version found for Kafka '%s' in Cluster ID '%s'", kafka.ID, kafka.ClusterID)
			return nil
		}
		kafka.Status = constants2.KafkaRequestStatusFailed.String()
		if err != nil {
			err = errors.Wrapf(err, "failed to get desired Strimzi version %s", kafka.ID)
		} else {
			err = errors.New(fmt.Sprintf("failed to get desired Strimzi version %s", kafka.ID))
		}
		kafka.FailedReason = err.Error()
		if err2 := k.kafkaService.Update(kafka); err2 != nil {
			return errors.Wrapf(err2, "failed to update failed kafka %s", kafka.ID)
		}
		return err
	}
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

	glog.Infof("Kafka instance with id %s is assigned to cluster with id %s", kafka.ID, kafka.ClusterID)
	kafka.Status = constants2.KafkaRequestStatusPreparing.String()
	if err2 := k.kafkaService.Update(kafka); err2 != nil {
		return errors.Wrapf(err2, "failed to update kafka %s with cluster details", kafka.ID)
	}
	return nil
}
