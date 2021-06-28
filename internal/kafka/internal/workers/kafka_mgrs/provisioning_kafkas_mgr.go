package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/google/uuid"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
)

// ProvisioningKafkaManager represents a kafka manager that periodically reconciles kafka requests
type ProvisioningKafkaManager struct {
	id                   string
	workerType           string
	isRunning            bool
	kafkaService         services.KafkaService
	observatoriumService services.ObservatoriumService
	configService        coreServices.ConfigService
	imStop               chan struct{}
	syncTeardown         sync.WaitGroup
	reconciler           workers.Reconciler
}

// NewProvisioningKafkaManager creates a new kafka manager
func NewProvisioningKafkaManager(kafkaService services.KafkaService, observatoriumService services.ObservatoriumService, configService coreServices.ConfigService, bus signalbus.SignalBus) *ProvisioningKafkaManager {
	return &ProvisioningKafkaManager{
		id:                   uuid.New().String(),
		workerType:           "provisioning_kafka",
		kafkaService:         kafkaService,
		observatoriumService: observatoriumService,
		configService:        configService,
		reconciler: workers.Reconciler{
			SignalBus: bus,
		},
	}
}

func (k *ProvisioningKafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *ProvisioningKafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *ProvisioningKafkaManager) GetID() string {
	return k.id
}

func (c *ProvisioningKafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *ProvisioningKafkaManager) Start() {
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *ProvisioningKafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *ProvisioningKafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *ProvisioningKafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
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
