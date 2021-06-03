package workers

import (
	"context"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
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
	metrics.SetLeaderWorkerMetric(k.workerType, true)
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ConnectorManager) Stop() {
	k.reconciler.Stop(k)
	metrics.SetLeaderWorkerMetric(k.workerType, false)
}

func (c *ConnectorManager) IsRunning() bool {
	return c.isRunning
}

func (c *ConnectorManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *ConnectorManager) Reconcile() []error {
	glog.V(5).Infoln("reconciling connectors")
	var errs []error

	serviceErr := k.connectorService.ForEach(func(connector *api.Connector) *serviceError.ServiceError {
		return InDBTransaction(func(ctx context.Context) error {
			if err := db.AddPostCommitAction(ctx, func() {
				k.lastVersion = connector.Version
			}); err != nil {
				return err
			}
			switch connector.Status.Phase {
			case api.ConnectorStatusPhaseAssigning:
				// if it's not been assigned yet, we can immediately delete it.
				if connector.DesiredState == "deleted" {
					if err := k.reconcileDeleted(ctx, connector); err != nil {
						errs = append(errs, err)
						glog.Errorf("failed to reconcile assigning connector %s: %v", connector.ID, err)
					}
				} else {
					if err := k.reconcileAssigning(ctx, connector); err != nil {
						errs = append(errs, err)
						glog.Errorf("failed to reconcile assigning connector %s: %v", connector.ID, err)
					}

				}
			}
			return nil
		})
	}, "phase IN (?)", []api.ConnectorStatusPhase{
		api.ConnectorStatusPhaseAssigning,
	})
	if serviceErr != nil {
		errs = append(errs, serviceErr)
	}

	// Process any connector updates...
	serviceErr = k.connectorService.ForEach(func(connector *api.Connector) *serviceError.ServiceError {
		return InDBTransaction(func(ctx context.Context) error {
			switch connector.Status.Phase {
			case api.ConnectorStatusPhaseAssigning:
			case api.ConnectorClusterPhaseDeleted:
				if err := k.reconcileDeleted(ctx, connector); err != nil {
					errs = append(errs, err)
					glog.Errorf("failed to reconcile assigning connector %s: %v", connector.ID, err)
				}
			default:
				if err := k.reconcileAssigned(ctx, connector); err != nil {
					return serviceError.GeneralError("failed to reconcile assigned connector %s: %v", connector.ID, err.Error())
				}
			}
			if err := db.AddPostCommitAction(ctx, func() {
				k.lastVersion = connector.Version
			}); err != nil {
				return err
			}
			return nil
		})
	}, "version > ?", k.lastVersion)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "connector manager"))
	}

	return errs
}

func InDBTransaction(f func(ctx context.Context) error) (rerr *serviceError.ServiceError) {
	ctx, err := db.NewContext(context.Background())
	if err != nil {
		return serviceError.GeneralError("failed to create tx: %v", err)
	}
	err = db.Begin(ctx)
	if err != nil {
		return serviceError.GeneralError("failed to create tx: %v", err)
	}
	defer func() {
		err = db.Resolve(ctx)
		if err != nil {
			rerr = serviceError.GeneralError("failed to resolve tx: %v", err)
		}
	}()
	err = f(ctx)
	if err != nil {
		db.MarkForRollback(ctx, rerr)
		if err, ok := err.(*serviceError.ServiceError); ok {
			return err
		}
		return serviceError.GeneralError("%v", err)
	}
	return nil
}

func (k *ConnectorManager) reconcileAssigning(ctx context.Context, connector *api.Connector) error {
	switch connector.TargetKind {
	case api.AddonTargetKind:

		cluster, err := k.connectorClusterService.FindReadyCluster(connector.Owner, connector.OrganisationId, connector.AddonClusterId)
		if err != nil {
			return errors.Wrapf(err, "failed to find cluster for connector request %s", connector.ID)
		}
		if cluster == nil {
			// we will try to find a ready cluster again in the next reconcile
			return nil
		} else {

			var status = api.ConnectorStatus{}
			status.ID = connector.ID
			status.ClusterID = cluster.ID
			status.Phase = api.ConnectorStatusPhaseAssigned
			if err = k.connectorService.SaveStatus(ctx, status); err != nil {
				return errors.Wrapf(err, "failed to update connector status %s with cluster details", connector.ID)
			}

			deployment := api.ConnectorDeployment{
				Meta: api.Meta{
					ID: api.NewID(),
				},
				ConnectorID:      connector.ID,
				ClusterID:        cluster.ID,
				ConnectorVersion: connector.Version,
				Status:           api.ConnectorDeploymentStatus{},
			}

			if err = k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
				return errors.Wrapf(err, "failed to create connector deployment for connector %s", connector.ID)
			}

		}

	default:
		return errors.Errorf("target kind not supported: %s", connector.TargetKind)
	}
	return nil
}

func (k *ConnectorManager) reconcileAssigned(ctx context.Context, connector *api.Connector) error {

	// Get the deployment for the connector...
	deployment, serr := k.connectorClusterService.GetDeploymentByConnectorId(ctx, connector.ID)
	if serr != nil {
		return serr
	}

	if deployment.ConnectorVersion != connector.Version {
		deployment.ConnectorVersion = connector.Version
		if err := k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
			return errors.Wrapf(err, "failed to create connector deployment for connector %s", connector.ID)
		}
	}
	return nil
}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *api.Connector) error {

	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return errors.Wrapf(err, "failed to delete connector %s", connector.ID)
	}
	return nil
}
