package workers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/syncsetresources"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// KafkaManager represents a kafka manager that periodically reconciles kafka requests
type KafkaManager struct {
	id                   string
	workerType           string
	isRunning            bool
	ocmClient            ocm.Client
	kafkaService         services.KafkaService
	keycloakService      services.KeycloakService
	observatoriumService services.ObservatoriumService
	configService        services.ConfigService
	quotaService         services.QuotaService
	timer                *time.Timer
	imStop               chan struct{}
	syncTeardown         sync.WaitGroup
	reconciler           Reconciler
	clusterPlmtStrategy  services.ClusterPlacementStrategy
}

// NewKafkaManager creates a new kafka manager
func NewKafkaManager(kafkaService services.KafkaService, ocmClient ocm.Client,
	id string, keycloakService services.KeycloakService, observatoriumService services.ObservatoriumService,
	configService services.ConfigService, quotaService services.QuotaService, clusterPlmtStrategy services.ClusterPlacementStrategy) *KafkaManager {
	return &KafkaManager{
		id:                   id,
		workerType:           "kafka",
		ocmClient:            ocmClient,
		kafkaService:         kafkaService,
		keycloakService:      keycloakService,
		observatoriumService: observatoriumService,
		configService:        configService,
		quotaService:         quotaService,
		clusterPlmtStrategy:  clusterPlmtStrategy,
	}
}

func (k *KafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *KafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *KafkaManager) GetID() string {
	return k.id
}

func (c *KafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *KafkaManager) Start() {
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *KafkaManager) Stop() {
	k.reconciler.Stop(k)
}

func (c *KafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *KafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *KafkaManager) reconcile() []error {
	glog.V(5).Infoln("reconciling kafkas")
	var errors []error

	accessControlListConfig := k.configService.GetConfig().AccessControlList
	if accessControlListConfig.EnableDenyList {
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			glog.Errorf("Failed to deprovision kafka for denied owners %s: %s", accessControlListConfig.DenyList, kafkaDeprovisioningForDeniedOwnersErr.Error())
			errors = append(errors, kafkaDeprovisioningForDeniedOwnersErr)
		}
	}

	// cleaning up expired kafkas
	kafkaConfig := k.configService.GetConfig().Kafka
	if kafkaConfig.EnableDeletionOfExpiredKafka {
		expiredKafkasError := k.kafkaService.DeprovisionExpiredKafkas(kafkaConfig.KafkaLifeSpan)
		if expiredKafkasError != nil {
			glog.Errorf("failed to deprovision expired Kafka instances due to error: %s", expiredKafkasError.Error())
			errors = append(errors, expiredKafkasError)
		}
	}

	// handle deprovisioning requests
	// if kas-fleetshard sync is not enabled, the status we should check is constants.KafkaRequestStatusDeprovision as control plane is responsible for deleting the data
	// otherwise the status should be constants.KafkaRequestStatusDeleted as only at that point the control plane should clean it up
	deprovisionStatus := constants.KafkaRequestStatusDeprovision
	if k.configService.GetConfig().Kafka.EnableKasFleetshardSync {
		deprovisionStatus = constants.KafkaRequestStatusDeleted
	}
	deprovisioningRequests, serviceErr := k.kafkaService.ListByStatus(deprovisionStatus)
	if serviceErr != nil {
		glog.Errorf("failed to list kafka deprovisioning requests: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	}
	for _, kafka := range deprovisioningRequests {
		if err := k.reconcileDeprovisioningRequest(kafka); err != nil {
			glog.Errorf("failed to reconcile deprovisioning request %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	// handle accepted kafkas
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	}
	for _, kafka := range acceptedKafkas {
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusAccepted, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcileAcceptedKafka(kafka); err != nil {
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	// handle preparing kafkas
	preparingKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusPreparing)
	if serviceErr != nil {
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	}
	for _, kafka := range preparingKafkas {
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusPreparing, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcilePreparedKafka(kafka); err != nil {
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}

	}

	// handle provisioning kafkas state
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		glog.Errorf("failed to list provisioning kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	}
	for _, kafka := range provisioningKafkas {
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
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

	readyKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusReady)
	if serviceErr != nil {
		glog.Errorf("failed to list ready kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	}
	if k.configService.GetConfig().Keycloak.EnableAuthenticationOnKafka {
		for _, kafka := range readyKafkas {
			if kafka.Status == string(constants.KafkaRequestStatusReady) {
				if err := k.reconcileSsoClientIDAndSecret(kafka); err != nil {
					glog.Errorf("failed to get provisioning kafkas sso client%s: %s", kafka.SsoClientID, err.Error())
					errors = append(errors, err)
					continue
				}
			}
		}
	}

	errors = k.setKafkaRequestsStatusCountMetric(errors)

	return errors
}

func (k *KafkaManager) reconcileDeniedKafkaOwners(deniedUsers config.DeniedUsers) *errors.ServiceError {
	if len(deniedUsers) < 1 {
		return nil
	}

	return k.kafkaService.DeprovisionKafkaForUsers(deniedUsers)
}

func (k *KafkaManager) reconcileAcceptedKafka(kafka *api.KafkaRequest) error {
	cluster, err := k.clusterPlmtStrategy.FindCluster(kafka)
	if err != nil {
		return fmt.Errorf("failed to find cluster for kafka request %s: %w", kafka.ID, err)
	}
	if cluster != nil {
		kafka.ClusterID = cluster.ClusterID
		if k.configService.GetConfig().Kafka.EnableQuotaService && kafka.SubscriptionId == "" {
			err := k.reconcileQuota(kafka)
			if err != nil {
				return err
			}
		}

		glog.Infof("Kafka instance with id %s is assigned to cluster with id %s", kafka.ID, kafka.ClusterID)

		kafka.Status = constants.KafkaRequestStatusPreparing.String()
		if err2 := k.kafkaService.Update(kafka); err2 != nil {
			return fmt.Errorf("failed to update kafka %s with cluster details: %w", kafka.ID, err2)
		}
	} else {
		glog.Warningf("No available cluster found for Kafka instance with id %s", kafka.ID)
	}
	return nil
}

// reserve: true creating the subscription, cluster_authorization is an idempotent endpoint. We will get the same subscription id for a KafkaRequest(id).
func (k *KafkaManager) reconcileQuota(kafka *api.KafkaRequest) error {
	// external_cluster_id and cluster_id both should be mapped with kafka ID.
	isAllowed, subscriptionId, err := k.quotaService.ReserveQuota("RHOSAKTrial", kafka.ID, kafka.ID, kafka.Owner, true, "single")
	if err != nil {
		return fmt.Errorf("failed to check quota for %s: %s", kafka.ID, err.Error())
	}
	if !isAllowed {
		kafka.FailedReason = "Insufficient quota"
		if executed, err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusFailed); executed && err != nil {
			return fmt.Errorf("failed to update kafka %s to status: %s", kafka.ID, err)
		}
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))

	}
	kafka.SubscriptionId = subscriptionId
	return nil
}

func (k *KafkaManager) reconcileDeprovisioningRequest(kafka *api.KafkaRequest) error {
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

func (k *KafkaManager) reconcilePreparedKafka(kafka *api.KafkaRequest) error {
	_, err := k.kafkaService.GetById(kafka.ID)
	if err != nil {
		sentry.CaptureException(err)
		return fmt.Errorf("failed to find kafka request %s: %w", kafka.ID, err)
	}

	if err := k.kafkaService.Create(kafka); err != nil {
		return k.handleKafkaRequestCreationError(kafka, err)
	}
	// consider the kafka in a provisioning state
	if executed, err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusProvisioning); executed && err != nil {
		return fmt.Errorf("failed to update kafka %s to status provisioning: %w", kafka.ID, err)
	}

	return nil
}

func (k *KafkaManager) reconcileProvisioningKafka(kafka *api.KafkaRequest) error {
	namespace, err := services.BuildNamespaceName(kafka)
	if err != nil {
		return err
	}
	kafkaState, err := k.observatoriumService.GetKafkaState(kafka.Name, namespace)
	if err != nil {
		sentry.CaptureException(fmt.Errorf("failed to get state from observatorium for kafka %s namespace %s cluster %s: %w", kafka.ID, namespace, kafka.ClusterID, err))
		return fmt.Errorf("failed to get state from observatorium for kafka %s namespace %s cluster %s: %w", kafka.ID, namespace, kafka.ClusterID, err)
	}
	if kafkaState.State == observatorium.ClusterStateReady {
		glog.Infof("Kafka %s state %s", kafka.ID, kafkaState.State)

		executed, err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusReady)
		if !executed {
			if err != nil {
				glog.Warningf("failed to update kafka %s to status ready: %s", kafka.ID, err.Reason)
			} else {
				glog.Warningf("failed to update kafka %s to status ready", kafka.ID)
			}
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to update kafka %s to status ready: %w", kafka.ID, err)
		}

		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		/**TODO if there is a creation failure, total operations needs to be incremented: this info. is not available at this time.*/
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))

		return nil
	}
	glog.V(1).Infof("reconciled kafka %s state %s", kafka.ID, kafkaState.State)
	return nil
}

func (k *KafkaManager) handleKafkaRequestCreationError(kafkaRequest *api.KafkaRequest, err *errors.ServiceError) error {
	// Important: we need to keep this check first as at the moment, the SSO-related errors also have http status code that is fall in the range of client errors.
	// So if this is not checked first, then err.IsClientErrorClass() will return true and that will fail the kafka instance creation as there is no retry if it's client errors.
	if err.IsFailedToCreateSSOClient() || err.IsFailedToGetSSOClient() || err.IsFailedToGetSSOClientSecret() || err.IsServerErrorClass() {
		durationSinceCreation := time.Since(kafkaRequest.CreatedAt)
		if durationSinceCreation > constants.KafkaMaxDurationWithProvisioningErrs {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
			kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
			kafkaRequest.FailedReason = err.Reason
			updateErr := k.kafkaService.Update(kafkaRequest)
			if updateErr != nil {
				return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
			}
			metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
			return fmt.Errorf("reached kafka %s max attempts", kafkaRequest.ID)
		}
	} else if err.IsClientErrorClass() {
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
		kafkaRequest.FailedReason = err.Reason
		updateErr := k.kafkaService.Update(kafkaRequest)
		if updateErr != nil {
			return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
		}
		metrics.KafkaRequestsStatusDurationMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
		sentry.CaptureException(err)
		return fmt.Errorf("error creating kafka %s: %w", kafkaRequest.ID, err)
	}

	return fmt.Errorf("failed to create kafka %s on cluster %s: %w", kafkaRequest.ID, kafkaRequest.ClusterID, err)
}

func (k *KafkaManager) reconcileSsoClientIDAndSecret(kafkaRequest *api.KafkaRequest) error {
	if kafkaRequest.SsoClientID == "" && kafkaRequest.SsoClientSecret == "" {
		kafkaRequest.SsoClientID = syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		secret, err := k.keycloakService.GetKafkaClientSecret(kafkaRequest.SsoClientID)
		if err != nil {
			return fmt.Errorf("failed to get sso client id & secret for kafka cluster: %s: %w", kafkaRequest.SsoClientID, err)
		}
		kafkaRequest.SsoClientSecret = secret
		if err = k.kafkaService.Update(kafkaRequest); err != nil {
			return fmt.Errorf("failed to update kafka %s with cluster details: %w", kafkaRequest.ID, err)
		}
	}
	return nil
}

func (k *KafkaManager) setKafkaRequestsStatusCountMetric(errors []error) []error {

	for _, status := range constants.AllKafkaStatus {

		requestedStatusList, serviceErr := k.kafkaService.ListByStatus(status)
		if serviceErr != nil {
			glog.Errorf("failed to list kafka %s requests: %s", status, serviceErr.Error())
			errors = append(errors, serviceErr)
		}
		metrics.KafkaRequestsStatusCountMetric(status, len(requestedStatusList))
	}

	return errors
}
