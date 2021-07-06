package kafka_mgrs

import (
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

// DeletingKafkaManager represents a kafka manager that periodically reconciles kafka requests
type DeletingKafkaManager struct {
	workers.BaseWorker
	kafkaService        services.KafkaService
	keycloakConfig      *keycloak.KeycloakConfig
	quotaServiceFactory services.QuotaServiceFactory
}

// NewDeletingKafkaManager creates a new kafka manager
func NewDeletingKafkaManager(kafkaService services.KafkaService, keycloakConfig *keycloak.KeycloakConfig, quotaServiceFactory services.QuotaServiceFactory, bus signalbus.SignalBus) *DeletingKafkaManager {
	return &DeletingKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "deleting_kafka",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:        kafkaService,
		keycloakConfig:      keycloakConfig,
		quotaServiceFactory: quotaServiceFactory,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *DeletingKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *DeletingKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *DeletingKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling deleting kafkas")
	var encounteredErrors []error

	// handle deleting kafka requests
	// Kafkas in a "deleting" state have been removed, along with all their resources (i.e. ManagedKafka, Kafka CRs),
	// from the data plane cluster by the KAS Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Kafkas and their records soft deleted from the database.

	deletingKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusDeleting)
	originalTotalKafkaInDeleting := len(deletingKafkas)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting kafka requests"))
	} else {
		glog.Infof("%s kafkas count = %d", constants2.KafkaRequestStatusDeleting.String(), originalTotalKafkaInDeleting)
	}

	// We also want to remove Kafkas that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list kafka deprovisioning requests"))
	} else {
		glog.Infof("%s kafkas count = %d", constants2.KafkaRequestStatusDeprovision.String(), len(deprovisioningKafkas))
	}

	for _, deprovisioningKafka := range deprovisioningKafkas {
		// As long as one of the three fields checked below are empty, the Kafka wouldn't have been provisioned to an OSD cluster and should be deleted immediately
		if deprovisioningKafka.BootstrapServerHost == "" {
			deletingKafkas = append(deletingKafkas, deprovisioningKafka)
			continue
		}

		// If EnableAuthenticationOnKafka is not set, these fields would also be empty even when provisioned to an OSD cluster
		if k.keycloakConfig.EnableAuthenticationOnKafka && (deprovisioningKafka.SsoClientID == "" || deprovisioningKafka.SsoClientSecret == "") {
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

func (k *DeletingKafkaManager) reconcileDeletingKafkas(kafka *dbapi.KafkaRequest) error {
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
