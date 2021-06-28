package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/google/uuid"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
)

// AcceptedKafkaManager represents a kafka manager that periodically reconciles kafka requests
type AcceptedKafkaManager struct {
	id                  string
	workerType          string
	isRunning           bool
	kafkaService        services.KafkaService
	configService       coreServices.ConfigService
	quotaServiceFactory coreServices.QuotaServiceFactory
	imStop              chan struct{}
	syncTeardown        sync.WaitGroup
	reconciler          workers.Reconciler
	clusterPlmtStrategy services.ClusterPlacementStrategy
}

// NewAcceptedKafkaManager creates a new kafka manager
func NewAcceptedKafkaManager(kafkaService services.KafkaService, configService coreServices.ConfigService, quotaServiceFactory coreServices.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, bus signalbus.SignalBus) *AcceptedKafkaManager {
	return &AcceptedKafkaManager{
		id:                  uuid.New().String(),
		workerType:          "accepted_kafka",
		kafkaService:        kafkaService,
		configService:       configService,
		quotaServiceFactory: quotaServiceFactory,
		clusterPlmtStrategy: clusterPlmtStrategy,
		reconciler: workers.Reconciler{
			SignalBus: bus,
		},
	}
}

func (k *AcceptedKafkaManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *AcceptedKafkaManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *AcceptedKafkaManager) GetID() string {
	return k.id
}

func (c *AcceptedKafkaManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *AcceptedKafkaManager) Start() {
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *AcceptedKafkaManager) Stop() {
	k.reconciler.Stop(k)
	metrics.ResetMetricsForKafkaManagers()
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *AcceptedKafkaManager) IsRunning() bool {
	return c.isRunning
}

func (c *AcceptedKafkaManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *AcceptedKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling accepted kafkas")
	var encounteredErrors []error

	// handle accepted kafkas
	acceptedKafkas, serviceErr := k.kafkaService.ListByStatus(constants.KafkaRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted kafkas"))
	} else {
		glog.Infof("accepted kafkas count = %d", len(acceptedKafkas))
	}

	for _, kafka := range acceptedKafkas {
		glog.V(10).Infof("accepted kafka id = %s", kafka.ID)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusAccepted, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		if err := k.reconcileAcceptedKafka(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted kafka %s", kafka.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedKafkaManager) reconcileAcceptedKafka(kafka *api.KafkaRequest) error {
	cluster, err := k.clusterPlmtStrategy.FindCluster(kafka)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for kafka request %s", kafka.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Kafka instance with id %s", kafka.ID)
		return nil
	}

	kafka.ClusterID = cluster.ClusterID
	if kafka.SubscriptionId == "" {
		shouldContinueProvision, err := k.reconcileQuota(kafka)
		if err != nil {
			return err
		}

		if !shouldContinueProvision {
			return nil
		}
	}

	glog.Infof("Kafka instance with id %s is assigned to cluster with id %s", kafka.ID, kafka.ClusterID)
	kafka.Status = constants.KafkaRequestStatusPreparing.String()
	if err2 := k.kafkaService.Update(kafka); err2 != nil {
		return errors.Wrapf(err2, "failed to update kafka %s with cluster details", kafka.ID)
	}
	return nil
}

// reserve: true creating the subscription, cluster_authorization is an idempotent endpoint. We will get the same subscription id for a KafkaRequest(id).
func (k *AcceptedKafkaManager) reconcileQuota(kafka *api.KafkaRequest) (bool, error) {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(kafka.QuotaType))
	if factoryErr != nil {
		return false, factoryErr
	}
	// external_cluster_id and cluster_id both should be mapped with kafka ID.
	subscriptionId, err := quotaService.ReserveQuota(kafka)
	if err != nil {
		if !err.InSufficientQuota() {
			return false, errors.Wrapf(err, "failed to check quota for %s", kafka.ID)
		}

		kafka.FailedReason = "Insufficient quota"
		kafka.Status = constants.KafkaRequestStatusFailed.String()
		if err := k.kafkaService.Update(kafka); err != nil {
			return false, errors.Wrapf(err, "failed to update kafka %s to failed status", kafka.ID)
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		logger.Logger.Warningf("Unable to provision kafka %s in %s dataplane cluster due to insufficient quota", kafka.ID, kafka.ClusterID)
		return false, nil
	}

	kafka.SubscriptionId = subscriptionId
	return true, nil
}
