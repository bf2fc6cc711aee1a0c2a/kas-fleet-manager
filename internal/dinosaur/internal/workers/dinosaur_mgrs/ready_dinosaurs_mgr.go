package dinosaur_mgrs

import (
	"fmt"
	"strings"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ReadyDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ReadyDinosaurManager struct {
	workers.BaseWorker
	dinosaurService    services.DinosaurService
	keycloakService coreServices.KeycloakService
	keycloakConfig  *keycloak.KeycloakConfig
}

// NewReadyDinosaurManager creates a new dinosaur manager
func NewReadyDinosaurManager(dinosaurService services.DinosaurService, keycloakService coreServices.DinosaurKeycloakService, keycloakConfig *keycloak.KeycloakConfig, bus signalbus.SignalBus) *ReadyDinosaurManager {
	return &ReadyDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "ready_dinosaur",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		dinosaurService:    dinosaurService,
		keycloakService: keycloakService,
		keycloakConfig:  keycloakConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *ReadyDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *ReadyDinosaurManager) Stop() {
	k.StopWorker(k)
}

func (k *ReadyDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling ready dinosaurs")
	if !k.keycloakConfig.EnableAuthenticationOnDinosaur {
		return nil
	}

	var encounteredErrors []error

	readyDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready dinosaurs"))
	} else {
		glog.Infof("ready dinosaurs count = %d", len(readyDinosaurs))
	}

	for _, dinosaur := range readyDinosaurs {
		glog.V(10).Infof("ready dinosaur id = %s", dinosaur.ID)
		if err := k.reconcileSsoClientIDAndSecret(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to get ready dinosaurs sso client: %s", dinosaur.ID))
		}

		if err := k.reconcileCanaryServiceAccount(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to create ready dinosaur canary service account: %s", dinosaur.ID))
		}
	}

	return encounteredErrors
}

func (k *ReadyDinosaurManager) reconcileSsoClientIDAndSecret(dinosaurRequest *dbapi.DinosaurRequest) error {
	if dinosaurRequest.SsoClientID == "" && dinosaurRequest.SsoClientSecret == "" {
		dinosaurRequest.SsoClientID = services.BuildKeycloakClientNameIdentifier(dinosaurRequest.ID)
		secret, err := k.keycloakService.GetDinosaurClientSecret(dinosaurRequest.SsoClientID)
		if err != nil {
			return errors.Wrapf(err, "failed to get sso client id & secret for dinosaur cluster: %s", dinosaurRequest.SsoClientID)
		}
		dinosaurRequest.SsoClientSecret = secret
		if err = k.dinosaurService.Update(dinosaurRequest); err != nil {
			return errors.Wrapf(err, "failed to update dinosaur %s with cluster details", dinosaurRequest.ID)
		}
	}
	return nil
}

// reconcileCanaryServiceAccount migrates all existing dinosaurs so that they will have the canary service account created.
// This is only meant to be a temporary code, in the future it can be replaced with the service account rotation logic
func (k *ReadyDinosaurManager) reconcileCanaryServiceAccount(dinosaurRequest *dbapi.DinosaurRequest) error {
	if dinosaurRequest.CanaryServiceAccountClientID == "" && dinosaurRequest.CanaryServiceAccountClientSecret == "" {
		clientId := strings.ToLower(fmt.Sprintf("%s-%s", services.CanaryServiceAccountPrefix, dinosaurRequest.ID))
		serviceAccountRequest := coreServices.CompleteServiceAccountRequest{
			Owner:          dinosaurRequest.Owner,
			OwnerAccountId: dinosaurRequest.OwnerAccountId,
			ClientId:       clientId,
			OrgId:          dinosaurRequest.OrganisationId,
			Name:           fmt.Sprintf("canary-service-account-for-dinosaur %s", dinosaurRequest.ID),
			Description:    fmt.Sprintf("canary service account for dinosaur %s", dinosaurRequest.ID),
		}

		serviceAccount, err := k.keycloakService.CreateServiceAccountInternal(serviceAccountRequest)
		if err != nil {
			return errors.Wrapf(err, "failed to create canary service account: %s", dinosaurRequest.SsoClientID)
		}
		dinosaurRequest.CanaryServiceAccountClientID = serviceAccount.ClientID
		dinosaurRequest.CanaryServiceAccountClientSecret = serviceAccount.ClientSecret
		if err = k.dinosaurService.Update(dinosaurRequest); err != nil {
			return errors.Wrapf(err, "failed to update dinosaur %s with canary service account details", dinosaurRequest.ID)
		}
	}

	return nil
}
