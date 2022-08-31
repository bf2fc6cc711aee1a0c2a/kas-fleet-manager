package workers

import (
	"context"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"reflect"

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
	db                      *db.ConnectionFactory
	ctx                     context.Context
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
		db:                      db,
	}

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
	glog.V(5).Infoln("Reconciling connectors...")
	var errs []error

	if k.ctx == nil {
		ctx, err := k.db.NewContext(context.Background())
		if err != nil {
			return []error{err}
		}
		k.ctx = ctx
	}

	// reconcile assigning connectors in "ready" desired state with "assigning" phase and a valid namespace id
	k.doReconcile(&errs, "assigning", k.reconcileAssigning,
		"desired_state = ? AND phase = ? AND connectors.namespace_id IS NOT NULL", dbapi.ConnectorReady, dbapi.ConnectorStatusPhaseAssigning)

	// reconcile unassigned connectors in "unassigned" desired state and "deleted" phase
	k.doReconcile(&errs, "unassigned", k.reconcileUnassigned,
		"desired_state = ? AND phase = ?", dbapi.ConnectorUnassigned, dbapi.ConnectorStatusPhaseDeleted)

	// reconcile deleting connectors with no deployments
	k.doReconcile(&errs, "deleting", k.reconcileDeleting,
		"desired_state = ? AND phase = ?", dbapi.ConnectorDeleted, dbapi.ConnectorStatusPhaseDeleting)

	// reconcile deleted connectors with no deployments
	k.doReconcile(&errs, "deleted", k.reconcileDeleted,
		"desired_state = ? AND phase IN ?", dbapi.ConnectorDeleted,
		[]string{string(dbapi.ConnectorStatusPhaseAssigning), string(dbapi.ConnectorStatusPhaseDeleted)})

	// reconcile connector updates for assigned connectors that aren't being deleted...
	k.doReconcile(&errs, "updated", k.reconcileConnectorUpdate,
		"version > ? AND phase NOT IN ?", k.lastVersion,
		[]string{string(dbapi.ConnectorStatusPhaseAssigning), string(dbapi.ConnectorStatusPhaseDeleting), string(dbapi.ConnectorStatusPhaseDeleted)})

	return errs
}

func (k *ConnectorManager) ReconcileConnectorCatalogEntry(id string, channel string, connectorChannelConfig *config.ConnectorChannelConfig) *serviceError.ServiceError {

	connectorShardMetadata := dbapi.ConnectorShardMetadata{
		ConnectorTypeId: id,
		Channel:         channel,
	}

	var err error
	connectorShardMetadata.Revision, err = GetShardMetadataRevision(connectorChannelConfig.ShardMetadata)
	if err != nil {
		return serviceError.GeneralError("failed to convert connector type %s, channel %s. Error in loaded connector type shard metadata %+v: %v", id, channel, connectorChannelConfig.ShardMetadata, err.Error())
	}
	connectorShardMetadata.ShardMetadata, err = json.Marshal(connectorChannelConfig.ShardMetadata)
	if err != nil {
		return serviceError.GeneralError("failed to convert connector type %s, channel %s: %v", id, channel, err.Error())
	}

	// We store connector type channels so that we can track changes and trigger redeployment of
	// associated connectors upon connector type channel changes.
	_, serr := k.connectorTypesService.PutConnectorShardMetadata(&connectorShardMetadata)
	if serr != nil {
		return serr
	}

	return nil
}

func GetShardMetadataRevision(connectorShardMetadata map[string]interface{}) (int64, error) {
	revision, connectorRevisionFound := connectorShardMetadata["connector_revision"]
	if connectorRevisionFound {
		floatRevision, isfloat64 := revision.(float64)
		if isfloat64 {
			return int64(floatRevision), nil
		} else {
			return 0, errors.Errorf("connector_revision in shard metadata was not an int but a %v", reflect.TypeOf(revision).Kind())
		}
	} else {
		return 0, errors.Errorf("connector_revision not found in shard metadata")
	}
}

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

	shardMetadata, err := k.connectorTypesService.GetLatestConnectorShardMetadata(connector.ConnectorTypeId, connector.Channel)
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
		ConnectorID:              connector.ID,
		ClusterID:                namespace.ClusterId,
		NamespaceID:              namespace.ID,
		ConnectorVersion:         connector.Version,
		ConnectorShardMetadataID: shardMetadata.ID,
		Status:                   dbapi.ConnectorDeploymentStatus{},
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

func (k *ConnectorManager) reconcileDeleting(ctx context.Context, connector *dbapi.Connector) error {
	_, err := k.connectorClusterService.GetDeploymentByConnectorId(ctx, connector.ID)
	if err != nil {
		if err.Is404() {
			// set namespace id to nil
			if err := k.db.New().Model(&connector).Where("id = ?", connector.ID).
				Update("namespace_id", nil).Error; err != nil {
				return errors.Wrapf(err, "failed to update namespace_id for connector %s", connector.ID)
			}
			// set status to `deleted`
			connector.Status.Phase = dbapi.ConnectorStatusPhaseDeleted
			if err = k.connectorService.SaveStatus(ctx, connector.Status); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *dbapi.Connector) error {
	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return err
	}
	return nil
}

func (k *ConnectorManager) reconcileConnectorUpdate(ctx context.Context, connector *dbapi.Connector) (err error) {

	// Get the deployment for the connector...
	deployment, serr := k.connectorClusterService.GetDeploymentByConnectorId(ctx, connector.ID)
	if serr != nil {
		err = serr
	} else {
		// we may need to update the deployment due to connector change.
		if deployment.ConnectorVersion != connector.Version {
			deployment.ConnectorVersion = connector.Version
			if serr = k.connectorClusterService.SaveDeployment(ctx, &deployment); serr != nil {
				err = errors.Wrapf(serr, "failed to update connector version in deployment for connector %s", connector.ID)
			}
		}
	}

	if cerr := db.AddPostCommitAction(ctx, func() {
		k.lastVersion = connector.Version
	}); cerr != nil {
		glog.Errorf("failed to AddPostCommitAction to save lastVersion %d: %v", connector.Version, cerr.Error())
		if err == nil {
			err = cerr
		} else {
			err = errors.Errorf("Multiple errors in reconciling connector %s: %s; %s", connector.ID, err, cerr)
		}
	}

	return err
}

func (k *ConnectorManager) doReconcile(errs *[]error, reconcilePhase string, reconcileFunc func(ctx context.Context, connector *dbapi.Connector) error, query string, args ...interface{}) {
	var count int64
	var serviceErrs []error
	glog.V(5).Infof("Reconciling %s connectors...", reconcilePhase)
	if serviceErrs = k.connectorService.ForEach(func(connector *dbapi.Connector) *serviceError.ServiceError {
		return InDBTransaction(k.ctx, func(ctx context.Context) error {
			if err := reconcileFunc(ctx, connector); err != nil {
				glog.Errorf("failed to reconcile %s connector %s in phase %s: %v", reconcilePhase,
					connector.ID, connector.Status.Phase, err)
				return err
			}
			count++
			return nil
		})
	}, query, args...); len(serviceErrs) > 0 {
		*errs = append(*errs, serviceErrs...)
	}
	if count == 0 && len(serviceErrs) == 0 {
		glog.V(5).Infof("No %s connectors", reconcilePhase)
	} else {
		glog.V(5).Infof("Reconciled %d %s connectors with %d errors", count, reconcilePhase, len(serviceErrs))
	}
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
