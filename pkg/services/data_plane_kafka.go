package services

import (
	"context"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
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
)

type DataPlaneKafkaService interface {
	UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *serviceError.ServiceError
}

type dataPlaneKafkaService struct {
	kafkaService   KafkaService
	clusterService ClusterService
}

func NewDataPlaneKafkaService(kafkaSrv KafkaService, clusterSrv ClusterService) *dataPlaneKafkaService {
	return &dataPlaneKafkaService{
		kafkaService:   kafkaSrv,
		clusterService: clusterSrv,
	}
}

func (d *dataPlaneKafkaService) UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *serviceError.ServiceError {
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
			log.Error(errors.Wrapf(getErr, "failed to get kafka cluster by id %s", ks.KafkaClusterId))
			continue
		}
		if kafka.ClusterID != clusterId {
			log.Warningf("clusterId for kafka cluster %s does not match clusterId. kafka clusterId = %s :: clusterId = %s", kafka.ID, kafka.ClusterID, clusterId)
			continue
		}
		var e *serviceError.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			e = d.setKafkaClusterReady(kafka)
		case statusError:
			e = d.setKafkaClusterFailed(kafka)
		case statusDeleted:
			e = d.setKafkaClusterDeleting(kafka)
		case statusRejected:
			e = d.reassignKafkaCluster(kafka)
		case statusUnknown:
			log.Warningf("kafka cluster %s status is unknown", ks.KafkaClusterId)
		default:
			log.V(5).Infof("kafka cluster %s is still installing", ks.KafkaClusterId)
		}
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating kafka %s status", ks.KafkaClusterId))
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterReady(kafka *api.KafkaRequest) *serviceError.ServiceError {
	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants.KafkaRequestStatusProvisioning)
	if err != nil {
		return err
	}

	if ok, err := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusReady); ok {
		if err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update status %s for kafka cluster %s", constants.KafkaRequestStatusReady, kafka.ID)
		}
		if shouldSendMetric {
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusReady, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
			metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
			metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterFailed(kafka *api.KafkaRequest) *serviceError.ServiceError {
	// only send metrics data if the current kafka request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkKafkaRequestCurrentStatus(kafka, constants.KafkaRequestStatusProvisioning)
	if err != nil {
		return err
	}
	if ok, err := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusFailed); ok {
		if err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update status %s for kafka cluster %s", constants.KafkaRequestStatusFailed, kafka.ID)
		}
		if shouldSendMetric {
			metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusFailed, kafka.ID, kafka.ClusterID, time.Since(kafka.CreatedAt))
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterDeleting(kafka *api.KafkaRequest) *serviceError.ServiceError {
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

func (d *dataPlaneKafkaService) reassignKafkaCluster(kafka *api.KafkaRequest) *serviceError.ServiceError {
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
		glog.Infof("kafka cluster %s is rejected and current status is %s", kafka.ID, kafka.Status)
	}

	return nil
}

func getStatus(status *api.DataPlaneKafkaStatus) kafkaStatus {
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

func (d *dataPlaneKafkaService) checkKafkaRequestCurrentStatus(kafka *api.KafkaRequest, status constants.KafkaStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.kafkaService.GetById(kafka.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}
