package workers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"sync"
	"time"

	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"

	"github.com/golang/glog"
	"github.com/google/uuid"
)

// ProcessorTypeManager represents a connector manager that reconciles connector types at startup
type ProcessorTypeManager struct {
	workers.BaseWorker
	processorTypesService services.ProcessorTypesService
	startupReconcileDone  bool
	startupReconcileWG    sync.WaitGroup
}

// NewApiServerReadyConditionForProcessors is used to inject a server.ApiServerReadyCondition into the server.ApiServer
// so that it waits for the ProcessorTypeManager to have completed a startup reconcile before accepting http requests.
func NewApiServerReadyConditionForProcessors(pm *ProcessorTypeManager) server.ApiServerReadyCondition {
	return &pm.startupReconcileWG
}

// NewProcessorTypeManager creates a new processor type manager
func NewProcessorTypeManager(
	processorTypesService services.ProcessorTypesService,
	reconciler workers.Reconciler,
) *ProcessorTypeManager {
	result := &ProcessorTypeManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "processor_type",
			Reconciler: reconciler,
		},
		processorTypesService: processorTypesService,
		startupReconcileDone:  false,
	}

	// The release of this waiting group signal the http service to start serving request
	// this needs to be done across multiple instances of fleetmanager running,
	// and yet just one instance of ProcessorTypeManager of those multiple fleet manager will run the reconcile loop.
	// The release of the waiting group must then be done outside the reconcile loop,
	// the condition is checked in runStartupReconcileCheckWorker().
	result.startupReconcileWG.Add(1)

	// Mark startupReconcileWG as done in a separate goroutine instead of in worker reconcile
	// this is required to allow multiple instances of fleetmanager to startup.
	result.runStartupReconcileCheckWorker()
	return result
}

// Start initializes the connector manager to reconcile connector requests
func (k *ProcessorTypeManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ProcessorTypeManager) Stop() {
	k.StopWorker(k)
}

// HasTerminated indicates whether the worker should be stopped and terminated
func (k *ProcessorTypeManager) HasTerminated() bool {
	return k.startupReconcileDone
}

func (k *ProcessorTypeManager) Reconcile() []error {
	if !k.startupReconcileDone {
		glog.V(5).Infoln("Reconciling startup processor catalog updates...")

		// the assumption here is that this runs on one instance only of fleetmanager,
		// runs only at startup and while requests are not being served
		// this call handles types that are not in catalog anymore,
		// removing unused types and marking used types as deprecated
		// TODO. See ConnectorTypeManager

		// We only need to reconcile channel updates once per process startup since,
		// configured channel settings are only loaded on startup.
		// These operations, once completed successfully, make the condition at runStartupReconcileCheckWorker() to pass
		// practically starting the serving of requests from the service.
		// IMPORTANT: Everything that should run before the first request is served should happen before this
		// TODO. See ConnectorTypeManager
		if err := k.ReconcileProcessorCatalog(); err != nil {
			return []error{err}
		}

		k.startupReconcileDone = true
		glog.V(5).Infoln("Catalog updates processed")
	}

	return nil
}

func (k *ProcessorTypeManager) ReconcileProcessorCatalog() *serviceError.ServiceError {
	// Hard coded until the Processor Catalog is available
	processorShardMetadata := dbapi.ProcessorShardMetadata{
		ID:             1,
		Revision:       0,
		LatestRevision: nil,
		ShardMetadata:  api.JSON(`{"dummy":"metadata"}`),
	}

	if _, err := k.processorTypesService.PutProcessorShardMetadata(&processorShardMetadata); err != nil {
		return err
	}

	return nil
}

func (k *ProcessorTypeManager) runStartupReconcileCheckWorker() {
	go func() {
		// wait 5 seconds to check to allow the reconcile function to run...
		time.Sleep(checkCatalogEntriesDuration)
		for !k.startupReconcileDone {
			glog.V(5).Infoln("Waiting for startup processor catalog updates...")
			// This will check the Processor Catalog has been loaded, parsed and stored
			// See ConnectorTypeManager for a better implementation (when the Processor Catalog is available)
			k.startupReconcileDone = true
		}
		glog.V(5).Infoln("Wait for processor catalog updates done!")
		k.startupReconcileWG.Done()
	}()
}
