package kafka_mgrs

import (
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

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
	reconciler    workers.Reconciler
}

// NewDeletingKafkaManager creates a new kafka manager
func NewDeletingKafkaManager(kafkaService services.KafkaService, id string, configService services.ConfigService, quotaService services.QuotaService) *DeletingKafkaManager {
	return &DeletingKafkaManager{
		id:            id,
		workerType:    "deleting_kafka",
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

func (k *DeletingKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling deleting kafkas")
	var encounteredErrors []error

	// List both status to keep backward compatibility. The "deleted" status should be removed soon
	deletingStatus := []constants.KafkaStatus{constants.KafkaRequestStatusDeleting, constants.KafkaRequestStatusDeleted}

	deprovisioningRequests, serviceErr := k.kafkaService.ListByStatus(deletingStatus...)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list kafka deprovisioning requests"))
	} else {
		glog.Infof("%s kafkas count = %d", deletingStatus[0].String(), len(deprovisioningRequests))
	}

	// We also want to remove Kafkas that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningRequestsTemp, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list kafka deprovisioning requests"))
	}

	for _, deprovisioningKafka := range deprovisioningRequestsTemp {
		// If the fleetshard sync is disabled, always rmeove deprovisioning Kafkas
		if !k.configService.GetConfig().Kafka.EnableKasFleetshardSync {
			deprovisioningRequests = append(deprovisioningRequests, deprovisioningKafka)
		}

		// As long as one of the three fields checked below are empty, the Kafka wouldn't have been provisioned to an OSD cluster and should be deleted immediately
		if deprovisioningKafka.BootstrapServerHost == "" {
			deprovisioningRequests = append(deprovisioningRequests, deprovisioningKafka)
			continue
		}

		// If EnableAuthenticationOnKafka is not set, these fields would also be empty even when provisioned to am OSD cluster
		if k.configService.GetConfig().Keycloak.EnableAuthenticationOnKafka && (deprovisioningKafka.SsoClientID == "" || deprovisioningKafka.SsoClientSecret == "") {
			deprovisioningRequests = append(deprovisioningRequests, deprovisioningKafka)
		}
	}

	for _, kafka := range deprovisioningRequests {
		glog.V(10).Infof("deprovisioning kafka id = %s", kafka.ID)
		if err := k.reconcileDeprovisioningRequest(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deprovisioning request %s", kafka.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *DeletingKafkaManager) reconcileDeprovisioningRequest(kafka *api.KafkaRequest) error {
	if k.configService.GetConfig().Kafka.EnableQuotaService && kafka.SubscriptionId != "" {
		if err := k.quotaService.DeleteQuota(kafka.SubscriptionId); err != nil {
			return errors.Wrapf(err, "failed to delete subscription id %s for kafka %s", kafka.SubscriptionId, kafka.ID)
		}
	}
	if err := k.kafkaService.Delete(kafka); err != nil {
		return errors.Wrapf(err, "failed to delete kafka %s", kafka.ID)
	}
	return nil
}
