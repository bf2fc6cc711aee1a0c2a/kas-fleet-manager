package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	serviceErr "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// we do not add "deleted" status to the list as the kafkas are soft deleted once the status is set to "deleted", so no need to count them here.
var kafkaMetricsStatuses = []constants.KafkaStatus{
	constants.KafkaRequestStatusAccepted,
	constants.KafkaRequestStatusPreparing,
	constants.KafkaRequestStatusProvisioning,
	constants.KafkaRequestStatusReady,
	constants.KafkaRequestStatusDeprovision,
	constants.KafkaRequestStatusDeleting,
	constants.KafkaRequestStatusFailed,
}

// KafkaManager represents a kafka manager that periodically reconciles kafka requests
type KafkaManager struct {
	id            string
	workerType    string
	isRunning     bool
	kafkaService  services.KafkaService
	configService services.ConfigService
	imStop        chan struct{}
	syncTeardown  sync.WaitGroup
	reconciler    workers.Reconciler
}

// NewKafkaManager creates a new kafka manager
func NewKafkaManager(kafkaService services.KafkaService, id string, configService services.ConfigService, bus signalbus.SignalBus) *KafkaManager {
	return &KafkaManager{
		id:            id,
		workerType:    "general_kafka_worker",
		kafkaService:  kafkaService,
		configService: configService,
		reconciler: workers.Reconciler{
			SignalBus: bus,
		},
	}
}

func (k *KafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *KafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *KafkaManager) GetID() string {
	return k.id
}

func (c *KafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *KafkaManager) Start() {
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *KafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *KafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *KafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *KafkaManager) Reconcile() []error {
	glog.Infoln("reconciling kafkas")
	var encounteredErrors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setKafkaStatusCountMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	// delete kafkas of denied owners
	accessControlListConfig := k.configService.GetConfig().AccessControlList
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("reconciling denied kafka owners")
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(kafkaDeprovisioningForDeniedOwnersErr, "Failed to deprovision kafka for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// cleaning up expired kafkas
	kafkaConfig := k.configService.GetConfig().Kafka
	if kafkaConfig.KafkaLifespan.EnableDeletionOfExpiredKafka {
		glog.Infoln("deprovisioning expired kafkas")
		expiredKafkasError := k.kafkaService.DeprovisionExpiredKafkas(kafkaConfig.KafkaLifespan.KafkaLifespanInHours)
		if expiredKafkasError != nil {
			wrappedError := errors.Wrap(expiredKafkasError, "failed to deprovision expired Kafka instances")
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	return encounteredErrors
}

func (k *KafkaManager) reconcileDeniedKafkaOwners(deniedUsers config.DeniedUsers) *serviceErr.ServiceError {
	if len(deniedUsers) < 1 {
		return nil
	}

	return k.kafkaService.DeprovisionKafkaForUsers(deniedUsers)
}

func (k *KafkaManager) setKafkaStatusCountMetric() []error {
	counters, err := k.kafkaService.CountByStatus(kafkaMetricsStatuses)
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Kafkas by status")}
	}

	for _, c := range counters {
		metrics.UpdateKafkaRequestsStatusCountMetric(c.Status, c.Count)
	}

	return nil
}
