package workers

import (
	"fmt"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
)

// DeletingKafkaManager represents a kafka manager that periodically reconciles kafka requests
type DeletingKafkaManager struct {
	id            string
	workerType    string
	isRunning     bool
	kafkaService  services.KafkaService
	configService services.ConfigService
	quotaService  services.QuotaService
	imStop        chan struct{}
	syncTeardown  sync.WaitGroup
	reconciler    Reconciler
}

// NewDeletingKafkaManager creates a new kafka manager
func NewDeletingKafkaManager(kafkaService services.KafkaService, id string, configService services.ConfigService, quotaService services.QuotaService) *DeletingKafkaManager {
	return &DeletingKafkaManager{
		id:            id,
		workerType:    "deleting_kaka",
		kafkaService:  kafkaService,
		configService: configService,
		quotaService:  quotaService,
	}
}

func (k *DeletingKafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *DeletingKafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *DeletingKafkaManager) GetID() string {
	return k.id
}

func (c *DeletingKafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *DeletingKafkaManager) Start() {
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *DeletingKafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *DeletingKafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *DeletingKafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *DeletingKafkaManager) reconcile() []error {
	glog.Infoln("reconciling deleting kafkas")
	var errors []error

	// handle deprovisioning requests
	// if kas-fleetshard sync is not enabled, the status we should check is constants.KafkaRequestStatusDeprovision as control plane is responsible for deleting the data
	// otherwise the status should be constants.KafkaRequestStatusDeleting as only at that point the control plane should clean it up
	deprovisionStatus := []constants.KafkaStatus{constants.KafkaRequestStatusDeprovision}
	if k.configService.GetConfig().Kafka.EnableKasFleetshardSync {
		// List both status to keep backward compatibility. The "deleted" status should be removed soon
		deprovisionStatus = []constants.KafkaStatus{constants.KafkaRequestStatusDeleting, constants.KafkaRequestStatusDeleted}
	}
	deprovisioningRequests, serviceErr := k.kafkaService.ListByStatus(deprovisionStatus...)
	if serviceErr != nil {
		glog.Errorf("failed to list kafka deprovisioning requests: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("%s kafkas count = %d", deprovisionStatus[0].String(), len(deprovisioningRequests))
	}

	for _, kafka := range deprovisioningRequests {
		glog.V(10).Infof("deprovisioning kafka id = %s", kafka.ID)
		if err := k.reconcileDeprovisioningRequest(kafka); err != nil {
			glog.Errorf("failed to reconcile deprovisioning request %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	return errors
}

func (k *DeletingKafkaManager) reconcileDeprovisioningRequest(kafka *api.KafkaRequest) error {
	if k.configService.GetConfig().Kafka.EnableQuotaService && kafka.SubscriptionId != "" {
		if err := k.quotaService.DeleteQuota(kafka.SubscriptionId); err != nil {
			return fmt.Errorf("failed to delete subscription id %s for kafka %s : %w", kafka.SubscriptionId, kafka.ID, err)
		}
	}
	if err := k.kafkaService.Delete(kafka); err != nil {
		return fmt.Errorf("failed to delete kafka %s: %w", kafka.ID, err)
	}
	return nil
}
