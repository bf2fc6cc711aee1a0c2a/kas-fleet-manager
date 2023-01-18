package kafka_mgrs

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	serviceErr "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// we do not add "deleted" status to the list as the kafkas are soft deleted once the status is set to "deleted", so no need to count them here.
var kafkaMetricsStatuses = []constants.KafkaStatus{
	constants.KafkaRequestStatusAccepted,
	constants.KafkaRequestStatusPreparing,
	constants.KafkaRequestStatusProvisioning,
	constants.KafkaRequestStatusReady,
	constants.KafkaRequestStatusDeprovision,
	constants.KafkaRequestStatusDeleting,
	constants.KafkaRequestStatusFailed,
	constants.KafkaRequestStatusSuspended,
	constants.KafkaRequestStatusSuspending,
	constants.KafkaRequestStatusResuming,
}

// KafkaManager represents a kafka manager that periodically reconciles kafka requests
type KafkaManager struct {
	workers.BaseWorker
	kafkaService            services.KafkaService
	clusterService          services.ClusterService
	accessControlListConfig *acl.AccessControlListConfig
	kafkaConfig             *config.KafkaConfig
	dataplaneClusterConfig  *config.DataplaneClusterConfig
	cloudProviders          *config.ProviderConfig
}

// NewKafkaManager creates a new kafka manager to reconcile kafkas
func NewKafkaManager(kafkaService services.KafkaService, accessControlList *acl.AccessControlListConfig, kafka *config.KafkaConfig, clusters *config.DataplaneClusterConfig, providers *config.ProviderConfig, reconciler workers.Reconciler, clusterService services.ClusterService) *KafkaManager {
	return &KafkaManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "general_kafka_worker",
			Reconciler: reconciler,
		},
		kafkaService:            kafkaService,
		accessControlListConfig: accessControlList,
		kafkaConfig:             kafka,
		dataplaneClusterConfig:  clusters,
		cloudProviders:          providers,
		clusterService:          clusterService,
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

	// get all kafkas and send their statuses to prometheus
	kafkas, err := k.kafkaService.ListAll()
	if err != nil {
		encounteredErrors = append(encounteredErrors, err)
	}

	for _, k := range kafkas {
		for _, s := range kafkaMetricsStatuses {
			if k.Status == s.String() {
				metrics.UpdateKafkaRequestsCurrentStatusInfoMetric(constants.KafkaStatus(k.Status), k.ID, k.ClusterID, 1.0)
			} else {
				metrics.UpdateKafkaRequestsCurrentStatusInfoMetric(constants.KafkaStatus(s), k.ID, k.ClusterID, 0.0)
			}
		}
	}

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setKafkaStatusCountMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	capacityError := k.updateClusterStatusCapacityMetrics()
	if capacityError != nil {
		encounteredErrors = append(encounteredErrors, capacityError)
	}

	// delete kafkas of denied owners
	accessControlListConfig := k.accessControlListConfig
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("Reconciling denied kafka owners")
		kafkaDeprovisioningForDeniedOwnersErr := k.reconcileDeniedKafkaOwners(accessControlListConfig.DenyList)
		if kafkaDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(kafkaDeprovisioningForDeniedOwnersErr, "failed to deprovision kafka for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// MGDSTRM-10012 temporarily reconcile updating the zero-value of ExpiredAt
	// for kafka requests
	updateErr := k.updateZeroValueOfKafkaRequestsExpiredAt()
	if updateErr != nil {
		encounteredErrors = append(encounteredErrors, updateErr)
		return encounteredErrors
	}

	// reconciles expires_at field for kafka instances
	updateExpiresAtErrors := k.reconcileKafkaExpiresAt(kafkas)
	if updateExpiresAtErrors != nil {
		wrappedError := errors.Wrap(updateExpiresAtErrors, "failed to update expires_at for kafka instances")
		encounteredErrors = append(encounteredErrors, wrappedError)
	}

	for _, kafka := range kafkas {
		if !kafka.CanBeAutomaticallySuspended() {
			// this kafka is not in a state that can be suspended
			continue
		}

		expired, remainingLifespan := kafka.IsExpired()
		if expired {
			// Expired kafkas will be deleted elsewhere
			continue
		}

		// get the billing model
		bm, err := k.kafkaConfig.GetBillingModelByID(kafka.InstanceType, kafka.ActualKafkaBillingModel)
		if err != nil {
			wrappedError := errors.Wrap(err, "failed to suspend expired Kafka instances")
			encounteredErrors = append(encounteredErrors, wrappedError)
			continue
		}

		if remainingLifespan.LessThanOrEqual(float64(bm.GracePeriodDays)) {
			glog.Infof("cluster with ID '%s' entered its grace period. Suspending", kafka.ID)
			// the instance is in grace period
			_, err := k.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusSuspending)
			if err != nil {
				wrappedError := errors.Wrap(err, "failed to suspend expired Kafka instances")
				encounteredErrors = append(encounteredErrors, wrappedError)
			}
		}
	}

	// cleaning up expired kafkas
	glog.Infoln("Deprovisioning expired kafkas")
	expiredKafkasError := k.kafkaService.DeprovisionExpiredKafkas()
	if expiredKafkasError != nil {
		wrappedError := errors.Wrap(expiredKafkasError, "failed to deprovision expired Kafka instances")
		encounteredErrors = append(encounteredErrors, wrappedError)
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

func (k *KafkaManager) updateClusterStatusCapacityMetrics() error {
	usedStreamingUnitsCountByRegion, err := k.clusterService.FindStreamingUnitCountByClusterAndInstanceType()
	if err != nil {
		return errors.Wrap(err, "failed to count Kafkas by region")
	}

	// always publish used capacity irrespective of scaling mode
	for _, usedStreamingUnitCount := range usedStreamingUnitsCountByRegion {
		if usedStreamingUnitCount.Status != api.ClusterAccepted.String() {
			metrics.UpdateClusterStatusCapacityUsedCount(usedStreamingUnitCount.CloudProvider, usedStreamingUnitCount.Region, usedStreamingUnitCount.InstanceType, usedStreamingUnitCount.ClusterId, float64(usedStreamingUnitCount.Count))
		}
	}

	if k.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		for _, usedStreamingUnitCount := range usedStreamingUnitsCountByRegion {
			if usedStreamingUnitCount.Status != api.ClusterAccepted.String() && usedStreamingUnitCount.ClusterType != api.EnterpriseDataPlaneClusterType.String() {
				availableAndMaxCapacityCounts, err := k.calculateAvailableAndMaxCapacityForDynamicScaling(usedStreamingUnitCount)
				if err != nil {
					return err
				}
				k.updateClusterStatusCapacityAvailableMetric(availableAndMaxCapacityCounts)
				k.updateClusterStatusCapacityMaxMetric(availableAndMaxCapacityCounts)
			}
		}
	}

	if k.dataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
		availableAndMaxCapacitiesCounts, err := k.calculateCapacityByRegionAndInstanceTypeForManualClusters(usedStreamingUnitsCountByRegion)
		if err != nil {
			return err
		}

		for _, availableAndMaxCapacityCounts := range availableAndMaxCapacitiesCounts {
			if availableAndMaxCapacityCounts.Status != api.ClusterAccepted.String() {
				k.updateClusterStatusCapacityAvailableMetric(availableAndMaxCapacityCounts)
				k.updateClusterStatusCapacityMaxMetric(availableAndMaxCapacityCounts)
			}
		}
	}

	// enterprise clusters max and available is handled separately
	enterpriseClustersAvailableAndMaxCapacityCounts := k.calculateAvailableAndMadCapacityForEnterpriseClusters(usedStreamingUnitsCountByRegion)
	for _, availableAndMaxCapacityCounts := range enterpriseClustersAvailableAndMaxCapacityCounts {
		k.updateClusterStatusCapacityAvailableMetric(availableAndMaxCapacityCounts)
		k.updateClusterStatusCapacityMaxMetric(availableAndMaxCapacityCounts)
	}

	return nil
}

func (k *KafkaManager) calculateAvailableAndMadCapacityForEnterpriseClusters(streamingUnitsByRegion []services.KafkaStreamingUnitCountPerCluster) []services.KafkaStreamingUnitCountPerCluster {
	var result []services.KafkaStreamingUnitCountPerCluster

	for _, cluster := range streamingUnitsByRegion {
		if cluster.ClusterType == api.EnterpriseDataPlaneClusterType.String() {
			result = append(result, services.KafkaStreamingUnitCountPerCluster{
				Region:        cluster.Region,
				InstanceType:  cluster.InstanceType,
				ClusterId:     cluster.ClusterId,
				Count:         int32(cluster.MaxUnits - cluster.Count),
				CloudProvider: cluster.CloudProvider,
				MaxUnits:      int32(cluster.MaxUnits),
			})
		}
	}
	return result
}

// calculateAvailableAndMaxCapacityForDynamicScaling takes in used capacity and compute available and max capacity by taking into consideration region limits and dynamic capacity info
// i.e MaxUnits value. Once the computation is completed, MaxUnits will indicate the maximum capacity which takes in region limits. And Count will indicate available capacity
func (k *KafkaManager) calculateAvailableAndMaxCapacityForDynamicScaling(streamingUnitsByRegion services.KafkaStreamingUnitCountPerCluster) (services.KafkaStreamingUnitCountPerCluster, error) {
	limit, err := k.getRegionInstanceTypeLimit(streamingUnitsByRegion.Region, streamingUnitsByRegion.CloudProvider, streamingUnitsByRegion.InstanceType)
	if err != nil {
		return services.KafkaStreamingUnitCountPerCluster{}, err
	}

	// The maximum available number of instances of this type is either the
	// cluster's own instance limit or the cloud provider's (depending on
	// which is reached sooner).
	maxAvailable := math.Min(float64(streamingUnitsByRegion.MaxUnits), limit)

	// A negative number could be the result of lowering the kafka instance limit after a higher number
	// of Kafkas have already been created. In this case we limit the metric to 0 anyways.
	available := math.Max(0, maxAvailable-float64(streamingUnitsByRegion.Count))

	return services.KafkaStreamingUnitCountPerCluster{
		Region:        streamingUnitsByRegion.Region,
		InstanceType:  streamingUnitsByRegion.InstanceType,
		ClusterId:     streamingUnitsByRegion.ClusterId,
		Count:         int32(available),
		CloudProvider: streamingUnitsByRegion.CloudProvider,
		MaxUnits:      int32(maxAvailable),
	}, nil
}

// calculateCapacityByRegionAndInstanceTypeForManualClusters compute the available capacity and maximum streaming unit capacity for each schedulable manual cluster,
// in a given cloud provider, region and for an instance type. Once the computation is done, Count will indicate the available capacity and MaxUnits will indicate the maximum streaming units
func (k *KafkaManager) calculateCapacityByRegionAndInstanceTypeForManualClusters(streamingUnitsByRegion []services.KafkaStreamingUnitCountPerCluster) ([]services.KafkaStreamingUnitCountPerCluster, error) {
	var result []services.KafkaStreamingUnitCountPerCluster

	// helper function that returns the sum of existing kafka instances for a clusterId split by
	// all existing on cluster vs. instance type existing on cluster
	findUsedCapacityForCluster := func(clusterId string, instanceType string) (int32, int32) {
		var totalUsed int32 = 0
		var instanceTypeUsed int32 = 0
		for _, kafkaInRegion := range streamingUnitsByRegion {
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

			limit, err := k.getRegionInstanceTypeLimit(cluster.Region, cluster.CloudProvider, instanceType)
			if err != nil {
				return nil, err
			}

			// The maximum available number of instances of this type is either the
			// cluster's own instance limit or the cloud provider's (depending on
			// which is reached sooner).
			maxAvailable := math.Min(float64(cluster.KafkaInstanceLimit), limit)

			// Current number of all instances and current number of given instance type (on the cluster)
			totalUsed, instanceTypeUsed := findUsedCapacityForCluster(cluster.ClusterId, instanceType)

			// Calculate how many more instances of a given type can be created:
			// min of ( total available for instance type, total available overall )
			availableByInstanceType := math.Min(maxAvailable-float64(instanceTypeUsed), float64(cluster.KafkaInstanceLimit)-float64(totalUsed))

			// A negative number could be the result of lowering the kafka instance limit after a higher number
			// of Kafkas have already been created. In this case we limit the metric to 0 anyways.
			if availableByInstanceType < 0 {
				availableByInstanceType = 0
			}

			result = append(result, services.KafkaStreamingUnitCountPerCluster{
				Region:        cluster.Region,
				InstanceType:  instanceType,
				ClusterId:     cluster.ClusterId,
				Count:         int32(availableByInstanceType),
				CloudProvider: cluster.CloudProvider,
				MaxUnits:      int32(maxAvailable),
			})
		}
	}

	// no scheduleable cluster or instance type unsupported
	return result, nil
}

func (k *KafkaManager) getRegionInstanceTypeLimit(region, cloudProvider, instanceType string) (float64, error) {
	instanceTypeLimit, err := k.cloudProviders.GetInstanceLimit(region, cloudProvider, instanceType)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get instance limit for %v on %v and instance type %v", region, cloudProvider, instanceType)
	}

	limit := math.MaxInt64
	if instanceTypeLimit != nil {
		limit = *instanceTypeLimit
	}

	return float64(limit), nil
}

func (k *KafkaManager) updateClusterStatusCapacityAvailableMetric(c services.KafkaStreamingUnitCountPerCluster) {
	metrics.UpdateClusterStatusCapacityAvailableCount(c.CloudProvider, c.Region, c.InstanceType, c.ClusterId, float64(c.Count))
}

func (k *KafkaManager) updateClusterStatusCapacityMaxMetric(c services.KafkaStreamingUnitCountPerCluster) {
	metrics.UpdateClusterStatusCapacityMaxCount(c.CloudProvider, c.Region, c.InstanceType, c.ClusterId, float64(c.MaxUnits))
}

func (k *KafkaManager) updateZeroValueOfKafkaRequestsExpiredAt() error {
	return k.kafkaService.UpdateZeroValueOfKafkaRequestsExpiredAt()
}

func (k *KafkaManager) reconcileKafkaExpiresAt(kafkas dbapi.KafkaList) serviceErr.ErrorList {
	logger.Logger.Infof("reconciling expiration date for kafka instances")
	var svcErrors serviceErr.ErrorList
	subscriptionStatusByOrgAndBillingModel := map[string]bool{}

	for _, kafka := range kafkas {
		logger.Logger.Infof("reconciling expires_at for kafka instance %q", kafka.ID)

		// skip update when Kafka is marked for deletion or is already being deleted
		if arrays.Contains(constants.GetDeletingStatuses(), kafka.Status) {
			logger.Logger.Infof("kafka %q is in %q state, skipping expires_at reconciliation", kafka.ID, kafka.Status)
			continue
		}

		instanceSize, err := k.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
		if err != nil {
			svcErrors = append(svcErrors, errors.Wrapf(err,
				"failed to get kafka instance size for %q with instance type %q and size id %q",
				kafka.ID, kafka.InstanceType, kafka.SizeId,
			))
			continue
		}

		// If lifespan seconds is not defined, expiration is determined based on the user/organisation's active subscription
		if instanceSize.LifespanSeconds == nil {
			logger.Logger.Infof("checking quota entitlement status for Kafka instance %q", kafka.ID)
			orgBillingModelID := fmt.Sprintf("%s-%s", kafka.OrganisationId, kafka.ActualKafkaBillingModel)
			active, ok := subscriptionStatusByOrgAndBillingModel[orgBillingModelID]
			if !ok {
				isActive, err := k.kafkaService.IsQuotaEntitlementActive(kafka)
				if err != nil {
					svcErrors = append(svcErrors, errors.Wrapf(err, "failed to get quota entitlement status of kafka instance %q", kafka.ID))
					continue
				}
				subscriptionStatusByOrgAndBillingModel[orgBillingModelID] = isActive
				active = isActive
			}

			if err := k.updateExpiresAtBasedOnQuotaEntitlement(kafka, active); err != nil {
				svcErrors = append(svcErrors, errors.Wrapf(err, "failed to update expires_at value based on quota entitlement for kafka instance %q", kafka.ID))
			}
		} else {
			// any Kafka instance types with lifespan seconds defined will have their expires_at value set on creation.
			// this is only temporary to ensure that the expires_at value is set for any Kafka instances created during the rollout of this change.
			if err := k.updateExpiresAtBasedOnLifespanSeconds(kafka, *instanceSize); err != nil {
				svcErrors = append(svcErrors, errors.Wrapf(err, "failed to update expires_at value based on lifespanSeconds for kafka instance %q", kafka.ID))
			}
		}
	}

	return svcErrors
}

// Updates expires_at field of the given Kafka instance based on the user/organisation's quota entitlement status
func (k *KafkaManager) updateExpiresAtBasedOnQuotaEntitlement(kafka *dbapi.KafkaRequest, isQuotaEntitlementActive bool) error {
	// if quota entitlement is active, ensure expires_at is set to null
	if isQuotaEntitlementActive && kafka.ExpiresAt.Valid {
		logger.Logger.Infof("updating expiration date of kafka instance %q to NULL", kafka.ID)
		return k.updateKafkaExpirationDate(kafka, nil)
	}

	// if quota entitlement is not active and expires_at is not already set, set its value based on the current time and grace period allowance
	// note that there is a temporary check for zero value of time.Time for cases where the kafka instance expires_at value
	// hasn't been updated to null yet after migration/rollout. This additional check can be removed once all Kafka instances
	// has been successfully migrated to the new expires_at type.
	if !isQuotaEntitlementActive && (!kafka.ExpiresAt.Valid || kafka.ExpiresAt.Time.IsZero()) {
		billingModel, err := k.kafkaConfig.GetBillingModelByID(kafka.InstanceType, kafka.ActualKafkaBillingModel)
		if err != nil {
			return err
		}

		// set expires_at to now + grace period days
		expiresAtTime := time.Now().AddDate(0, 0, billingModel.GracePeriodDays)
		logger.Logger.Infof("quota entitlement for kafka instance %q is no longer active, updating expires_at to %q", kafka.ID, expiresAtTime.Format(time.RFC1123Z))
		return k.updateKafkaExpirationDate(kafka, &expiresAtTime)
	}

	logger.Logger.Infof("no expires_at changes needed for kafka %q, skipping update", kafka.ID)
	return nil
}

// Updates expires_at field based on the lifespan seconds configured per Kafka instance type/size
func (k *KafkaManager) updateExpiresAtBasedOnLifespanSeconds(kafka *dbapi.KafkaRequest, instanceSize config.KafkaInstanceSize) error {
	// note that there is a temporary check for zero value of time.Time in cases where the kafka instance expires_at value
	// hasn't been updated to null yet after migration/rollout. This additional check can be removed once all Kafka instances
	// has been successfully migrated to the new expires_at type
	if !kafka.ExpiresAt.Valid || kafka.ExpiresAt.Time.IsZero() {
		// set expires_at to created_at + lifespanSeconds
		expiresAtTime := kafka.CreatedAt.Add(time.Duration(*instanceSize.LifespanSeconds) * time.Second)
		logger.Logger.Infof("lifespanSeconds defined for Kafka %q, updating expires_at to %q as it is not yet set", kafka.ID, expiresAtTime.Format(time.RFC1123Z))
		return k.updateKafkaExpirationDate(kafka, &expiresAtTime)
	}

	logger.Logger.Infof("lifespanSeconds defined for Kafka %q but expires_at is already set, skipping update", kafka.ID)
	return nil
}

// updates the expires_at field for the given Kafka instance
func (k *KafkaManager) updateKafkaExpirationDate(kafka *dbapi.KafkaRequest, expiresAtTime *time.Time) error {
	var expiresAt sql.NullTime
	if expiresAtTime != nil {
		expiresAt = sql.NullTime{Time: *expiresAtTime, Valid: true}
	}

	if err := k.kafkaService.Updates(kafka, map[string]interface{}{
		"expires_at": expiresAt,
	}); err != nil {
		return err
	}

	return nil
}
