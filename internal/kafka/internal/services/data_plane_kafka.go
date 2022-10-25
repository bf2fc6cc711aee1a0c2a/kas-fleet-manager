package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type managedKafkaStatus string

func (ks managedKafkaStatus) String() string {
	return string(ks)
}

type managedKafkaDeploymentType string

func (dt managedKafkaDeploymentType) String() string {
	return string(dt)
}

type totalCountPerReservedManagedKafkaDeploymentStatus map[managedKafkaStatus]int
type reservedManagedKafkaStatusCountPerInstanceType map[api.ClusterInstanceTypeSupport]totalCountPerReservedManagedKafkaDeploymentStatus

const (
	statusInstalling          managedKafkaStatus         = "installing"
	statusReady               managedKafkaStatus         = "ready"
	statusError               managedKafkaStatus         = "error"
	statusRejected            managedKafkaStatus         = "rejected"
	statusRejectedClusterFull managedKafkaStatus         = "rejectedClusterFull"
	statusDeleted             managedKafkaStatus         = "deleted"
	statusUnknown             managedKafkaStatus         = "unknown"
	statusSuspended           managedKafkaStatus         = "suspended"
	strimziUpdating           string                     = "StrimziUpdating"
	kafkaUpdating             string                     = "KafkaUpdating"
	kafkaIBPUpdating          string                     = "KafkaIbpUpdating"
	reservedDeploymentType    managedKafkaDeploymentType = "reserved"
	realDeploymentType        managedKafkaDeploymentType = "real"
)

//go:generate moq -out data_plane_kafka_service_moq.go . DataPlaneKafkaService
type DataPlaneKafkaService interface {
	UpdateDataPlaneKafkaService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError
}

type dataPlaneKafkaService struct {
	kafkaService   KafkaService
	clusterService ClusterService
	kafkaConfig    *config.KafkaConfig
}

func NewDataPlaneKafkaService(kafkaSrv KafkaService, clusterSrv ClusterService, kafkaConfig *config.KafkaConfig) *dataPlaneKafkaService {
	return &dataPlaneKafkaService{
		kafkaService:   kafkaSrv,
		clusterService: clusterSrv,
		kafkaConfig:    kafkaConfig,
	}
}

func (d *dataPlaneKafkaService) UpdateDataPlaneKafkaService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError {
	cluster, err := d.clusterService.FindClusterByID(clusterID)
	log := logger.NewUHCLogger(ctx)
	if err != nil {
		return err
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return serviceError.BadRequest("cluster id %s not found", clusterID)
	}

	prewarmingStatusInfo := reservedManagedKafkaStatusCountPerInstanceType{}
	supportedInstanceTypes := cluster.GetSupportedInstanceTypes()

	// prefill prewarming status info with empty count
	for _, supportedInstanceType := range supportedInstanceTypes {
		// sets all the statuses to 0 to allow for the count to drop
		prewarmingStatusInfo[api.ClusterInstanceTypeSupport(supportedInstanceType)] = totalCountPerReservedManagedKafkaDeploymentStatus{
			statusReady:               0,
			statusError:               0,
			statusUnknown:             0,
			statusDeleted:             0,
			statusRejected:            0,
			statusInstalling:          0,
			statusRejectedClusterFull: 0,
		}
	}

	for _, ks := range status {
		managedKafkaDeploymentType := d.getManagedKafkaDeploymentType(ks)
		switch managedKafkaDeploymentType {
		case realDeploymentType:
			d.processRealKafkaDeployment(ks, cluster, log)
		case reservedDeploymentType:
			d.processReservedKafkaDeployment(ks, prewarmingStatusInfo, log, clusterID)
		}
	}

	// emits the kas_fleet_manager_prewarmed_kafka_instances metrics
	for instanceType, instanceTypePrewarmingStatusInfo := range prewarmingStatusInfo {
		for status, count := range instanceTypePrewarmingStatusInfo {
			metrics.UpdateClusterPrewarmingStatusInfoCountMetric(metrics.PrewarmingStatusInfo{
				ClusterID:    cluster.ClusterID,
				InstanceType: instanceType.String(),
				Status:       status.String(),
				Count:        count,
			})
		}
	}

	return nil
}

// processRealKafkaDeployment process reserved kafka instances and updates their status in cluster
func (d *dataPlaneKafkaService) processReservedKafkaDeployment(ks *dbapi.DataPlaneKafkaStatus, prewarmingStatusInfo reservedManagedKafkaStatusCountPerInstanceType, log logger.UHCLogger, clusterID string) {
	parsedID := strings.Split(ks.KafkaClusterId, "-")
	if len(parsedID) < 4 {
		log.Error(fmt.Errorf("The reserved %q ID does not follow the format 'reserved-kafka-<instance-type>-<number>'", ks.KafkaClusterId))
		return
	}

	instanceType := parsedID[2]
	_, ok := prewarmingStatusInfo[api.ClusterInstanceTypeSupport(instanceType)]
	if !ok {
		log.Error(fmt.Errorf("The reserved %q ID is not supported in the cluster with cluster_id %q", ks.KafkaClusterId, clusterID))
		return
	}

	reservedManagedKafkaStatus := d.getManagedKafkaStatus(ks)
	prewarmingStatusInfo[api.ClusterInstanceTypeSupport(instanceType)][reservedManagedKafkaStatus] += 1
}

// processRealKafkaDeployment process real kafka instances and updates their status and stores other info coming data plane
func (d *dataPlaneKafkaService) processRealKafkaDeployment(ks *dbapi.DataPlaneKafkaStatus, cluster *api.Cluster, log logger.UHCLogger) {
	kafka, getErr := d.kafkaService.GetById(ks.KafkaClusterId)
	if getErr != nil {
		glog.Error(errors.Wrapf(getErr, "failed to get kafka cluster by id %s", ks.KafkaClusterId))
		return
	}
	if kafka.ClusterID != cluster.ClusterID {
		log.Warningf("clusterId for kafka cluster %s does not match clusterId. kafka clusterId = %s :: clusterId = %s", kafka.ID, kafka.ClusterID, cluster.ClusterID)
		return
	}

	// Notes on state transitions
	//  - 'suspending' state can only be set by an admin user from a 'ready' state via the /admin/kafkas/ endpoint.
	//     This must only transition to 'resuming', 'suspended' or 'deprovision'.
	//     - FSO will only send the status 'suspended' if the Kafka instance was already in a 'suspending' or 'suspended' state.
	//     - FSO will not change the status of the ManagedKafka CR prior to transitioning to 'suspended' unless an error occurs.
	//       e.g. If the Kafka instance was in a 'ready' state previously, FSO will keep reporting its status as 'ready' until it
	//            finally reports 'suspended'. If an error occurs, the 'Ready' condition of the CR will have the values 'Status=False,Reason=Error'.
	//  - 'resuming' state can only be set by an admin user from a 'suspending' or 'suspended' state via the /admin/kafkas/ endpoint.
	//     This must only transition to 'ready', 'failed' or 'deprovision'.
	//  - 'suspended' state must only transition to 'resuming' or 'deprovision'.
	//  - 'deprovision' state is set by the user. This must only transition to 'deleting' or 'failed'.
	//     - FSO will only send the status 'deleted' if the Kafka instance was already in a 'deprovisioning' state.
	//  - 'failed' (or 'error') state may occur at any time.
	//     - KFM must never transition a Kafka instance to 'failed' from 'suspending' and 'suspended' states.
	//  - Routes should only be created once. They will remain uncahnged and will continue to be published in the status even if the Kafka instance was suspended.
	var e *serviceError.ServiceError
	switch s := d.getManagedKafkaStatus(ks); s {
	case statusReady:
		if kafka.Status != constants.KafkaRequestStatusSuspending.String() && kafka.Status != constants.KafkaRequestStatusSuspended.String() {
			// Store the routes (and create them) when Kafka is ready. By the time it is ready, the routes should definitely be there.
			e = d.persistKafkaRoutes(kafka, ks, cluster)
			if e == nil {
				kafka.AdminApiServerURL = ks.AdminServerURI
				e = d.setKafkaClusterReady(kafka)
			}
		}
	case statusInstalling:
		// Store the routes (and create them) if they are available at this stage to lessen the length of time taken to provision the Kafka.
		// The routes list will either be empty or complete.
		e = d.persistKafkaRoutes(kafka, ks, cluster)
	case statusError:
		// when getStatus returns statusError we know that the ready
		// condition will be there so there's no need to check for it
		readyCondition, _ := ks.GetReadyCondition()
		// Do not store the error in the KafkaRequest object as this will be seen by the end user when the Kafka instance is in a 'suspended'
		// or 'suspending' state. This is not actionable by the user. This error will be logged and captured in Sentry instead.
		if kafka.Status != constants.KafkaRequestStatusSuspending.String() && kafka.Status != constants.KafkaRequestStatusSuspended.String() {
			e = d.setKafkaClusterFailed(kafka, readyCondition.Message)
		} else {
			log.Errorf("kafka %q with status %q received errors from data plane: %q", kafka.ID, kafka.Status, readyCondition.Message)
		}
	case statusDeleted:
		e = d.setKafkaClusterDeleting(kafka)
	case statusRejected:
		e = d.reassignKafkaCluster(kafka)
	case statusRejectedClusterFull:
		e = d.unassignKafkaFromDataplaneCluster(kafka)
	case statusSuspended:
		if kafka.Status == constants.KafkaRequestStatusSuspending.String() {
			logger.Logger.Infof("updating status of kafka %q from %q to %q", kafka.ID, kafka.Status, constants.KafkaRequestStatusSuspended)
			_, e = d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusSuspended)
		}
	case statusUnknown:
		log.Infof("kafka cluster %s status is unknown", ks.KafkaClusterId)
	default:
		log.V(5).Infof("kafka cluster %s is still installing", ks.KafkaClusterId)
	}
	if e != nil {
		log.Error(errors.Wrapf(e, "Error updating kafka %s status", ks.KafkaClusterId))
	}

	e = d.setKafkaRequestVersionFields(kafka, ks)
	if e != nil {
		log.Error(errors.Wrapf(e, "Error updating kafka '%s' version fields", ks.KafkaClusterId))
	}
}

func (d *dataPlaneKafkaService) setKafkaClusterReady(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	if !kafka.RoutesCreated {
		logger.Logger.V(10).Infof("routes for kafka %s are not created", kafka.ID)
		return nil
	} else {
		logger.Logger.Infof("routes for kafka %s are created", kafka.ID)
	}
	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants.KafkaRequestStatusProvisioning)

	if err != nil {
		return err
	}

	err = d.kafkaService.Updates(kafka, map[string]interface{}{"admin_api_server_url": kafka.AdminApiServerURL, "failed_reason": "", "status": constants.KafkaRequestStatusReady.String()})
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update kafka cluster %s", kafka.ID)
	}

	if shouldSendMetric {
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	}

	return nil
}

func (d *dataPlaneKafkaService) setKafkaRequestVersionFields(kafka *dbapi.KafkaRequest, status *dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError {
	needsUpdate := false
	prevActualKafkaVersion := kafka.ActualKafkaVersion
	if status.KafkaVersion != "" && status.KafkaVersion != kafka.ActualKafkaVersion {
		logger.Logger.Infof("Updating Kafka version for Kafka ID '%s' from '%s' to '%s'", kafka.ID, prevActualKafkaVersion, status.KafkaVersion)
		kafka.ActualKafkaVersion = status.KafkaVersion
		needsUpdate = true
	}

	prevActualKafkaIBPVersion := kafka.ActualKafkaIBPVersion
	if status.KafkaIBPVersion != "" && status.KafkaIBPVersion != kafka.ActualKafkaIBPVersion {
		logger.Logger.Infof("Updating Kafka IBP version for Kafka ID '%s' from '%s' to '%s'", kafka.ID, prevActualKafkaIBPVersion, status.KafkaIBPVersion)
		kafka.ActualKafkaIBPVersion = status.KafkaIBPVersion
		needsUpdate = true
	}

	prevActualStrimziVersion := kafka.ActualStrimziVersion
	if status.StrimziVersion != "" && status.StrimziVersion != kafka.ActualStrimziVersion {
		logger.Logger.Infof("Updating Strimzi version for Kafka ID '%s' from '%s' to '%s'", kafka.ID, prevActualStrimziVersion, status.StrimziVersion)
		kafka.ActualStrimziVersion = status.StrimziVersion
		needsUpdate = true
	}

	readyCondition, found := status.GetReadyCondition()
	if found {
		// TODO is this really correct? What happens if there is a StrimziUpdating reason
		// but the 'status' is false? What does that mean and how should we behave?
		prevStrimziUpgrading := kafka.StrimziUpgrading
		strimziUpdatingReasonIsSet := readyCondition.Reason == strimziUpdating
		if strimziUpdatingReasonIsSet && !prevStrimziUpgrading {
			logger.Logger.Infof("Strimzi version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevStrimziUpgrading, strimziUpdatingReasonIsSet)
			kafka.StrimziUpgrading = true
			needsUpdate = true
		}
		if !strimziUpdatingReasonIsSet && prevStrimziUpgrading {
			logger.Logger.Infof("Strimzi version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevStrimziUpgrading, strimziUpdatingReasonIsSet)
			kafka.StrimziUpgrading = false
			needsUpdate = true
		}

		prevKafkaUpgrading := kafka.KafkaUpgrading
		kafkaUpdatingReasonIsSet := readyCondition.Reason == kafkaUpdating
		if kafkaUpdatingReasonIsSet && !prevKafkaUpgrading {
			logger.Logger.Infof("Kafka version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevKafkaUpgrading, kafkaUpdatingReasonIsSet)
			kafka.KafkaUpgrading = true
			needsUpdate = true
		}
		if !kafkaUpdatingReasonIsSet && prevKafkaUpgrading {
			logger.Logger.Infof("Kafka version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevKafkaUpgrading, kafkaUpdatingReasonIsSet)
			kafka.KafkaUpgrading = false
			needsUpdate = true
		}

		prevKafkaIBPUpgrading := kafka.KafkaIBPUpgrading
		kafkaIBPUpdatingReasonIsSet := readyCondition.Reason == kafkaIBPUpdating
		if kafkaIBPUpdatingReasonIsSet && !prevKafkaIBPUpgrading {
			logger.Logger.Infof("Kafka IBP version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevKafkaIBPUpgrading, kafkaIBPUpdatingReasonIsSet)
			kafka.KafkaIBPUpgrading = true
			needsUpdate = true
		}
		if !kafkaIBPUpdatingReasonIsSet && prevKafkaIBPUpgrading {
			logger.Logger.Infof("Kafka IBP version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevKafkaIBPUpgrading, kafkaIBPUpdatingReasonIsSet)
			kafka.KafkaIBPUpgrading = false
			needsUpdate = true
		}

	}

	if needsUpdate {
		versionFields := map[string]interface{}{
			"actual_strimzi_version":   kafka.ActualStrimziVersion,
			"actual_kafka_version":     kafka.ActualKafkaVersion,
			"actual_kafka_ibp_version": kafka.ActualKafkaIBPVersion,
			"strimzi_upgrading":        kafka.StrimziUpgrading,
			"kafka_upgrading":          kafka.KafkaUpgrading,
			"kafka_ibp_upgrading":      kafka.KafkaIBPUpgrading,
		}

		if err := d.kafkaService.Updates(kafka, versionFields); err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update actual version fields for kafka cluster %s", kafka.ID)
		}
	}

	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterFailed(kafka *dbapi.KafkaRequest, errMessage string) *serviceError.ServiceError {
	// if kafka was already reported as failed we don't do anything
	if kafka.Status == string(constants.KafkaRequestStatusFailed) {
		return nil
	}

	logger.Logger.Errorf("Kafka status for Kafka ID '%s' in ClusterID '%s' reported as failed by KAS Fleet Shard Operator: '%s'", kafka.ID, kafka.ClusterID, errMessage)

	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants.KafkaRequestStatusProvisioning)
	if err != nil {
		return err
	}

	kafka.Status = string(constants.KafkaRequestStatusFailed)
	kafka.FailedReason = "Kafka reported as failed from the data plane"
	err = d.kafkaService.Update(kafka)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update kafka cluster to %s status for kafka cluster %s", constants.KafkaRequestStatusFailed, kafka.ID)
	}
	if shouldSendMetric {
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	}

	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterDeleting(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	// If the Kafka cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for kafka cluster %s", constants.KafkaRequestStatusDeleting, kafka.ID)
		} else {
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusDeleting, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		}
	}
	return nil
}

// reassigns a Kafka instance to another data plane cluster. It only reassigns Kafka instances in a 'provisioning' state.
func (d *dataPlaneKafkaService) reassignKafkaCluster(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	if kafka.Status == constants.KafkaRequestStatusProvisioning.String() {
		// If a Kafka cluster is rejected by the kas-fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the kas-fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for kafka clusters if there are multiple OSD clusters and the value here can be the key of that table
		kafka.PlacementId = api.NewID()
		if err := d.kafkaService.Update(kafka); err != nil {
			return err
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	} else {
		logger.Logger.Infof("kafka cluster %s is rejected and current status is %s", kafka.ID, kafka.Status)
	}

	return nil
}

// unassigns a Kafka instance from a data plane cluster. This is only done for Kafka instances in a 'provisioning' state.
func (d *dataPlaneKafkaService) unassignKafkaFromDataplaneCluster(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	if kafka.Status == constants.KafkaRequestStatusProvisioning.String() {
		logger.Logger.Infof("kafka %s is being unassigned from cluster %s", kafka.ID, kafka.ClusterID)
		if err := d.kafkaService.Updates(kafka, map[string]interface{}{
			"cluster_id":                "",
			"bootstrap_server_host":     "",
			"desired_strimzi_version":   "",
			"desired_kafka_version":     "",
			"desired_kafka_ibp_version": "",
		}); err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to reset fields for kafka cluster %s", kafka.ID)
		}

		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	} else {
		logger.Logger.Infof("kafka cluster %s is rejected and current status is %s", kafka.ID, kafka.Status)
	}

	return nil
}

func (d *dataPlaneKafkaService) checkKafkaRequestCurrentStatus(kafka *dbapi.KafkaRequest, status constants.KafkaStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.kafkaService.GetById(kafka.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

// stores routes reported by data plane to the database if not already persisted
func (d *dataPlaneKafkaService) persistKafkaRoutes(kafka *dbapi.KafkaRequest, kafkaStatus *dbapi.DataPlaneKafkaStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if kafka.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Kafka %s as they are already stored", kafka.ID)
		return nil
	}

	if len(kafkaStatus.Routes) < 1 {
		logger.Logger.V(10).Infof("skip persisting routes for Kafka %s as they are not available", kafka.ID)
		return nil
	}

	logger.Logger.Infof("store routes information for kafka %s", kafka.ID)
	clusterDNS, err := d.clusterService.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to get DNS entry for cluster %s", cluster.ClusterID)
	}

	routesInRequest := kafkaStatus.Routes
	var routes []dbapi.DataPlaneKafkaRoute

	var routesErr error
	baseClusterDomain := strings.TrimPrefix(clusterDNS, fmt.Sprintf("%s.", constants.DefaultIngressDnsNamePrefix))
	if routes, routesErr = d.buildKafkaRoutes(routesInRequest, kafka, baseClusterDomain); routesErr != nil {
		return serviceError.NewWithCause(serviceError.ErrorBadRequest, routesErr, "routes are not valid")
	}

	if err := kafka.SetRoutes(routes); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set routes for kafka %s", kafka.ID)
	}

	if err := d.kafkaService.Update(kafka); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for kafka cluster %s", kafka.ID)
	}

	return nil
}

func (d *dataPlaneKafkaService) getManagedKafkaStatus(status *dbapi.DataPlaneKafkaStatus) managedKafkaStatus {
	for _, c := range status.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			// Kafka can be upgraded when suspended. During upgrade, the Kafka instance will be resumed
			// and status will be changed with appropriate values for 'Status'. However, 'Reason' will still be set to 'Suspended'
			// The check for 'Suspended' state must always be first to accommodate this.
			if strings.EqualFold(c.Reason, "Suspended") {
				return statusSuspended
			}
			if strings.EqualFold(c.Status, "True") {
				return statusReady
			}
			if strings.EqualFold(c.Status, "Unknown") {
				return statusUnknown
			}
			if strings.EqualFold(c.Reason, "Installing") {
				return statusInstalling
			}
			if strings.EqualFold(c.Reason, "Deleted") {
				return statusDeleted
			}
			if strings.EqualFold(c.Reason, "Error") {
				return statusError
			}
			if strings.EqualFold(c.Reason, "Rejected") {
				if strings.EqualFold(c.Message, "Cluster has insufficient resources") {
					return statusRejectedClusterFull
				}
				return statusRejected
			}
		}
	}
	return statusInstalling
}

func (d *dataPlaneKafkaService) buildKafkaRoutes(routesInRequest []dbapi.DataPlaneKafkaRouteRequest, kafka *dbapi.KafkaRequest, clusterDNS string) ([]dbapi.DataPlaneKafkaRoute, error) {
	routes := []dbapi.DataPlaneKafkaRoute{}
	bootstrapServer := kafka.BootstrapServerHost
	for _, r := range routesInRequest {
		if strings.HasSuffix(r.Router, clusterDNS) {
			router := dbapi.DataPlaneKafkaRoute{
				Router: r.Router,
			}
			if r.Prefix != "" {
				router.Domain = fmt.Sprintf("%s-%s", r.Prefix, bootstrapServer)
			} else {
				router.Domain = bootstrapServer
			}
			routes = append(routes, router)
		} else {
			return nil, errors.Errorf("router domain is not valid. router = %s, expected domain = %s", r.Router, clusterDNS)
		}
	}
	return routes, nil
}

// getManagedKafkaDeploymentType gets the deployment type from the data plane kafka status
// If the id of the status begins with "reserved-" then the deployment type is reserved.
// Otherwise, it is a "real" deployment
func (d *dataPlaneKafkaService) getManagedKafkaDeploymentType(ks *dbapi.DataPlaneKafkaStatus) managedKafkaDeploymentType {
	if strings.HasPrefix(ks.KafkaClusterId, fmt.Sprintf("%s-", reservedDeploymentType.String())) {
		return reservedDeploymentType
	}

	return realDeploymentType
}
