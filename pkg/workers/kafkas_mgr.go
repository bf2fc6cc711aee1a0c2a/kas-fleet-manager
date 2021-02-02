package workers

import (
	"fmt"
	"sync"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services/syncsetresources"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

// KafkaManager represents a kafka manager that periodically reconciles kafka requests
type KafkaManager struct {
	id                   string
	workerType           string
	isRunning            bool
	ocmClient            ocm.Client
	clusterService       services.ClusterService
	kafkaService         services.KafkaService
	keycloakService      services.KeycloakService
	observatoriumService services.ObservatoriumService
	timer                *time.Timer
	imStop               chan struct{}
	syncTeardown         sync.WaitGroup
	reconciler           Reconciler
}

// NewKafkaManager creates a new kafka manager
func NewKafkaManager(kafkaService services.KafkaService, clusterService services.ClusterService, ocmClient ocm.Client, id string, keycloakService services.KeycloakService, observatoriumService services.ObservatoriumService) *KafkaManager {
	return &KafkaManager{
		id:                   id,
		workerType:           "kafka",
		ocmClient:            ocmClient,
		clusterService:       clusterService,
		kafkaService:         kafkaService,
		keycloakService:      keycloakService,
		observatoriumService: observatoriumService,
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

func (k *KafkaManager) reconcile() {
	glog.V(5).Infoln("reconciling kafkas")

	// handle deprovisioning requests
	deprovisioningRequests, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusDeprovision)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list kafka deprovisioning requests: %s", serviceErr.Error())
	}

	for _, kafka := range deprovisioningRequests {
		if err := k.reconcileDeprovisioningRequest(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile deprovisioning request %s: %s", kafka.ID, err.Error())
			continue
		}
	}

	// handle accepted kafkas
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
	}

	for _, kafka := range acceptedKafkas {
		if err := k.reconcileAcceptedKafka(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			continue
		}
	}

	// handle preparing kafkas
	preparingKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusPreparing)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
	}

	for _, kafka := range preparingKafkas {
		if err := k.reconcilePreparedKafka(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			continue
		}
	}

	// handle provisioning kafkas state
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list provisioning kafkas: %s", serviceErr.Error())
	}

	for _, kafka := range provisioningKafkas {
		if err := k.reconcileProvisioningKafka(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("reconcile provisioning %s: %s", kafka.ID, err.Error())
			continue
		}
	}

}

func (k *KafkaManager) reconcileAcceptedKafka(kafka *api.KafkaRequest) error {
	cluster, err := k.clusterService.FindCluster(services.FindClusterCriteria{
		Provider: kafka.CloudProvider,
		Region:   kafka.Region,
		MultiAZ:  kafka.MultiAZ,
		Status:   api.ClusterReady,
	})
	if err != nil {
		return fmt.Errorf("failed to find cluster for kafka request %s: %w", kafka.ID, err)
	}
	if cluster != nil {
		kafka.ClusterID = cluster.ClusterID
		kafka.Status = constants.KafkaRequestStatusPreparing.String()
		if err = k.kafkaService.Update(kafka); err != nil {
			return fmt.Errorf("failed to update kafka %s with cluster details: %w", kafka.ID, err)
		}
	}
	return nil
}

func (k *KafkaManager) reconcileDeprovisioningRequest(kafka *api.KafkaRequest) error {
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

	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		clientName := syncsetresources.BuildKeycloakClientNameIdentifier(kafka.ID)
		keycloakSecret, err := k.keycloakService.RegisterKafkaClientInSSO(clientName, kafka.OrganisationId)
		if err != nil || keycloakSecret == "" {
			return fmt.Errorf("failed to create sso client %s: %w", kafka.ID, err)
		}
	}

	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)

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
		sentry.CaptureException(err)
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
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		return nil
	}
	glog.V(1).Infof("reconciled kafka %s state %s", kafka.ID, kafkaState.State)
	return nil
}

func (k *KafkaManager) handleKafkaRequestCreationError(kafkaRequest *api.KafkaRequest, err *errors.ServiceError) error {
	if err.IsClientErrorClass() {
		kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
		kafkaRequest.FailedReason = err.Reason
		updateErr := k.kafkaService.Update(kafkaRequest)
		if updateErr != nil {
			return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
		}
		sentry.CaptureException(err)
		return fmt.Errorf("error creating kafka %s: %w", kafkaRequest.ID, err)
	}

	if err.IsServerErrorClass() {
		durationSinceCreation := time.Since(kafkaRequest.CreatedAt)
		if durationSinceCreation > constants.KafkaMaxDurationWithProvisioningErrs {
			kafkaRequest.Status = string(constants.KafkaRequestStatusFailed)
			kafkaRequest.FailedReason = err.Reason
			updateErr := k.kafkaService.Update(kafkaRequest)
			if updateErr != nil {
				return fmt.Errorf("failed to update kafka %s: %w", kafkaRequest.ID, err)
			}

			return fmt.Errorf("reached kafka %s max attempts", kafkaRequest.ID)
		}
	}

	if err.IsFailedToCreateSSOClient() {
		clientName := syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		// todo retry logic for kafka client creation in mas sso
		keycloakErr := k.keycloakService.IsKafkaClientExist(clientName)
		if keycloakErr != nil {
			if executed, updateErr := k.kafkaService.UpdateStatus(kafkaRequest.ID, constants.KafkaRequestStatusFailed); executed && updateErr != nil {
				return fmt.Errorf("failed to update kafka %s to status: %w", kafkaRequest.ID, updateErr)
			}
			return fmt.Errorf("failed to create mas sso client for the kafka %s on cluster %s: %w", kafkaRequest.ID, kafkaRequest.ClusterID, err)
		}
	}

	return fmt.Errorf("failed to create kafka %s on cluster %s: %w", kafkaRequest.ID, kafkaRequest.ClusterID, err)
}
