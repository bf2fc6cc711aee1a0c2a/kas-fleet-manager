package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	serviceErr "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
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
	workers.BaseWorker
	kafkaService            services.KafkaService
	accessControlListConfig *acl.AccessControlListConfig
	kafkaConfig             *config.KafkaConfig
}

// NewKafkaManager creates a new kafka manager
func NewKafkaManager(kafkaService services.KafkaService, accessControlList *acl.AccessControlListConfig, kafka *config.KafkaConfig, bus signalbus.SignalBus) *KafkaManager {
	return &KafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "general_kafka_worker",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:            kafkaService,
		accessControlListConfig: accessControlList,
		kafkaConfig:             kafka,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *KafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *KafkaManager) Stop() {
	k.StopWorker(k)
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
	accessControlListConfig := k.accessControlListConfig
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("reconciling denied kafka owners")
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(kafkaDeprovisioningForDeniedOwnersErr, "Failed to deprovision kafka for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// cleaning up expired qkafkas
	kafkaConfig := k.kafkaConfig
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

func (k *KafkaManager) reconcileDeniedKafkaOwners(deniedUsers acl.DeniedUsers) *serviceErr.ServiceError {
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
