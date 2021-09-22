package kafka_mgrs

import (
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ReadyKafkaManager represents a kafka manager that periodically reconciles kafka requests
type ReadyKafkaManager struct {
	workers.BaseWorker
	kafkaService    services.KafkaService
	keycloakService coreServices.KeycloakService
	keycloakConfig  *keycloak.KeycloakConfig
}

// NewReadyKafkaManager creates a new kafka manager
func NewReadyKafkaManager(kafkaService services.KafkaService, keycloakService coreServices.KafkaKeycloakService, keycloakConfig *keycloak.KeycloakConfig, bus signalbus.SignalBus) *ReadyKafkaManager {
	return &ReadyKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "ready_kafka",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:    kafkaService,
		keycloakService: keycloakService,
		keycloakConfig:  keycloakConfig,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *ReadyKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *ReadyKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *ReadyKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling ready kafkas")
	if !k.keycloakConfig.EnableAuthenticationOnKafka {
		return nil
	}

	var encounteredErrors []error

	readyKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready kafkas"))
	} else {
		glog.Infof("ready kafkas count = %d", len(readyKafkas))
	}

	for _, kafka := range readyKafkas {
		glog.V(10).Infof("ready kafka id = %s", kafka.ID)
		if err := k.reconcileSsoClientIDAndSecret(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to get ready kafkas sso client: %s", kafka.ID))
		}
	}

	return encounteredErrors
}

func (k *ReadyKafkaManager) reconcileSsoClientIDAndSecret(kafkaRequest *dbapi.KafkaRequest) error {
	if kafkaRequest.SsoClientID == "" && kafkaRequest.SsoClientSecret == "" {
		kafkaRequest.SsoClientID = services.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		secret, err := k.keycloakService.GetKafkaClientSecret(kafkaRequest.SsoClientID)
		if err != nil {
			return errors.Wrapf(err, "failed to get sso client id & secret for kafka cluster: %s", kafkaRequest.SsoClientID)
		}
		kafkaRequest.SsoClientSecret = secret
		if err = k.kafkaService.Update(kafkaRequest); err != nil {
			return errors.Wrapf(err, "failed to update kafka %s with cluster details", kafkaRequest.ID)
		}
	}
	return nil
}
