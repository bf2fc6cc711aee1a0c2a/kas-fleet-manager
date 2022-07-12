package kafka_mgrs

import (
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"
)

// ProvisioningKafkaManager represents a kafka manager that periodically reconciles provisioning kafka requests.
type ProvisioningKafkaManager struct {
	workers.BaseWorker
	kafkaService             services.KafkaService
	clusterPlacementStrategy services.ClusterPlacementStrategy
}

// NewProvisioningKafkaManager creates a new kafka manager to reconcile provisioning kafkas.
func NewProvisioningKafkaManager(kafkaService services.KafkaService, reconciler workers.Reconciler, clusterPlacementStrategy services.ClusterPlacementStrategy) *ProvisioningKafkaManager {
	return &ProvisioningKafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "provisioning_kafka",
			Reconciler: reconciler,
		},
		kafkaService:             kafkaService,
		clusterPlacementStrategy: clusterPlacementStrategy,
	}
}

// Start initializes the kafka manager to reconcile provisioning kafka requests.
func (k *ProvisioningKafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling provisioning kafka requests to stop.
func (k *ProvisioningKafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *ProvisioningKafkaManager) Reconcile() []error {
	glog.Infoln("reconciling kafkas")
	var encounteredErrors []error

	// handle provisioning kafkas state.
	// Kafkas in a "provisioning" state means that it is ready to be sent to the KAS Fleetshard Operator for Kafka creation in the data plane cluster.
	// The update of the Kafka request status from 'provisioning' to another state will be handled by the KAS Fleetshard Operator.
	// We only need to update the metrics here.
	provisioningKafkas, serviceErr := k.kafkaService.ListByStatus(constants2.KafkaRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning kafkas"))
	} else {
		glog.Infof("provisioning kafkas count = %d", len(provisioningKafkas))
	}

	for _, kafka := range provisioningKafkas {
		glog.V(10).Infof("provisioning kafka id = %s", kafka.ID)
		if kafka.ClusterID == "" {
			if err := k.reassignProvisioningKafka(kafka); err != nil {
				encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile provisioning kafka %s", kafka.ID))
				continue
			}
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	}

	return encounteredErrors
}
func (k *ProvisioningKafkaManager) reassignProvisioningKafka(kafka *dbapi.KafkaRequest) error {
	cluster, e := k.clusterPlacementStrategy.FindCluster(kafka)
	if e != nil || cluster == nil {
		return errors.Errorf("Region %s cannot accept instance type: %s at this moment for kafka %s", kafka.Region, kafka.InstanceType, kafka.ID)
	}

	kafka.ClusterID = cluster.ClusterID

	err := k.kafkaService.AssignBootstrapServerHost(kafka)
	if err != nil {
		return errors.Wrapf(err, "error assigning bootstrap server host to kafka %s", kafka.ID)
	}

	latestStrimziVersion, latestStrimziVersionErr := cluster.GetLatestAvailableAndReadyStrimziVersion()
	if latestStrimziVersionErr != nil {
		return errors.Wrapf(latestStrimziVersionErr, "error finding ready strimzi versions for kafka %s", kafka.ID)
	}
	if latestStrimziVersion == nil {
		return errors.Errorf("could not find latest strimzi version for kafka %s", kafka.ID)
	}

	kafka.DesiredStrimziVersion = latestStrimziVersion.Version

	desiredKafkaVersion := latestStrimziVersion.GetLatestKafkaVersion()
	if desiredKafkaVersion == nil {
		return errors.Errorf("failed to get Kafka version for kafka %s", kafka.ID)
	}
	kafka.DesiredKafkaVersion = desiredKafkaVersion.Version

	desiredKafkaIBPVersion := latestStrimziVersion.GetLatestKafkaIBPVersion()
	if desiredKafkaIBPVersion == nil {
		return errors.Errorf("failed to get Kafka IBP version for kafka %s", kafka.ID)
	}
	kafka.DesiredKafkaIBPVersion = desiredKafkaIBPVersion.Version

	updateErr := k.kafkaService.Update(kafka)
	if updateErr != nil {
		return errors.Errorf("Failed to update kafka %s in provisioning state", kafka.ID)
	}
	return nil
}
