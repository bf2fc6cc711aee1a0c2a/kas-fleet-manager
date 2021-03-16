package services

import (
	"context"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

type kafkaStatus string

const (
	statusInstalling kafkaStatus = "installing"
	statusReady      kafkaStatus = "ready"
	statusError      kafkaStatus = "error"
	statusRejected   kafkaStatus = "rejected"
	statusDeleted    kafkaStatus = "deleted"
)

type DataPlaneKafkaService interface {
	UpdateDataPlaneKafkaService(ctx context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *errors.ServiceError
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

func (d *dataPlaneKafkaService) UpdateDataPlaneKafkaService(_ context.Context, clusterId string, status []*api.DataPlaneKafkaStatus) *errors.ServiceError {
	cluster, err := d.clusterService.FindClusterByID(clusterId)
	if err != nil {
		return err
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster id %s not found", clusterId)
	}
	for _, ks := range status {
		kafka, getErr := d.kafkaService.GetById(ks.KafkaClusterId)
		if getErr != nil {
			glog.Errorf("failed to get kafka cluster by id %s due to error: %v", ks.KafkaClusterId, getErr)
			continue
		}
		if kafka.ClusterID != clusterId {
			glog.Warningf("clusterId for kafka cluster %s does not match clusterId. kafka clusterId = %s :: clusterId = %s", kafka.ID, kafka.ClusterID, clusterId)
			continue
		}
		var e *errors.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			e = d.setKafkaClusterReady(kafka)
		case statusError:
			e = d.setKafkaClusterFailed(kafka)
		case statusDeleted:
			e = d.setKafkaClusterDeleted(kafka)
		case statusRejected:
			e = d.reassignKafkaCluster(kafka)
		default:
			glog.V(5).Infof("kafka cluster %s is still installing", ks.KafkaClusterId)
		}
		if e != nil {
			sentry.CaptureException(e)
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterReady(kafka *api.KafkaRequest) *errors.ServiceError {
	if ok, err := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusReady); ok {
		if err != nil {
			glog.Errorf("failed to update status %s for kafka cluster %s due to error: %v", constants.KafkaRequestStatusReady, kafka.ID, err)
			return err
		}
		metrics.UpdateKafkaCreationDurationMetric(metrics.JobTypeKafkaCreate, time.Since(kafka.CreatedAt))
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationCreate)
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterFailed(kafka *api.KafkaRequest) *errors.ServiceError {
	if ok, err := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusFailed); ok {
		if err != nil {
			glog.Errorf("failed to update status %s for kafka cluster %s due to error: %v", constants.KafkaRequestStatusFailed, kafka.ID, err)
			return err
		}
		metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationCreate)
	}
	return nil
}

func (d *dataPlaneKafkaService) setKafkaClusterDeleted(kafka *api.KafkaRequest) *errors.ServiceError {
	// If the Kafka cluster is deleted from the data plane cluster, we will make it as "deleted" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.kafkaService.UpdateStatus(kafka.ID, constants.KafkaRequestStatusDeleted); ok {
		if updateErr != nil {
			glog.Errorf("failed to update status %s for kafka cluster %s due to error: %v", constants.KafkaRequestStatusDeleted, kafka.ID, updateErr)
			return updateErr
		}
	}
	return nil
}

func (d *dataPlaneKafkaService) reassignKafkaCluster(kafka *api.KafkaRequest) *errors.ServiceError {
	if kafka.Status == constants.KafkaRequestStatusProvisioning.String() {
		// If a Kafka cluster is rejected by the kas-fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the kas-fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for kafka clusters if there are multiple OSD clusters and the value here can be the key of that table
		kafka.PlacementId = api.NewID()
		if err := d.kafkaService.Update(kafka); err != nil {
			return err
		}
	} else {
		glog.Infof("kafka cluster %s is rejected and current status is %s", kafka.ID, kafka.Status)
	}
	return nil
}

func getStatus(status *api.DataPlaneKafkaStatus) kafkaStatus {
	for _, c := range status.Conditions {
		if strings.EqualFold(c.Type, "Ready") && strings.EqualFold(c.Status, "True") {
			return statusReady
		}
		if strings.EqualFold(c.Type, "NotReady") && strings.EqualFold(c.Status, "True") && !strings.EqualFold(c.Reason, "Creating") {
			return statusError
		}
		if strings.EqualFold(c.Type, "Deleted") && strings.EqualFold(c.Status, "True") {
			return statusDeleted
		}
		if strings.EqualFold(c.Type, "Rejected") && strings.EqualFold(c.Status, "True") {
			return statusRejected
		}
	}
	return statusInstalling
}
