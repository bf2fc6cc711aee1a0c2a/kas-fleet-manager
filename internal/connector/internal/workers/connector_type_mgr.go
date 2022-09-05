package workers

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/golang/glog"
	"github.com/google/uuid"
)

const checkCatalogEntriesDuration = 5 * time.Second

// ConnectorTypeManager represents a connector manager that reconciles connector types at startup
type ConnectorTypeManager struct {
	workers.BaseWorker
	connectorClusterService services.ConnectorClusterService
	connectorTypesService   services.ConnectorTypesService
	startupReconcileDone    bool
	startupReconcileWG      sync.WaitGroup
}

// NewApiServerReadyCondition is used to inject a server.ApiServerReadyCondition into the server.ApiServer
// so that it waits for the ConnectorTypeManager to have completed a startup reconcile before accepting http requests.
func NewApiServerReadyCondition(cm *ConnectorTypeManager) server.ApiServerReadyCondition {
	return &cm.startupReconcileWG
}

// NewConnectorTypeManager creates a new connector type manager
func NewConnectorTypeManager(
	connectorTypesService services.ConnectorTypesService,
	connectorService services.ConnectorsService,
	connectorClusterService services.ConnectorClusterService,
	vaultService vault.VaultService,
	db *db.ConnectionFactory,
	reconciler workers.Reconciler,
	env *environments.Env,
) *ConnectorTypeManager {
	result := &ConnectorTypeManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "connector_type",
			Reconciler: reconciler,
		},
		connectorClusterService: connectorClusterService,
		connectorTypesService:   connectorTypesService,
		startupReconcileDone:    false,
	}

	// The release of this waiting group signal the http service to start serving request
	// this needs to be done across multiple instances of fleetmanager running,
	// and yet just one instance of ConnectorTypeManager of those multiple fleet manager will run the reconcile loop.
	// The release of the waiting group must then be done outside the reconcile loop,
	// the condition is checked in runStartupReconcileCheckWorker().
	result.startupReconcileWG.Add(1)

	// Mark startupReconcileWG as done in a separate goroutine instead of in worker reconcile
	// this is required to allow multiple instances of fleetmanager to startup.
	result.runStartupReconcileCheckWorker()
	return result
}

// Start initializes the connector manager to reconcile connector requests
func (k *ConnectorTypeManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ConnectorTypeManager) Stop() {
	k.StopWorker(k)
}

// HasTerminated indicates whether the worker should be stopped and terminated
func (k *ConnectorTypeManager) HasTerminated() bool {
	return k.startupReconcileDone
}

func (k *ConnectorTypeManager) Reconcile() []error {
	if !k.startupReconcileDone {
		glog.V(5).Infoln("Reconciling startup connector catalog updates...")

		// the assumption here is that this runs on one instance only of fleetmanager,
		// runs only at startup and while requests are not being served
		if err := k.connectorTypesService.DeleteUnusedAndNotInCatalog(); err != nil {
			return []error{err}
		}

		// We only need to reconcile channel updates once per process startup since,
		// configured channel settings are only loaded on startup.
		// These operations, once completed successfully, make the condition at runStartupReconcileCheckWorker() to pass
		// practically starting the serving of requests from the service.
		// IMPORTANT: Everything that should run before the first request is served should happen before this
		if err := k.connectorTypesService.ForEachConnectorCatalogEntry(k.ReconcileConnectorCatalogEntry); err != nil {
			return []error{err}
		}

		if err := k.connectorClusterService.CleanupDeployments(); err != nil {
			return []error{err}
		}

		k.startupReconcileDone = true
		glog.V(5).Infoln("Catalog updates processed")
	}

	return nil
}

func (k *ConnectorTypeManager) ReconcileConnectorCatalogEntry(id string, channel string, connectorChannelConfig *config.ConnectorChannelConfig) *serviceError.ServiceError {

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

func (k *ConnectorTypeManager) runStartupReconcileCheckWorker() {
	go func() {
		for !k.startupReconcileDone {
			glog.V(5).Infoln("Waiting for startup connector catalog updates...")
			// this check that ConnectorTypes in the current configured catalog have the same checksum of the one
			// stored in the db (comparing them by id).
			done, err := k.connectorTypesService.CatalogEntriesReconciled()
			if err != nil {
				glog.Errorf("Error checking catalog entry checksums: %s", err)
			} else if done {
				k.startupReconcileDone = true
			} else {
				// wait another 5 seconds to check
				time.Sleep(checkCatalogEntriesDuration)
			}
		}
		glog.V(5).Infoln("Wait for connector catalog updates done!")
		k.startupReconcileWG.Done()
	}()
}
