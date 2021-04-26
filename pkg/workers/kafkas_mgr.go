package workers

import (
	"fmt"
	"sync"
	"time"

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
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *KafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *KafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *KafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *KafkaManager) reconcile() []error {
	glog.Infoln("reconciling kafkas")
	var errors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setKafkaStatusCountMetric()
	if len(statusErrors) > 0 {
		errors = append(errors, statusErrors...)
	}

	accessControlListConfig := k.configService.GetConfig().AccessControlList
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("reconciling denied kafka owners")
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			glog.Errorf("Failed to deprovision kafka for denied owners %s: %s", accessControlListConfig.DenyList, kafkaDeprovisioningForDeniedOwnersErr.Error())
			errors = append(errors, kafkaDeprovisioningForDeniedOwnersErr)
		}
	}

	// cleaning up expired kafkas
	kafkaConfig := k.configService.GetConfig().Kafka
	if kafkaConfig.KafkaLifespan.EnableDeletionOfExpiredKafka {
		glog.Infoln("deprovisioning expired kafkas")
		expiredKafkasError := k.kafkaService.DeprovisionExpiredKafkas(kafkaConfig.KafkaLifespan.KafkaLifespanInHours)
		if expiredKafkasError != nil {
			glog.Errorf("failed to deprovision expired Kafka instances due to error: %s", expiredKafkasError.Error())
			errors = append(errors, expiredKafkasError)
		}
	}

	// handle deleting kafka requests
	// Kafkas in a "deleting" state have been removed, along with all their resources (i.e. ManagedKafka, Kafka CRs),
	// from the data plane cluster. This reconcile phase ensures that any other dependencies (i.e. SSO clients, CNAME records)
	// are cleaned up for these Kafkas and their records soft deleted from the database.

	// The "deleted" status has been replaced by "deleting" and should be removed soon. We need to list both status here to keep backward compatibility.
	deprovisionStatus := []constants.KafkaStatus{constants.KafkaRequestStatusDeleting, constants.KafkaRequestStatusDeleted}
	deprovisioningRequests, serviceErr := k.kafkaService.ListByStatus(deprovisionStatus...)
	if serviceErr != nil {
		glog.Errorf("failed to list deleting kafka requests: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("%s kafkas count = %d", deprovisionStatus[0].String(), len(deprovisioningRequests))
	}

	for _, kafka := range deprovisioningRequests {
		glog.V(10).Infof("deleting kafka id = %s", kafka.ID)
		if err := k.reconcileDeletedKafkas(kafka); err != nil {
			glog.Errorf("failed to reconcile deleting request %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	// handle accepted kafkas
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("accepted kafkas count = %d", len(acceptedKafkas))
	}

	for _, kafka := range acceptedKafkas {
		glog.V(10).Infof("accepted kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusAccepted, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
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
	} else {
		glog.Infof("preparing kafkas count = %d", len(preparingKafkas))
	}

	for _, kafka := range preparingKafkas {
		glog.V(10).Infof("preparing kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusPreparing, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcilePreparingKafka(kafka); err != nil {
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			errors = append(errors, err)
			continue
		}

	}

	readyKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusReady)
	if serviceErr != nil {
		glog.Errorf("failed to list ready kafkas: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("ready kafkas count = %d", len(readyKafkas))
	}
	if k.configService.GetConfig().Keycloak.EnableAuthenticationOnKafka {
		for _, kafka := range readyKafkas {
			if kafka.Status == string(constants.KafkaRequestStatusReady) {
				glog.V(10).Infof("ready kafka id = %s", kafka.ID)
				if err := k.reconcileSsoClientIDAndSecret(kafka); err != nil {
					glog.Errorf("failed to get provisioning kafkas sso client%s: %s", kafka.SsoClientID, err.Error())
					errors = append(errors, err)
					continue
				}
			}
		}
	}

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
		kafka.Status = constants.KafkaRequestStatusFailed.String()
		if err := k.kafkaService.Update(kafka); err != nil {
			return fmt.Errorf("failed to update kafka %s to status: %s", kafka.ID, err)
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))

	}
	kafka.SubscriptionId = subscriptionId
	return nil
}

func (k *KafkaManager) reconcileDeletedKafkas(kafka *api.KafkaRequest) error {
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

func (k *KafkaManager) reconcilePreparingKafka(kafka *api.KafkaRequest) error {
	if err := k.kafkaService.PrepareKafkaRequest(kafka); err != nil {
		return k.handleKafkaRequestCreationError(kafka, err)
	}

	return nil
}

func (k *KafkaManager) handleKafkaRequestCreationError(kafkaRequest *api.KafkaRequest, err *errors.ServiceError) error {
	if err.IsServerErrorClass() {
		// retry the kafka creation request only if the failure is caused by server errors
		// and the time elapsed since its db record was created is still within the threshold.
		durationSinceCreation := time.Since(kafkaRequest.CreatedAt)
		if durationSinceCreation > constants.KafkaMaxDurationWithProvisioningErrs {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
			kafkaRequest.Status = constants.KafkaRequestStatusFailed.String()
			kafkaRequest.FailedReason = err.Reason
			updateErr := k.kafkaService.Update(kafkaRequest)
			if updateErr != nil {
				glog.Errorf("Failed to update kafka %s in failed state due to %s. Kafka failed reason %s", kafkaRequest.ID, updateErr.Error(), kafkaRequest.FailedReason)
				return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
			}
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
			glog.Errorf("Kafka %s is in server error failed state due to %s. Maximum attempts has been reached", kafkaRequest.ID, err.Error())
			return fmt.Errorf("reached kafka %s max attempts", kafkaRequest.ID)
		}
	} else if err.IsClientErrorClass() {
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		kafkaRequest.Status = constants.KafkaRequestStatusFailed.String()
		kafkaRequest.FailedReason = err.Reason
		updateErr := k.kafkaService.Update(kafkaRequest)
		if updateErr != nil {
			glog.Errorf("Failed to update kafka %s in failed state due to %s. Kafka failed reason %s", kafkaRequest.ID, updateErr.Error(), kafkaRequest.FailedReason)
			return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
		sentry.CaptureException(err)
		glog.Errorf("Kafka %s is in a client error failed state due to %s. ", kafkaRequest.ID, err.Error())
		return fmt.Errorf("error creating kafka %s: %w", kafkaRequest.ID, err)
	}

	glog.Errorf("Kafka %s is in failed state due to %s", kafkaRequest.ID, err.Error())
	return fmt.Errorf("failed to create kafka %s on cluster %s: %w", kafkaRequest.ID, kafkaRequest.ClusterID, err)
}

func (k *KafkaManager) reconcileSsoClientIDAndSecret(kafkaRequest *api.KafkaRequest) error {
	if kafkaRequest.SsoClientID == "" && kafkaRequest.SsoClientSecret == "" {
		kafkaRequest.SsoClientID = services.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
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

func (k *KafkaManager) setKafkaStatusCountMetric() []error {
	// we do not add "deleted" status to the list as the kafkas are soft deleted once the status is set to "deleted", so no need to count them here.
	status := []constants.KafkaStatus{
		constants.KafkaRequestStatusAccepted,
		constants.KafkaRequestStatusPreparing,
		constants.KafkaRequestStatusProvisioning,
		constants.KafkaRequestStatusReady,
		constants.KafkaRequestStatusDeprovision,
		constants.KafkaRequestStatusDeleting,
		constants.KafkaRequestStatusFailed,
	}
	var errors []error
	if counters, err := k.kafkaService.CountByStatus(status); err != nil {
		glog.Errorf("failed to count Kafkas by status: %s", err.Error())
		errors = append(errors, err)
	} else {
		for _, c := range counters {
			metrics.UpdateKafkaRequestsStatusCountMetric(c.Status, c.Count)
		}
	}

	return errors
}
