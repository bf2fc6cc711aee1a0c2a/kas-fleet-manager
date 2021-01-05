package workers

import (
	"fmt"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services/syncsetresources"
	"strings"
	"sync"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
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

	// handle provisioning kafkas
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list accepted kafkas: %s", serviceErr.Error())
	}

	for _, kafka := range provisioningKafkas {
		if err := k.reconcileProvisionedKafka(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile accepted kafka %s: %s", kafka.ID, err.Error())
			continue
		}
	}

	// handle resource creation kafkas state
	resourceCreationKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusResourceCreation)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list resource creation kafkas: %s", serviceErr.Error())
	}

	for _, kafka := range resourceCreationKafkas {
		if err := k.reconcileResourceCreationKafka(kafka); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("reconcile resource creating %s: %s", kafka.ID, err.Error())
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
		kafka.Status = constants.KafkaRequestStatusProvisioning.String()
		if err = k.kafkaService.Update(kafka); err != nil {
			return fmt.Errorf("failed to update kafka %s with cluster details: %w", kafka.ID, err)
		}
	}
	return nil
}

func (k *KafkaManager) reconcileProvisionedKafka(kafka *api.KafkaRequest) error {
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
		if err.IsFailedToCreateSSOClient() {
			clientId := fmt.Sprintf("%s-%s", "kafka", strings.ToLower(kafka.ID))
			// todo retry logic for kafka client creation in mas sso
			keycloakErr := k.keycloakService.IsKafkaClientExist(clientId)
			if keycloakErr != nil {
				if updateErr := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusFailed); updateErr != nil {
					return fmt.Errorf("failed to update kafka %s to status: %w", kafka.ID, updateErr)
				}
				return fmt.Errorf("failed to create mas sso client for the kafka %s on cluster %s: %w", kafka.ID, kafka.ClusterID, err)
			}
		}

		return fmt.Errorf("failed to create kafka %s on cluster %s: %w", kafka.ID, kafka.ClusterID, err)
	}
	// consider the kafka in a resource creation state
	if err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusResourceCreation); err != nil {
		return fmt.Errorf("failed to update kafka %s to status resource creation: %w", kafka.ID, err)
	}

	return nil
}

func (k *KafkaManager) reconcileResourceCreationKafka(kafka *api.KafkaRequest) error {
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
		if err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusReady); err != nil {
			return fmt.Errorf("failed to update kafka %s to status ready: %w", kafka.ID, err)
		}
		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		return nil
	}
	glog.V(1).Infof("reconciled kafka %s state %s", kafka.ID, kafkaState.State)
	return nil
}
