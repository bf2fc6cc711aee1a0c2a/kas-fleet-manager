package workers

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ConnectorManager represents a connector manager that periodically reconciles connector requests
type ConnectorManager struct {
	workers.BaseWorker
	connectorService        services.ConnectorsService
	connectorClusterService services.ConnectorClusterService
	connectorTypesService   services.ConnectorTypesService
	vaultService            vault.VaultService
	lastVersion             int64
	startupReconcileDone    bool
	startupReconcileWG      sync.WaitGroup
	db                      *db.ConnectionFactory
	ctx                     context.Context
}

// NewApiServerReadyCondition is used to inject a server.ApiServerReadyCondition into the server.ApiServer
// so that it waits for the ConnectorManager to have completed a startup reconcile before accepting http requests.
func NewApiServerReadyCondition(cm *ConnectorManager) server.ApiServerReadyCondition {
	return &cm.startupReconcileWG
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(
	connectorTypesService services.ConnectorTypesService,
	connectorService services.ConnectorsService,
	connectorClusterService services.ConnectorClusterService,
	vaultService vault.VaultService,
	db *db.ConnectionFactory,
	reconciler workers.Reconciler,
) *ConnectorManager {
	result := &ConnectorManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector",
			Reconciler: reconciler,
		},
		connectorService:        connectorService,
		connectorClusterService: connectorClusterService,
		connectorTypesService:   connectorTypesService,
		vaultService:            vaultService,
		startupReconcileDone:    false,
		db:                      db,
	}
	result.startupReconcileWG.Add(1)
	return result
}

// Start initializes the connector manager to reconcile connector requests
func (k *ConnectorManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ConnectorManager) Stop() {
	k.StopWorker(k)
}

func (k *ConnectorManager) Reconcile() []error {
	glog.V(5).Infoln("reconciling connectors")
	var errs []error

	if !k.startupReconcileDone {

		// We only need to reconcile channel updates once per process startup since,
		// configured channel settings are only loaded on startup.
		if err := k.connectorTypesService.ForEachConnectorCatalogEntry(k.ReconcileConnectorCatalogEntry); err != nil {
			return []error{err}
		}

		if err := k.connectorClusterService.CleanupDeployments(); err != nil {
			return []error{err}
		}

		k.startupReconcileDone = true
		k.startupReconcileWG.Done()
	}

	if k.ctx == nil {
		ctx, err := k.db.NewContext(context.Background())
		if err != nil {
			return []error{err}
		}
		k.ctx = ctx
	}

	// reconcile assigning connectors in "ready" desired state with "assigning" phase and a valid namespace id
	if serviceErr := k.connectorService.ForEach(func(connector *dbapi.Connector) *serviceError.ServiceError {
		return InDBTransaction(k.ctx, func(ctx context.Context) error {
			if err := k.reconcileAssigning(ctx, connector); err != nil {
				errs = append(errs, err)
				glog.Errorf("failed to reconcile assigning connector %s: %v", connector.ID, err)
			}
			return nil
		})
	}, "desired_state = ? AND phase = ? AND connectors.namespace_id IS NOT NULL",
		dbapi.ConnectorReady, dbapi.ConnectorStatusPhaseAssigning); serviceErr != nil {
		errs = append(errs, serviceErr)
	}

	// reconcile unassigned connectors in "unassigned" desired state and "deleted" phase
	if serviceErr := k.connectorService.ForEach(func(connector *dbapi.Connector) *serviceError.ServiceError {
		return InDBTransaction(k.ctx, func(ctx context.Context) error {
			// we can remove the connector from the namespace and set phase to "assigning" now...
			if err := k.reconcileUnassigned(ctx, connector); err != nil {
				errs = append(errs, err)
				glog.Errorf("failed to reconcile unassigned connector %s: %v", connector.ID, err)
			}
			return nil
		})
	}, "desired_state = ? AND phase = ?", dbapi.ConnectorUnassigned, dbapi.ConnectorStatusPhaseDeleted); serviceErr != nil {
		errs = append(errs, serviceErr)
	}

	// reconcile deleted connectors with no deployments
	if serviceErr := k.connectorService.ForEach(func(connector *dbapi.Connector) *serviceError.ServiceError {
		return InDBTransaction(k.ctx, func(ctx context.Context) error {
			// ok to delete the connector without deployment
			if err := k.reconcileDeleted(ctx, connector); err != nil {
				errs = append(errs, err)
				glog.Errorf("failed to reconcile deleted connector %s: %v", connector.ID, err)
			}
			return nil
		})
	}, "desired_state = ? AND phase IN ?", dbapi.ConnectorDeleted,
		[]string{string(dbapi.ConnectorStatusPhaseAssigning), string(dbapi.ConnectorStatusPhaseDeleted)}); serviceErr != nil {
		errs = append(errs, serviceErr)
	}

	// Process any connector updates...
	if serviceErr := k.connectorService.ForEach(func(connector *dbapi.Connector) *serviceError.ServiceError {
		return InDBTransaction(k.ctx, func(ctx context.Context) error {
			if err := k.reconcileConnectorUpdate(ctx, connector); err != nil {
				return serviceError.GeneralError("failed to reconcile assigned connector %s: %v", connector.ID, err.Error())
			}
			if err := db.AddPostCommitAction(ctx, func() {
				k.lastVersion = connector.Version
			}); err != nil {
				return err
			}
			return nil
		})
	}, "version > ? AND desired_state != ? AND phase != ?", k.lastVersion, dbapi.ConnectorUnassigned, dbapi.ConnectorStatusPhaseAssigning); serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "connector manager"))
	}

	return errs
}

func (k *ConnectorManager) ReconcileConnectorCatalogEntry(id string, channel string, ccc *config.ConnectorChannelConfig) *serviceError.ServiceError {

	ctc := dbapi.ConnectorShardMetadata{
		ConnectorTypeId: id,
		Channel:         channel,
	}
	var err error
	ctc.ShardMetadata, err = json.Marshal(ccc.ShardMetadata)
	if err != nil {
		return serviceError.GeneralError("failed to convert connector type %s, channel %s: %v", id, channel, err.Error())
	}

	// We store connector type channels so we can track changes and trigger redeployment of
	// associated connectors upon connector type channel changes.
	_, serr := k.connectorTypesService.PutConnectorShardMetadata(&ctc)
	if serr != nil {
		return serr
	}

	return nil
}

//goland:noinspection VacuumSwitchStatement
func (k *ConnectorManager) reconcileAssigning(ctx context.Context, connector *dbapi.Connector) error {
	var namespace *dbapi.ConnectorNamespace
	namespace, err := k.connectorClusterService.FindAvailableNamespace(connector.Owner, connector.OrganisationId, connector.NamespaceId)
	if err != nil {
		return errors.Wrapf(err, "failed to find namespace for connector request %s", connector.ID)
	}
	if namespace == nil {
		// we will try to find a ready namespace again in the next reconcile
		return nil
	}

	channelVersion, err := k.connectorTypesService.GetLatestConnectorShardMetadataID(connector.ConnectorTypeId, connector.Channel)
	if err != nil {
		return errors.Wrapf(err, "failed to get latest channel version for connector request %s", connector.ID)
	}

	var status = dbapi.ConnectorStatus{}
	status.ID = connector.ID
	status.NamespaceID = &namespace.ID
	status.Phase = dbapi.ConnectorStatusPhaseAssigned
	if err = k.connectorService.SaveStatus(ctx, status); err != nil {
		return errors.Wrapf(err, "failed to update connector status %s with namespace details", status.ID)
	}

	deployment := dbapi.ConnectorDeployment{
		Model: db.Model{
			ID: api.NewID(),
		},
		ConnectorID:            connector.ID,
		ClusterID:              namespace.ClusterId,
		NamespaceID:            namespace.ID,
		ConnectorVersion:       connector.Version,
		ConnectorTypeChannelId: channelVersion,
		Status:                 dbapi.ConnectorDeploymentStatus{},
	}

	if err = k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
		return errors.Wrapf(err, "failed to create connector deployment for connector %s", connector.ID)
	}

	return nil
}

func (k *ConnectorManager) reconcileUnassigned(ctx context.Context, connector *dbapi.Connector) error {
	// set phase to "assigning" and namespace_id to nil
	connector.Status.Phase = dbapi.ConnectorStatusPhaseAssigning
	connector.Status.NamespaceID = nil
	connector.NamespaceId = nil

	if err := k.db.New().Model(&connector).Where("id = ?", connector.ID).
		Update("namespace_id", nil).Error; err != nil {
		return errors.Wrapf(err, "failed to update namespace_id for connector %s", connector.ID)
	}
	if err := k.connectorService.SaveStatus(ctx, connector.Status); err != nil {
		return errors.Wrapf(err, "failed to update phase to assigning for connector %s", connector.ID)
	}

	return nil
}

func (k *ConnectorManager) reconcileConnectorUpdate(ctx context.Context, connector *dbapi.Connector) error {

	// Get the deployment for the connector...
	deployment, serr := k.connectorClusterService.GetDeploymentByConnectorId(ctx, connector.ID)
	if serr != nil {
		return serr
	}

	// we may need to update the deployment due to connector change.
	if deployment.ConnectorVersion != connector.Version {
		deployment.ConnectorVersion = connector.Version
		if err := k.connectorClusterService.SaveDeployment(ctx, &deployment); err != nil {
			return errors.Wrapf(err, "failed to create connector deployment for connector %s", connector.ID)
		}
	}

	return nil
}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *dbapi.Connector) error {
	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return errors.Wrapf(err, "failed to delete connector %s", connector.ID)
	}
	return nil
}

func InDBTransaction(ctx context.Context, f func(ctx context.Context) error) (rerr *serviceError.ServiceError) {
	err := db.Begin(ctx)
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
