package kafka_mgrs

import (
	"fmt"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/google/uuid"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

// AcceptedKafkaManager represents a kafka manager that periodically reconciles kafka requests
type AcceptedKafkaManager struct {
	workers.BaseWorker
	kafkaService           services.KafkaService
	quotaServiceFactory    services.QuotaServiceFactory
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	dataPlaneClusterConfig *config.DataplaneClusterConfig
}

// NewAcceptedKafkaManager creates a new kafka manager
func NewAcceptedKafkaManager(kafkaService services.KafkaService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, bus signalbus.SignalBus, dataPlaneClusterConfig *config.DataplaneClusterConfig) *AcceptedKafkaManager {
	return &AcceptedKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "accepted_kafka",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:           kafkaService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *AcceptedKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
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
	cluster, err := k.clusterPlmtStrategy.FindCluster(kafka)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for kafka request %s", kafka.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Kafka instance with id %s", kafka.ID)
		return nil
	}

	kafka.ClusterID = cluster.ClusterID
	if k.dataPlaneClusterConfig.StrimziOperatorVersion == "" {
		readyStrimziVersions, err := cluster.GetAvailableAndReadyStrimziVersions()
		if err != nil || len(readyStrimziVersions) == 0 {
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
		} else {
			kafka.DesiredStrimziVersion = readyStrimziVersions[len(readyStrimziVersions)-1].Version
		}
	} else {
		kafka.DesiredStrimziVersion = k.dataPlaneClusterConfig.StrimziOperatorVersion
	}

	glog.Infof("Kafka instance with id %s is assigned to cluster with id %s", kafka.ID, kafka.ClusterID)
	kafka.Status = constants2.KafkaRequestStatusPreparing.String()
	if err2 := k.kafkaService.Update(kafka); err2 != nil {
		return errors.Wrapf(err2, "failed to update kafka %s with cluster details", kafka.ID)
	}
	return nil
}
