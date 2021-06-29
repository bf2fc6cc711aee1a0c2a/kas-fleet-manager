package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/google/uuid"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/golang/glog"

	serviceErr "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

// PreparingKafkaManager represents a kafka manager that periodically reconciles kafka requests
type PreparingKafkaManager struct {
	id           string
	workerType   string
	isRunning    bool
	kafkaService services.KafkaService
	imStop       chan struct{}
	syncTeardown sync.WaitGroup
	reconciler   workers.Reconciler
}

// NewPreparingKafkaManager creates a new kafka manager
func NewPreparingKafkaManager(kafkaService services.KafkaService, bus signalbus.SignalBus) *PreparingKafkaManager {
	return &PreparingKafkaManager{
		id:           uuid.New().String(),
		workerType:   "preparing_kafka",
		kafkaService: kafkaService,
		reconciler: workers.Reconciler{
			SignalBus: bus,
		},
	}
}

func (k *PreparingKafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *PreparingKafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *PreparingKafkaManager) GetID() string {
	return k.id
}

func (c *PreparingKafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *PreparingKafkaManager) Start() {
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *PreparingKafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *PreparingKafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *PreparingKafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *PreparingKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling preparing kafkas")
	var encounteredErrors []error

	// handle preparing kafkas
	preparingKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusPreparing)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list preparing kafkas"))
	} else {
		glog.Infof("preparing kafkas count = %d", len(preparingKafkas))
	}

	for _, kafka := range preparingKafkas {
		glog.V(10).Infof("preparing kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusPreparing, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcilePreparingKafka(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile preparing kafka %s", kafka.ID))
			continue
		}

	}

	return encounteredErrors
}

func (k *PreparingKafkaManager) reconcilePreparingKafka(kafka *dbapi.KafkaRequest) error {
	if err := k.kafkaService.PrepareKafkaRequest(kafka); err != nil {
		return k.handleKafkaRequestCreationError(kafka, err)
	}

	return nil
}

func (k *PreparingKafkaManager) handleKafkaRequestCreationError(kafkaRequest *dbapi.KafkaRequest, err *serviceErr.ServiceError) error {
	if err.IsServerErrorClass() {
		// retry the kafka creation request only if the failure is caused by server errors
		// and the time elapsed since its db record was created is still within the threshold.
		durationSinceCreation := time.Since(kafkaRequest.CreatedAt)
		if durationSinceCreation > constants.KafkaMaxDurationWithProvisioningErrs {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
			kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
			kafkaRequest.FailedReason = err.Reason
			updateErr := k.kafkaService.Update(kafkaRequest)
			if updateErr != nil {
				return errors.Wrapf(updateErr, "Failed to update kafka %s in failed state. Kafka failed reason %s", kafkaRequest.ID, kafkaRequest.FailedReason)
			}
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
			return errors.Wrapf(err, "Kafka %s is in server error failed state. Maximum attempts has been reached", kafkaRequest.ID)
		}
	} else if err.IsClientErrorClass() {
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
		kafkaRequest.FailedReason = err.Reason
		updateErr := k.kafkaService.Update(kafkaRequest)
		if updateErr != nil {
			return errors.Wrapf(err, "Failed to update kafka %s in failed state", kafkaRequest.ID)
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
		return errors.Wrapf(err, "error creating kafka %s", kafkaRequest.ID)
	}

	return errors.Wrapf(err, "failed to create kafka %s on cluster %s", kafkaRequest.ID, kafkaRequest.ClusterID)
}
