package dinosaur_mgrs

import (
	"fmt"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/signalbus"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"
)

// AcceptedDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type AcceptedDinosaurManager struct {
	workers.BaseWorker
	dinosaurService        services.DinosaurService
	quotaServiceFactory    services.QuotaServiceFactory
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	dataPlaneClusterConfig *config.DataplaneClusterConfig
}

// NewAcceptedDinosaurManager creates a new dinosaur manager
func NewAcceptedDinosaurManager(dinosaurService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, bus signalbus.SignalBus, dataPlaneClusterConfig *config.DataplaneClusterConfig) *AcceptedDinosaurManager {
	return &AcceptedDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "accepted_dinosaur",
			Reconciler: workers.Reconciler{
				SignalBus: bus,
			},
		},
		dinosaurService:        dinosaurService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *AcceptedDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *AcceptedDinosaurManager) Stop() {
	k.StopWorker(k)
}

func (k *AcceptedDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling accepted dinosaurs")
	var encounteredErrors []error

	// handle accepted dinosaurs
	acceptedDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted dinosaurs"))
	} else {
		glog.Infof("accepted dinosaurs count = %d", len(acceptedDinosaurs))
	}

	for _, dinosaur := range acceptedDinosaurs {
		glog.V(10).Infof("accepted dinosaur id = %s", dinosaur.ID)
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusAccepted, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		if err := k.reconcileAcceptedDinosaur(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted dinosaur %s", dinosaur.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedDinosaurManager) reconcileAcceptedDinosaur(dinosaur *dbapi.DinosaurRequest) error {
	cluster, err := k.clusterPlmtStrategy.FindCluster(dinosaur)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for dinosaur request %s", dinosaur.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Dinosaur instance with id %s", dinosaur.ID)
		return nil
	}

	dinosaur.ClusterID = cluster.ClusterID
	if k.dataPlaneClusterConfig.StrimziOperatorVersion == "" {
		readyStrimziVersions, err := cluster.GetAvailableAndReadyStrimziVersions()
		if err != nil || len(readyStrimziVersions) == 0 {
			dinosaur.Status = constants2.DinosaurRequestStatusFailed.String()
			if err != nil {
				err = errors.Wrapf(err, "failed to get desired Strimzi version %s", dinosaur.ID)
			} else {
				err = errors.New(fmt.Sprintf("failed to get desired Strimzi version %s", dinosaur.ID))
			}
			dinosaur.FailedReason = err.Error()
			if err2 := k.dinosaurService.Update(dinosaur); err2 != nil {
				return errors.Wrapf(err2, "failed to update failed dinosaur %s", dinosaur.ID)
			}
			return err
		} else {
			dinosaur.DesiredStrimziVersion = readyStrimziVersions[len(readyStrimziVersions)-1].Version
		}
	} else {
		dinosaur.DesiredStrimziVersion = k.dataPlaneClusterConfig.StrimziOperatorVersion
	}

	glog.Infof("Dinosaur instance with id %s is assigned to cluster with id %s", dinosaur.ID, dinosaur.ClusterID)
	dinosaur.Status = constants2.DinosaurRequestStatusPreparing.String()
	if err2 := k.dinosaurService.Update(dinosaur); err2 != nil {
		return errors.Wrapf(err2, "failed to update dinosaur %s with cluster details", dinosaur.ID)
	}
	return nil
}
