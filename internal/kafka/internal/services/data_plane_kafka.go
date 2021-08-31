package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type kafkaStatus string

const (
	statusInstalling kafkaStatus = "installing"
	statusReady      kafkaStatus = "ready"
	statusError      kafkaStatus = "error"
	statusRejected   kafkaStatus = "rejected"
	statusDeleted    kafkaStatus = "deleted"
	statusUnknown    kafkaStatus = "unknown"
	strimziUpdating  string      = "StrimziUpdating"
)

type DataPlaneKafkaService interface {
	UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError
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

func (d *dataPlaneKafkaService) UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError {
	cluster, err := d.clusterService.FindClusterByID(clusterId)
	log := logger.NewUHCLogger(ctx)
	if err != nil {
		return err
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return serviceError.BadRequest("Cluster id %s not found", clusterId)
	}
	for _, ks := range status {
		kafka, getErr := d.kafkaService.GetById(ks.KafkaClusterId)
		if getErr != nil {
			glog.Error(errors.Wrapf(getErr, "failed to get kafka cluster by id %s", ks.KafkaClusterId))
			continue
		}
		if kafka.ClusterID != clusterId {
			log.Warningf("clusterId for kafka cluster %s does not match clusterId. kafka clusterId = %s :: clusterId = %s", kafka.ID, kafka.ClusterID, clusterId)
			continue
		}
		var e *serviceError.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			// for now, only store the routes (and create them) when the Kafkas are ready, as by the time they are ready,
			// the routes should definitely be there if they are set by kas-fleetshard.
			// If they are not there then we can set the default ones because it means fleetshard won't set them.
			// Once the changes are made on the fleetshard side, we can move this outside the ready state
			e = d.persistKafkaRoutes(kafka, ks, cluster)
			if e == nil {
				e = d.setKafkaClusterReady(kafka)
			}
		case statusError:
			// when getStatus returns statusError we know that the ready
			// condition will be there so there's no need to check for it
			readyCondition, _ := ks.GetReadyCondition()
			e = d.setKafkaClusterFailed(kafka, readyCondition.Message)
		case statusDeleted:
			e = d.setKafkaClusterDeleting(kafka)
		case statusRejected:
			e = d.reassignKafkaCluster(kafka)
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
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterReady(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	if !kafka.RoutesCreated {
		logger.Logger.V(10).Infof("routes for kafka %s are not created", kafka.ID)
		return nil
	} else {
		logger.Logger.Infof("routes for kafka %s are created", kafka.ID)
	}
	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants2.KafkaRequestStatusProvisioning)
	if err != nil {
		return err
	}

	kafka.FailedReason = ""
	kafka.Status = string(constants2.KafkaRequestStatusReady)
	err = d.kafkaService.Update(kafka)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update status %s for kafka cluster %s", constants2.KafkaRequestStatusReady, kafka.ID)
	}
	if shouldSendMetric {
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationCreate)
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationCreate)
	}
	/*
		if ok, err := d.kafkaService.UpdateStatus(kafka.ID, constants2.KafkaRequestStatusReady); ok {
			if err != nil {
				return serviceError.NewWithCause(err.Code, err, "failed to update status %s for kafka cluster %s", constants2.KafkaRequestStatusReady, kafka.ID)
			}
			if shouldSendMetric {
				metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
				metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
				metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationCreate)
				metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationCreate)
			}
			if kafka.FailedReason != "" {
				kafka.FailedReason = ""
				err = d.kafkaService.Update(kafka)
				if err != nil {
					return serviceError.NewWithCause(err.Code, err, "failed to reset fail reason for kafka cluster %s", kafka.ID)
				}
			}
		}
	*/
	return nil
}

func (d *dataPlaneKafkaService) setKafkaRequestVersionFields(kafka *dbapi.KafkaRequest, status *dbapi.DataPlaneKafkaStatus) *serviceError.ServiceError {
	needsUpdate := false
	prevActualKafkaVersion := status.KafkaVersion
	if status.KafkaVersion != "" && status.KafkaVersion != kafka.ActualKafkaVersion {
		logger.Logger.Infof("Updating Kafka version for Kafka ID '%s' from '%s' to '%s'", kafka.ID, prevActualKafkaVersion, status.KafkaVersion)
		kafka.ActualKafkaVersion = status.KafkaVersion
		needsUpdate = true
	}

	prevActualStrimziVersion := status.StrimziVersion
	if status.StrimziVersion != "" && status.StrimziVersion != kafka.ActualStrimziVersion {
		logger.Logger.Infof("Updating Strimzi version for Kafka ID '%s' from '%s' to '%s'", kafka.ID, prevActualStrimziVersion, status.StrimziVersion)
		kafka.ActualStrimziVersion = status.StrimziVersion
		needsUpdate = true
	}

	// TODO for now we don't set the kafka_upgrading attribute at all because
	// kas fleet shard operator still does not explicitely report whether kafka
	// is being upgraded, as it is being done with strimzi operator. Wait until
	// kas fleet shard operator has a way to report it and update the logic to do
	// appropriately set it. section wait until kas fleetshard operator has a way to report whether
	// kafka.KafkaUpgrading = kafkaBeingUpgraded
	// prevKafkaUpgrading := kafka.KafkaUpgrading
	// kafkaBeingUpgraded := kafka.DesiredKafkaVersion != kafka.ActualKafkaVersion
	// if kafkaBeingUpgraded != kafka.KafkaUpgrading {
	// 	logger.Logger.Infof("Kafka version for Kafka ID '%s' upgrade state changed from %t to %t", kafka.ID, prevKafkaUpgrading, kafkaBeingUpgraded)
	// 	kafka.KafkaUpgrading = kafkaBeingUpgraded
	// 	needsUpdate = true
	// }

	prevStrimziUpgrading := kafka.StrimziUpgrading
	readyCondition, found := status.GetReadyCondition()
	if found {
		// TODO is this really correct? What happens if there is a StrimziUpdating reason
		// but the 'status' is false? What does that mean and how should we behave?
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
	}

	if needsUpdate {
		versionFields := map[string]interface{}{
			"actual_kafka_version":   kafka.ActualKafkaVersion,
			"actual_strimzi_version": kafka.ActualStrimziVersion,
			"strimzi_upgrading":      kafka.StrimziUpgrading,
		}

		if err := d.kafkaService.Updates(kafka, versionFields); err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update actual version fields for kafka cluster %s", kafka.ID)
		}
	}

	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterFailed(kafka *dbapi.KafkaRequest, errMessage string) *serviceError.ServiceError {
	// if kafka was already reported as failed we don't do anything
	if kafka.Status == string(constants2.KafkaRequestStatusFailed) {
		return nil
	}

	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants2.KafkaRequestStatusProvisioning)
	if err != nil {
		return err
	}

	kafka.Status = string(constants2.KafkaRequestStatusFailed)
	kafka.FailedReason = fmt.Sprintf("Kafka reported as failed: '%s'", errMessage)
	err = d.kafkaService.Update(kafka)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update kafka cluster to %s status for kafka cluster %s", constants2.KafkaRequestStatusFailed, kafka.ID)
	}
	if shouldSendMetric {
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationCreate)
	}
	logger.Logger.Errorf("Kafka status reported as failed by KAS Fleet Shard Operator: '%s'", errMessage)

	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterDeleting(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	// If the Kafka cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.kafkaService.UpdateStatus(kafka.ID, constants2.KafkaRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for kafka cluster %s", constants2.KafkaRequestStatusDeleting, kafka.ID)
		} else {
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusDeleting, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) reassignKafkaCluster(kafka *dbapi.KafkaRequest) *serviceError.ServiceError {
	if kafka.Status == constants2.KafkaRequestStatusProvisioning.String() {
		// If a Kafka cluster is rejected by the kas-fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the kas-fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for kafka clusters if there are multiple OSD clusters and the value here can be the key of that table
		kafka.PlacementId = api.NewID()
		if err := d.kafkaService.Update(kafka); err != nil {
			return err
		}
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusProvisioning, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
	} else {
		logger.Logger.Infof("kafka cluster %s is rejected and current status is %s", kafka.ID, kafka.Status)
	}

	return nil
}

func getStatus(status *dbapi.DataPlaneKafkaStatus) kafkaStatus {
	for _, c := range status.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
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
				return statusRejected
			}
		}
	}
	return statusInstalling
}
func (d *dataPlaneKafkaService) checkKafkaRequestCurrentStatus(kafka *dbapi.KafkaRequest, status constants2.KafkaStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.kafkaService.GetById(kafka.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

func (d *dataPlaneKafkaService) persistKafkaRoutes(kafka *dbapi.KafkaRequest, kafkaStatus *dbapi.DataPlaneKafkaStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if kafka.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Kafka %s as they are already stored", kafka.ID)
		return nil
	}
	logger.Logger.Infof("store routes information for kafka %s", kafka.ID)
	clusterDNS, err := d.clusterService.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to get DNS entry for cluster %s", cluster.ClusterID)
	}

	routesInRequest := kafkaStatus.Routes
	var routes []dbapi.DataPlaneKafkaRoute
	if len(routesInRequest) == 0 {
		// TODO: This is here for keep backward compatibility. Remove this once the kas-fleetshard added implementation for routes. We no longer need to produce default routes at all.
		oldMKIngressDNS := strings.Replace(clusterDNS, constants2.DefaultIngressDnsNamePrefix, constants2.ManagedKafkaIngressDnsNamePrefix, 1)
		routes = kafka.GetDefaultRoutes(oldMKIngressDNS, d.kafkaConfig.NumOfBrokers)
	} else {
		var routesErr error
		baseClusterDomain := strings.TrimPrefix(clusterDNS, fmt.Sprintf("%s.", constants2.DefaultIngressDnsNamePrefix))
		if routes, routesErr = buildRoutes(routesInRequest, kafka, baseClusterDomain); routesErr != nil {
			return serviceError.NewWithCause(serviceError.ErrorBadRequest, routesErr, "routes are not valid")
		}
	}

	if err := kafka.SetRoutes(routes); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set routes for kafka %s", kafka.ID)
	}

	if err := d.kafkaService.Update(kafka); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for kafka cluster %s", kafka.ID)
	}
	return nil
}

func buildRoutes(routesInRequest []dbapi.DataPlaneKafkaRouteRequest, kafka *dbapi.KafkaRequest, clusterDNS string) ([]dbapi.DataPlaneKafkaRoute, error) {
	var routes []dbapi.DataPlaneKafkaRoute
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
