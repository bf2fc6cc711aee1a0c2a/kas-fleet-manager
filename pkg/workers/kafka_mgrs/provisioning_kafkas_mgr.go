package kafka_mgrs

import (
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// ProvisioningKafkaManager represents a kafka manager that periodically reconciles kafka requests
type ProvisioningKafkaManager struct {
	id                   string
	workerType           string
	isRunning            bool
	kafkaService         services.KafkaService
	observatoriumService services.ObservatoriumService
	configService        services.ConfigService
	imStop               chan struct{}
	syncTeardown         sync.WaitGroup
	reconciler           workers.Reconciler
}

// NewProvisioningKafkaManager creates a new kafka manager
func NewProvisioningKafkaManager(kafkaService services.KafkaService, id string, observatoriumService services.ObservatoriumService, configService services.ConfigService) *ProvisioningKafkaManager {
	return &ProvisioningKafkaManager{
		id:                   id,
		workerType:           "provisioning_kafka",
		kafkaService:         kafkaService,
		observatoriumService: observatoriumService,
		configService:        configService,
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
	var errors []error

	// handle provisioning kafkas state
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		glog.Errorf("failed to list provisioning kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("provisioning kafkas count = %d", len(provisioningKafkas))
	}
	for _, kafka := range provisioningKafkas {
		glog.V(10).Infof("provisioning kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		// only need to check if Kafka clusters are ready if kas-fleetshard sync is not enabled
		// otherwise they will be set to be ready when kas-fleetshard reports status back to the control plane
		if !k.configService.GetConfig().Kafka.EnableKasFleetshardSync {
			if err := k.reconcileProvisioningKafka(kafka); err != nil {
				glog.Errorf("reconcile provisioning %s: %s", kafka.ID, err.Error())
				errors = append(errors, err)
				continue
			}
		}
	}

	return errors
}

func (k *ProvisioningKafkaManager) reconcileProvisioningKafka(kafka *api.KafkaRequest) error {
	namespace, err := services.BuildNamespaceName(kafka)
	if err != nil {
		return err
	}
	kafkaState, err := k.observatoriumService.GetKafkaState(kafka.Name, namespace)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "failed to get state from observatorium for kafka %s namespace %s cluster %s", kafka.ID, namespace, kafka.ClusterID)
		sentry.CaptureException(wrappedErr)
		return wrappedErr
	}
	if kafkaState.State == observatorium.ClusterStateReady {
		glog.Infof("Kafka %s state %s", kafka.ID, kafkaState.State)
		kafka.Status = constants.KafkaRequestStatusReady.String()
		err := k.kafkaService.Update(kafka)
		if err != nil {
			return errors.Wrapf(err, "failed to update kafka %s to status ready", kafka.ID)
		}

		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		/**TODO if there is a creation failure, total operations needs to be incremented: this info. is not available at this time.*/
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	}
	glog.V(1).Infof("reconciled kafka %s state %s", kafka.ID, kafkaState.State)
	return nil
}
