package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/google/uuid"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/golang/glog"
)

// ProvisioningKafkaManager represents a kafka manager that periodically reconciles kafka requests
type ProvisioningKafkaManager struct {
	workers.BaseWorker
	kafkaService         services.KafkaService
	observatoriumService services.ObservatoriumService
	configService        services.ConfigService
}

// NewProvisioningKafkaManager creates a new kafka manager
func NewProvisioningKafkaManager(kafkaService services.KafkaService, observatoriumService services.ObservatoriumService, configService services.ConfigService, bus signalbus.SignalBus) *ProvisioningKafkaManager {
	return &ProvisioningKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "provisioning_kafka",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:         kafkaService,
		observatoriumService: observatoriumService,
		configService:        configService,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *ProvisioningKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *ProvisioningKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *ProvisioningKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling kafkas")
	var encounteredErrors []error

	// handle provisioning kafkas state
	// Kafkas in a "provisioning" state means that it is ready to be sent to the KAS Fleetshard Operator for Kafka creation in the data plane cluster.
	// The update of the Kafka request status from 'provisioning' to another state will be handled by the KAS Fleetshard Operator.
	// We only need to update the metrics here.
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning kafkas"))
	} else {
		glog.Infof("provisioning kafkas count = %d", len(provisioningKafkas))
	}
	for _, kafka := range provisioningKafkas {
		glog.V(10).Infof("provisioning kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	}

	return encounteredErrors
}
