package kafka_mgrs

import (
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ReadyKafkaManager represents a kafka manager that periodically reconciles ready kafka requests.
type ReadyKafkaManager struct {
	workers.BaseWorker
	kafkaService    services.KafkaService
	keycloakService sso.KeycloakService
	keycloakConfig  *keycloak.KeycloakConfig
}

// NewReadyKafkaManager creates a new kafka manager to reconcile ready kafkas.
func NewReadyKafkaManager(kafkaService services.KafkaService, keycloakService sso.KafkaKeycloakService, keycloakConfig *keycloak.KeycloakConfig, reconciler workers.Reconciler) *ReadyKafkaManager {
	return &ReadyKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "ready_kafka",
			Reconciler: reconciler,
		},
		kafkaService:    kafkaService,
		keycloakService: keycloakService,
		keycloakConfig:  keycloakConfig,
	}
}

// Start initializes the kafka manager to reconcile ready kafka requests.
func (k *ReadyKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling ready kafka requests to stop.
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
		if err := k.reconcileCanaryServiceAccount(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to create ready kafka canary service account: %s", kafka.ID))
		}
	}

	return encounteredErrors
}

// reconcileCanaryServiceAccount migrates all existing kafkas so that they will have the canary service account created.
// This is only meant to be a temporary code, in the future it can be replaced with the service account rotation logic.
func (k *ReadyKafkaManager) reconcileCanaryServiceAccount(kafkaRequest *dbapi.KafkaRequest) error {
	if kafkaRequest.CanaryServiceAccountClientID == "" && kafkaRequest.CanaryServiceAccountClientSecret == "" {
		clientId := strings.ToLower(fmt.Sprintf("%s-%s", services.CanaryServiceAccountPrefix, kafkaRequest.ID))
		serviceAccountRequest := sso.CompleteServiceAccountRequest{
			Owner:          kafkaRequest.Owner,
			OwnerAccountId: kafkaRequest.OwnerAccountId,
			ClientId:       clientId,
			OrgId:          kafkaRequest.OrganisationId,
			Name:           fmt.Sprintf("canary-service-account-for-kafka %s", kafkaRequest.ID),
			Description:    fmt.Sprintf("canary service account for kafka %s", kafkaRequest.ID),
		}

		serviceAccount, err := k.keycloakService.CreateServiceAccountInternal(serviceAccountRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to create canary service account: %s", err.Error())
		}
		kafkaRequest.CanaryServiceAccountClientID = serviceAccount.ClientID
		kafkaRequest.CanaryServiceAccountClientSecret = serviceAccount.ClientSecret
		if err = k.kafkaService.Update(kafkaRequest); err != nil {
			return errors.Wrapf(err, "failed to update kafka %s with canary service account details", kafkaRequest.ID)
		}
	}

	return nil
}
