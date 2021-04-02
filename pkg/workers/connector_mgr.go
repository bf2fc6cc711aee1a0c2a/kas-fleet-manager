package workers

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"sync"
)

// ConnectorManager represents a connector manager that periodically reconciles connector requests
type ConnectorManager struct {
	id                      string
	workerType              string
	isRunning               bool
	imStop                  chan struct{}
	syncTeardown            sync.WaitGroup
	reconciler              Reconciler
	connectorService        services.ConnectorsService
	connectorClusterService services.ConnectorClusterService
	observatoriumService    services.ObservatoriumService
	connectorTypesService   services.ConnectorTypesService
	vaultService            services.VaultService
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(id string, connectorTypesService services.ConnectorTypesService, connectorService services.ConnectorsService, connectorClusterService services.ConnectorClusterService, observatoriumService services.ObservatoriumService, vaultService services.VaultService) *ConnectorManager {
	return &ConnectorManager{
		id:                      id,
		workerType:              "connector",
		connectorService:        connectorService,
		connectorClusterService: connectorClusterService,
		observatoriumService:    observatoriumService,
		connectorTypesService:   connectorTypesService,
		vaultService:            vaultService,
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
		api.ConnectorStatusDeleted,
	}
	serviceErr := k.connectorService.ForEachInStatus(statuses, func(connector *api.Connector) *errors.ServiceError {

		ctx, err := db.NewContext(context.Background())
		if err != nil {
			return errors.GeneralError("failed to create tx: %v", err)
		}
		err = db.Begin(ctx)
		if err != nil {
			return errors.GeneralError("failed to create tx: %v", err)
		}
		defer func() {
			err := db.Resolve(ctx)
			if err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to resolve tx: %s", err.Error())
			}
		}()

		switch connector.Status {
		case api.ConnectorStatusAssigning:
			if err := k.reconcileAccepted(ctx, connector); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to reconcile accepted connector %s: %s", connector.ID, err.Error())
			}
		case api.ConnectorClusterPhaseDeleted:
			if err := k.reconcileDeleted(ctx, connector); err != nil {
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

func (k *ConnectorManager) reconcileAccepted(ctx context.Context, connector *api.Connector) error {
	switch connector.TargetKind {
	case api.AddonTargetKind:

		cluster, err := k.connectorClusterService.FindReadyCluster(connector.Owner, connector.OrganisationId, connector.AddonClusterId)
		if err != nil {
			return fmt.Errorf("failed to find cluster for connector request %s: %w", connector.ID, err)
		}
		if cluster != nil {

			address, err := k.connectorTypesService.GetServiceAddress(connector.ConnectorTypeId)
			if err != nil {
				return fmt.Errorf("failed to get service address for connector type %s: %w", connector.ConnectorTypeId, err)
			}

			connector.ClusterID = cluster.ID
			connector.Status = api.ConnectorStatusAssigned
			if err = k.connectorService.Update(ctx, connector); err != nil {
				return fmt.Errorf("failed to update connector %s with cluster details: %w", connector.ID, err)
			}

			deployment := api.ConnectorDeployment{
				Meta: api.Meta{
					ID: api.NewID(),
				},
				ConnectorID:          connector.ID,
				ConnectorVersion:     connector.Version,
				ClusterID:            cluster.ID,
				ConnectorTypeService: address,
				SpecChecksum:         "",
				Status:               api.ConnectorDeploymentStatus{},
			}

			spec, err := k.connectorClusterService.GetConnectorClusterSpec(ctx, deployment)
			if err != nil {
				return err
			}

			checksum, eerr := services.Checksum(spec)
			if eerr != nil {
				return fmt.Errorf("could not checksum deployment sepc for connector %s: %w", connector.ID, err)
			}
			deployment.SpecChecksum = checksum

			if err = k.connectorClusterService.CreateDeployment(ctx, &deployment); err != nil {
				return fmt.Errorf("failed to create connector deployment for connector %s: %w", connector.ID, err)
			}

		}

	default:
		return fmt.Errorf("target kind not supported: %s", connector.TargetKind)
	}
	return nil
}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *api.Connector) error {
	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return fmt.Errorf("failed to delete connector %s: %w", connector.ID, err)
	}
	return nil
}
