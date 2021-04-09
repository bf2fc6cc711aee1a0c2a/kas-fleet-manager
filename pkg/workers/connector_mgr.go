package workers

import (
	"context"
	"fmt"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
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
	lastVersion             int64
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

	serviceErr := k.connectorService.ForEach(func(connector *api.Connector) *errors.ServiceError {
		return InDBTransaction(func(ctx context.Context) error {
			switch connector.Status.Phase {
			case api.ConnectorStatusPhaseAssigning:
				if err := k.reconcileAssigning(ctx, connector); err != nil {
					return errors.GeneralError("failed to reconcile assigning connector %s: %v", connector.ID, err)
				}
			case api.ConnectorClusterPhaseDeleted:
				if err := k.reconcileDeleted(ctx, connector); err != nil {
					return errors.GeneralError("failed to reconcile deleted connector %s: %v", connector.ID, err.Error())
				}
			}
			return nil
		})
	}, "phase IN (?)", []api.ConnectorStatusPhase{
		api.ConnectorStatusPhaseAssigning,
		api.ConnectorStatusPhaseDeleted,
	})
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("connector manager: %s", serviceErr.Error())
	}

	// Process any connector updates...
	serviceErr = k.connectorService.ForEach(func(connector *api.Connector) *errors.ServiceError {
		return InDBTransaction(func(ctx context.Context) error {
			if err := db.AddPostCommitAction(ctx, func() {
				k.lastVersion = connector.Version
			}); err != nil {
				return err
			}
			if err := k.reconcileAssigned(ctx, connector); err != nil {
				return errors.GeneralError("failed to reconcile assigned connector %s: %v", connector.ID, err.Error())
			}
			return nil
		})
	}, "version > ? AND phase IN (?) ", k.lastVersion, []api.ConnectorStatusPhase{
		api.ConnectorStatusPhaseProvisioning,
		api.ConnectorStatusPhaseAssigned,
		api.ConnectorStatusPhaseFailed,
		api.ConnectorStatusPhaseReady,
	})

	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("connector manager: %s", serviceErr.Error())
	}
}

func InDBTransaction(f func(ctx context.Context) error) (rerr *errors.ServiceError) {
	ctx, err := db.NewContext(context.Background())
	if err != nil {
		return errors.GeneralError("failed to create tx: %v", err)
	}
	err = db.Begin(ctx)
	if err != nil {
		return errors.GeneralError("failed to create tx: %v", err)
	}
	defer func() {
		err = db.Resolve(ctx)
		if err != nil {
			rerr = errors.GeneralError("failed to resolve tx: %v", err)
		}
	}()
	err = f(ctx)
	if err != nil {
		db.MarkForRollback(ctx, rerr)
		if err, ok := err.(*errors.ServiceError); ok {
			return err
		}
		return errors.GeneralError("%v", err)
	}
	return nil
}

func (k *ConnectorManager) reconcileAssigning(ctx context.Context, connector *api.Connector) error {
	switch connector.TargetKind {
	case api.AddonTargetKind:

		cluster, err := k.connectorClusterService.FindReadyCluster(connector.Owner, connector.OrganisationId, connector.AddonClusterId)
		if err != nil {
			return fmt.Errorf("failed to find cluster for connector request %s: %w", connector.ID, err)
		}
		if cluster == nil {
			// we will try to find a ready cluster again in the next reconcile
			return nil
		} else {

			address, err := k.connectorTypesService.GetServiceAddress(connector.ConnectorTypeId)
			if err != nil {
				return fmt.Errorf("failed to get service address for connector type %s: %w", connector.ConnectorTypeId, err)
			}

			var status = api.ConnectorStatus{}
			status.ID = connector.ID
			status.ClusterID = cluster.ID
			status.Phase = api.ConnectorStatusPhaseAssigned
			if err = k.connectorService.SaveStatus(ctx, status); err != nil {
				return fmt.Errorf("failed to update connector status %s with cluster details: %w", connector.ID, err)
			}

			deployment := api.ConnectorDeployment{
				Meta: api.Meta{
					ID: api.NewID(),
				},
				ConnectorID:          connector.ID,
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

			if err = k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
				return fmt.Errorf("failed to create connector deployment for connector %s: %w", connector.ID, err)
			}

		}

	default:
		return fmt.Errorf("target kind not supported: %s", connector.TargetKind)
	}
	return nil
}

func (k *ConnectorManager) reconcileAssigned(ctx context.Context, connector *api.Connector) error {

	// Get the deployment for the connector...
	deployment, serr := k.connectorClusterService.GetDeployment(ctx, connector.ID)
	if serr != nil {
		return serr
	}

	// reify the connector
	spec, err := k.connectorClusterService.GetConnectorClusterSpec(ctx, deployment)
	if err != nil {
		return err
	}

	checksum, eerr := services.Checksum(spec)
	if eerr != nil {
		return fmt.Errorf("could not checksum deployment sepc for connector %s: %w", connector.ID, err)
	}

	if deployment.SpecChecksum != checksum {
		deployment.SpecChecksum = checksum
		if err = k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
			return fmt.Errorf("failed to create connector deployment for connector %s: %w", connector.ID, err)
		}
	}
	return nil
}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *api.Connector) error {
	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return fmt.Errorf("failed to delete connector %s: %w", connector.ID, err)
	}
	return nil
}
