package kafka_mgrs

import (
	"fmt"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
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
	accountService  account.AccountService
}

// NewReadyKafkaManager creates a new kafka manager
func NewReadyKafkaManager(accountService account.AccountService, kafkaService services.KafkaService, keycloakService coreServices.KafkaKeycloakService, keycloakConfig *keycloak.KeycloakConfig, bus signalbus.SignalBus) *ReadyKafkaManager {
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
		accountService:  accountService,
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
	var encounteredErrors []error

	readyKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready kafkas"))
	} else {
		glog.Infof("ready kafkas count = %d", len(readyKafkas))
	}

	for _, kafka := range readyKafkas {
		glog.V(10).Infof("ready kafka id = %s", kafka.ID)

		if err := k.reconcileAccountNumber(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile account number: %s", kafka.ID))
		}

		if !k.keycloakConfig.EnableAuthenticationOnKafka {
			if err := k.reconcileSsoClientIDAndSecret(kafka); err != nil {
				encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to get ready kafkas sso client: %s", kafka.ID))
			}

			if err := k.reconcileCanaryServiceAccount(kafka); err != nil {
				encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to create ready kafka canary service account: %s", kafka.ID))
			}
		}
	}

	return encounteredErrors
}

func (k *ReadyKafkaManager) reconcileAccountNumber(kafkaRequest *dbapi.KafkaRequest) error {
	if kafkaRequest.AccountNumber == "" {
		if organisation, err := k.accountService.GetOrganization(fmt.Sprintf("external_id='%s'", kafkaRequest.OrganisationId)); err != nil {
			return errors.Wrapf(err, "failed retrieving the account number for organization with external_id '%s'", kafkaRequest.OrganisationId)
		} else {
			kafkaRequest.AccountNumber = organisation.EbsAccountID()
			if err := k.kafkaService.Update(kafkaRequest); err != nil {
				return errors.Wrapf(err, "failed to update kafka request '%s'", kafkaRequest.ID)
			}
		}
	}

	return nil
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

// reconcileCanaryServiceAccount migrates all existing kafkas so that they will have the canary service account created.
// This is only meant to be a temporary code, in the future it can be replaced with the service account rotation logic
func (k *ReadyKafkaManager) reconcileCanaryServiceAccount(kafkaRequest *dbapi.KafkaRequest) error {
	if kafkaRequest.CanaryServiceAccountClientID == "" && kafkaRequest.CanaryServiceAccountClientSecret == "" {
		serviceAccountRequest := coreServices.CompleteServiceAccountRequest{
			Owner:          kafkaRequest.Owner,
			OwnerAccountId: kafkaRequest.OwnerAccountId,
			ClientId:       fmt.Sprintf("%s-%s", services.CanaryServiceAccountPrefix, kafkaRequest.ID),
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
