package kafka_mgrs

import (
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	serviceErr "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"math"
	"strings"
)

// we do not add "deleted" status to the list as the kafkas are soft deleted once the status is set to "deleted", so no need to count them here.
var kafkaMetricsStatuses = []constants2.KafkaStatus{
	constants2.KafkaRequestStatusAccepted,
	constants2.KafkaRequestStatusPreparing,
	constants2.KafkaRequestStatusProvisioning,
	constants2.KafkaRequestStatusReady,
	constants2.KafkaRequestStatusDeprovision,
	constants2.KafkaRequestStatusDeleting,
	constants2.KafkaRequestStatusFailed,
}

// KafkaManager represents a kafka manager that periodically reconciles kafka requests
type KafkaManager struct {
	workers.BaseWorker
	kafkaService            services.KafkaService
	accessControlListConfig *acl.AccessControlListConfig
	kafkaConfig             *config.KafkaConfig
	dataplaneClusterConfig  *config.DataplaneClusterConfig
	cloudProviders          *config.ProviderConfig
}

// NewKafkaManager creates a new kafka manager
func NewKafkaManager(kafkaService services.KafkaService, accessControlList *acl.AccessControlListConfig, kafka *config.KafkaConfig, bus signalbus.SignalBus, clusters *config.DataplaneClusterConfig, providers *config.ProviderConfig) *KafkaManager {
	return &KafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "general_kafka_worker",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		kafkaService:            kafkaService,
		accessControlListConfig: accessControlList,
		kafkaConfig:             kafka,
		dataplaneClusterConfig:  clusters,
		cloudProviders:          providers,
	}
}

// Start initializes the kafka manager to reconcile kafka requests
func (k *KafkaManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka requests to stop.
func (k *KafkaManager) Stop() {
	k.StopWorker(k)
}

func (k *KafkaManager) Reconcile() []error {
	glog.Infoln("reconciling kafkas")
	var encounteredErrors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setKafkaStatusCountMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	statusErrors = k.setClusterStatusCapacityMetrics()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	// delete kafkas of denied owners
	accessControlListConfig := k.accessControlListConfig
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("reconciling denied kafka owners")
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(kafkaDeprovisioningForDeniedOwnersErr, "Failed to deprovision kafka for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// cleaning up expired qkafkas
	kafkaConfig := k.kafkaConfig
	if kafkaConfig.KafkaLifespan.EnableDeletionOfExpiredKafka {
		glog.Infoln("deprovisioning expired kafkas")
		expiredKafkasError := k.kafkaService.DeprovisionExpiredKafkas(kafkaConfig.KafkaLifespan.KafkaLifespanInHours)
		if expiredKafkasError != nil {
			wrappedError := errors.Wrap(expiredKafkasError, "failed to deprovision expired Kafka instances")
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	return encounteredErrors
}

func (k *KafkaManager) reconcileDeniedKafkaOwners(deniedUsers acl.DeniedUsers) *serviceErr.ServiceError {
	if len(deniedUsers) < 1 {
		return nil
	}

	return k.kafkaService.DeprovisionKafkaForUsers(deniedUsers)
}

func (k *KafkaManager) setKafkaStatusCountMetric() []error {
	counters, err := k.kafkaService.CountByStatus(kafkaMetricsStatuses)
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Kafkas by status")}
	}

	for _, c := range counters {
		metrics.UpdateKafkaRequestsStatusCountMetric(c.Status, c.Count)
	}

	return nil
}

func (k *KafkaManager) setClusterStatusCapacityMetrics() []error {
	kafkasByRegion, err := k.kafkaService.CountByRegionAndInstanceType()
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Kafkas by region")}
	}

	for _, region := range kafkasByRegion {
		used := float64(region.Count)
		metrics.UpdateClusterStatusCapacityUsedCount(region.CloudProvider, region.Region, region.InstanceType, region.ClusterId, used)
	}

	return k.setClusterStatusCapacityAvailableMetric(kafkasByRegion)
}

func (k *KafkaManager) setClusterStatusCapacityAvailableMetric(kafkasByRegion []services.KafkaRegionCount) []error {
	// helper function that returns the sum of existing kafka instances for a clusterId
	findUsedCapacityForCluster := func(clusterId string, instanceType string) (int, int) {
		totalUsed := 0
		instanceTypeUsed := 0
		for _, kafkaInRegion := range kafkasByRegion {
			if kafkaInRegion.ClusterId == clusterId {
				totalUsed += kafkaInRegion.Count

				if kafkaInRegion.InstanceType == instanceType {
					instanceTypeUsed += kafkaInRegion.Count
				}
			}
		}
		return totalUsed, instanceTypeUsed
	}

	for _, cluster := range k.dataplaneClusterConfig.ClusterConfig.GetManualClusters() {
		if !cluster.Schedulable {
			continue
		}

		supportedInstanceTypes := strings.Split(cluster.SupportedInstanceType, ",")
		for _, instanceType := range supportedInstanceTypes {
			if instanceType == "" {
				continue
			}

			instanceTypeLimit, err := k.cloudProviders.GetInstanceLimit(cluster.Region, cluster.CloudProvider, instanceType)
			if err != nil {
				return []error{errors.Wrapf(err, "failed to get instance limit for %v on %v and instance type %v",
					cluster.Region, cluster.CloudProvider, instanceType)}
			}

			var limit = math.MaxInt64
			if instanceTypeLimit != nil {
				limit = *instanceTypeLimit
			}

			// The maximum available number of instances of this type is either the
			// cluster's own instance limit or the cloud provider's (depending on
			// which is reached sooner).
			maxAvailable := math.Min(float64(cluster.KafkaInstanceLimit), float64(limit))

			totalUsed, instanceTypeUsed := findUsedCapacityForCluster(cluster.ClusterId, instanceType)

			// Calculate how many more instances of a given type can be created:
			// min of ( total available for instance type, total available overall )
			availableByInstanceType := math.Min(maxAvailable-float64(instanceTypeUsed), float64(cluster.KafkaInstanceLimit)-float64(totalUsed))
			metrics.UpdateClusterStatusCapacityAvailableCount(cluster.CloudProvider, cluster.Region, instanceType, cluster.ClusterId, availableByInstanceType)
		}
	}

	return nil
}
