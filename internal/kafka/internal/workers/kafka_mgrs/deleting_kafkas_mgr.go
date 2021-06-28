package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/google/uuid"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
)

// DeletingKafkaManager represents a kafka manager that periodically reconciles kafka requests
type DeletingKafkaManager struct {
	id                  string
	workerType          string
	isRunning           bool
	kafkaService        services.KafkaService
	configService       coreServices.ConfigService
	quotaServiceFactory coreServices.QuotaServiceFactory
	imStop              chan struct{}
	syncTeardown        sync.WaitGroup
	reconciler          workers.Reconciler
}

// NewDeletingKafkaManager creates a new kafka manager
func NewDeletingKafkaManager(kafkaService services.KafkaService, configService coreServices.ConfigService, quotaServiceFactory coreServices.QuotaServiceFactory, bus signalbus.SignalBus) *DeletingKafkaManager {
	return &DeletingKafkaManager{
		id:                  uuid.New().String(),
		workerType:          "deleting_kafka",
		kafkaService:        kafkaService,
		configService:       configService,
		quotaServiceFactory: quotaServiceFactory,
		reconciler: workers.Reconciler{
			SignalBus: bus,
		},
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

	// handle deleting kafka requests
	// Kafkas in a "deleting" state have been removed, along with all their resources (i.e. ManagedKafka, Kafka CRs),
	// from the data plane cluster by the KAS Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Kafkas and their records soft deleted from the database.

	deletingKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusDeleting)
	originalTotalKafkaInDeleting := len(deletingKafkas)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting kafka requests"))
	} else {
		glog.Infof("%s kafkas count = %d", constants.KafkaRequestStatusDeleting.String(), originalTotalKafkaInDeleting)
	}

	// We also want to remove Kafkas that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list kafka deprovisioning requests"))
	} else {
		glog.Infof("%s kafkas count = %d", constants.KafkaRequestStatusDeprovision.String(), len(deprovisioningKafkas))
	}

	for _, deprovisioningKafka := range deprovisioningKafkas {
		// As long as one of the three fields checked below are empty, the Kafka wouldn't have been provisioned to an OSD cluster and should be deleted immediately
		if deprovisioningKafka.BootstrapServerHost == "" {
			deletingKafkas = append(deletingKafkas, deprovisioningKafka)
			continue
		}

		// If EnableAuthenticationOnKafka is not set, these fields would also be empty even when provisioned to an OSD cluster
		if k.configService.GetConfig().Keycloak.EnableAuthenticationOnKafka && (deprovisioningKafka.SsoClientID == "" || deprovisioningKafka.SsoClientSecret == "") {
			deletingKafkas = append(deletingKafkas, deprovisioningKafka)
		}
	}

	glog.Infof("An additional of kafkas count = %d which are marked for removal before being provisioned will also be deleted", len(deletingKafkas)-originalTotalKafkaInDeleting)

	for _, kafka := range deletingKafkas {
		glog.V(10).Infof("deleting kafka id = %s", kafka.ID)
		if err := k.reconcileDeletingKafkas(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deleting kafka request %s", kafka.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *DeletingKafkaManager) reconcileDeletingKafkas(kafka *api.KafkaRequest) error {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(kafka.QuotaType))
	if factoryErr != nil {
		return factoryErr
	}
	err := quotaService.DeleteQuota(kafka.SubscriptionId)
	if err != nil {
		return errors.Wrapf(err, "failed to delete subscription id %s for kafka %s", kafka.SubscriptionId, kafka.ID)
	}

	if err := k.kafkaService.Delete(kafka); err != nil {
		return errors.Wrapf(err, "failed to delete kafka %s", kafka.ID)
	}
	return nil
}
