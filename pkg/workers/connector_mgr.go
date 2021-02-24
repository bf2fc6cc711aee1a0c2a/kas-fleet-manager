package workers

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"sync"
)

// ConnectorManager represents a connector manager that periodically reconciles connector requests
type ConnectorManager struct {
	ctx                     context.Context
	id                      string
	workerType              string
	isRunning               bool
	imStop                  chan struct{}
	syncTeardown            sync.WaitGroup
	reconciler              Reconciler
	connectorService        services.ConnectorsService
	connectorClusterService services.ConnectorClusterService
	observatoriumService    services.ObservatoriumService
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(id string, connectorService services.ConnectorsService, connectorClusterService services.ConnectorClusterService, observatoriumService services.ObservatoriumService) *ConnectorManager {
	return &ConnectorManager{
		ctx:                     context.Background(),
		id:                      id,
		workerType:              "connector",
		connectorService:        connectorService,
		connectorClusterService: connectorClusterService,
		observatoriumService:    observatoriumService,
	}
}

func (k *ConnectorManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *ConnectorManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *ConnectorManager) GetID() string {
	return k.id
}

func (c *ConnectorManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the connector manager to reconcile connector requests
func (k *ConnectorManager) Start() {
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ConnectorManager) Stop() {
	k.reconciler.Stop(k)
}

func (c *ConnectorManager) IsRunning() bool {
	return c.isRunning
}

func (c *ConnectorManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *ConnectorManager) reconcile() {
	glog.V(5).Infoln("reconciling connectors")

	statuses := []api.ConnectorStatus{
		api.ConnectorStatusAssigning,
		api.ConnectorClusterStatusDeleted,
	}
	serviceErr := k.connectorService.ForEachInStatus(statuses, func(connector *api.Connector) *errors.ServiceError {
		switch connector.Status {
		case api.ConnectorStatusAssigning:
			if err := k.reconcileAccepted(connector); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to reconcile accepted connector %s: %s", connector.ID, err.Error())
			}
		case api.ConnectorClusterStatusDeleted:
			if err := k.reconcileDeleted(connector); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to reconcile accepted connector %s: %s", connector.ID, err.Error())
			}
		}
		return nil
	})
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list connectors: %s", serviceErr.Error())
	}
}

func (k *ConnectorManager) reconcileAccepted(connector *api.Connector) error {
	switch connector.TargetKind {
	case api.AddonTargetKind:

		cluster, err := k.connectorClusterService.FindReadyCluster(connector.Owner, connector.AddonGroup)
		if err != nil {
			return fmt.Errorf("failed to find cluster for connector request %s: %w", connector.ID, err)
		}
		if cluster != nil {
			connector.ClusterID = cluster.ID
			connector.Status = api.ConnectorStatusAssigned
			if err = k.connectorService.Update(k.ctx, connector); err != nil {
				return fmt.Errorf("failed to update connector %s with cluster details: %w", connector.ID, err)
			}
		}

	default:
		return fmt.Errorf("target kind not supported: %s", connector.TargetKind)
	}
	return nil
}

func (k *ConnectorManager) reconcileDeleted(connector *api.Connector) error {
	if err := k.connectorService.Delete(k.ctx, connector.KafkaID, connector.ID); err != nil {
		return fmt.Errorf("failed to delete connector %s: %w", connector.ID, err)
	}
	return nil
}
