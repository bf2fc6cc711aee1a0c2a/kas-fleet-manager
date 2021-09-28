package kafka_mgrs

import (
	"fmt"
	"strings"

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

// FailedKafka represents a kafka manager that periodically reconciles kafka requests
type FailedKafka struct {
	workers.BaseWorker
	kafkaService    services.KafkaService
	keycloakService coreServices.KeycloakService
	keycloakConfig  *keycloak.KeycloakConfig
}

// NewFailedKafka creates a new kafka manager
func NewFailedKafka(kafkaService services.KafkaService, keycloakService coreServices.KafkaKeycloakService, keycloakConfig *keycloak.KeycloakConfig, bus signalbus.SignalBus) *FailedKafka {
	return &FailedKafka{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "failed_kafka",
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
func (k *FailedKafka) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *FailedKafka) Stop() {
	k.StopWorker(k)
}

func (k *FailedKafka) Reconcile() []error {
	glog.Infoln("reconciling failed kafkas")
	if !k.keycloakConfig.EnableAuthenticationOnKafka {
		return nil
	}

	var encounteredErrors []error

	failedKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusFailed)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list failed kafkas"))
	} else {
		glog.Infof("failed kafkas count = %d", len(failedKafkas))
	}

	for _, kafka := range failedKafkas {
		glog.V(10).Infof("failed kafka id = %s", kafka.ID)
		if err := k.reconcileCanaryServiceAccount(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to create failed kafka canary service account: %s", kafka.ID))
		}
	}

	return encounteredErrors
}

// reconcileCanaryServiceAccount migrates all existing kafkas so that they will have the canary service account created.
// This is only meant to be a temporary code, in the future it can be replaced with the service account rotation logic
func (k *FailedKafka) reconcileCanaryServiceAccount(kafkaRequest *dbapi.KafkaRequest) error {
	if kafkaRequest.CanaryServiceAccountClientID == "" && kafkaRequest.CanaryServiceAccountClientSecret == "" {
		clientId := strings.ToLower(fmt.Sprintf("%s-%s", services.CanaryServiceAccountPrefix, kafkaRequest.ID))
		serviceAccountRequest := coreServices.CompleteServiceAccountRequest{
			Owner:          kafkaRequest.Owner,
			OwnerAccountId: kafkaRequest.OwnerAccountId,
			ClientId:       clientId,
			OrgId:          kafkaRequest.OrganisationId,
			Name:           fmt.Sprintf("canary-service-account-for-kafka %s", kafkaRequest.ID),
			Description:    fmt.Sprintf("canary service account for kafka %s", kafkaRequest.ID),
		}

		serviceAccount, err := k.keycloakService.CreateServiceAccountInternal(serviceAccountRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to create canary service account: %s", kafkaRequest.SsoClientID)
		}
		kafkaRequest.CanaryServiceAccountClientID = serviceAccount.ClientID
		kafkaRequest.CanaryServiceAccountClientSecret = serviceAccount.ClientSecret
		if err = k.kafkaService.Update(kafkaRequest); err != nil {
			return errors.Wrapf(err, "failed to update kafka %s with canary service account details", kafkaRequest.ID)
		}
	}

	return nil
}
