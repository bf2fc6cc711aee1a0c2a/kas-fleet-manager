package workers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ProcessorManager represents a processor manager that periodically reconciles processor requests
type ProcessorManager struct {
	workers.BaseWorker
	processorsService           services.ProcessorsService
	processorDeploymentsService services.ProcessorDeploymentsService
	namespaceService            services.ConnectorNamespaceService
	lastVersion                 int64
	db                          *db.ConnectionFactory
	ctx                         context.Context
}

// NewProcessorManager creates a new processor manager
func NewProcessorManager(
	processorsService services.ProcessorsService,
	processorDeploymentsService services.ProcessorDeploymentsService,
	namespaceService services.ConnectorNamespaceService,
	db *db.ConnectionFactory,
	reconciler workers.Reconciler,
) *ProcessorManager {
	result := &ProcessorManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "processor",
			Reconciler: reconciler,
		},
		processorsService:           processorsService,
		processorDeploymentsService: processorDeploymentsService,
		namespaceService:            namespaceService,
		db:                          db,
	}

	return result
}

// Start initializes the processor manager to reconcile processor requests
func (m *ProcessorManager) Start() {
	m.StartWorker(m)
}

// Stop causes the process for reconciling processor requests to stop.
func (m *ProcessorManager) Stop() {
	m.StopWorker(m)
}

func (m *ProcessorManager) Reconcile() []error {
	glog.V(5).Infoln("Reconciling processors...")
	var errs []error

	if m.ctx == nil {
		ctx, err := m.db.NewContext(context.Background())
		if err != nil {
			return []error{err}
		}
		m.ctx = ctx
	}

	// Create a ProcessorDeployment record and transition from Preparing to Prepared
	// Since Processors must have a Namespace that must pre-exist we could either:
	// - Create the ProcessorDeployment record when creating the Processor and remove this code
	// - Merge ProcessorDeployment fields into Processor and remove this code
	m.doReconcile(&errs, "preparing", m.reconcilePreparing,
		"desired_state = ? AND phase = ? AND processors.namespace_id IS NOT NULL", dbapi.ProcessorReady, dbapi.ProcessorStatusPhasePreparing)

	// Delete ProcessorDeployment once it has transitioned to Deleted by FleetShard
	m.doReconcile(&errs, "deprovisioning", m.reconcileDeprovisioning,
		"desired_state = ? AND phase = ?", dbapi.ProcessorDeleted, dbapi.ProcessorStatusPhaseDeprovisioning)

	// Delete Processor once it has transitioned to Deleted
	m.doReconcile(&errs, "deleted", m.reconcileDeleted,
		"desired_state = ? AND phase IN ?", dbapi.ProcessorDeleted,
		[]string{string(dbapi.ProcessorStatusPhasePreparing), string(dbapi.ProcessorStatusPhaseDeleted)})

	// Ensure the ProcessorDeployment version matches that of the Processor
	// Quite what the relevance of this is I don't know; but I suspect it has something to do with optimistic locks on Processor
	m.doReconcile(&errs, "updated", m.reconcileProcessorUpdate,
		"version > ? AND phase NOT IN ?", m.lastVersion,
		[]string{string(dbapi.ProcessorStatusPhasePreparing), string(dbapi.ProcessorStatusPhaseDeleting), string(dbapi.ProcessorStatusPhaseDeleted)})

	return errs
}

func (m *ProcessorManager) reconcilePreparing(ctx context.Context, processor *dbapi.Processor) error {
	var namespace *dbapi.ConnectorNamespace
	// Processors reside in the same namespace as the Connectors to which they can integrate.
	namespace, err := m.namespaceService.Get(ctx, processor.NamespaceId)
	if err != nil {
		return errors.Wrapf(err, "Failed to find Namespace for Processor %s", processor.ID)
	}
	if namespace == nil {
		return errors.Wrapf(err, "Failed to find Namespace for Processor %s", processor.ID)
	}

	// Processors don't really support different metadata but include this until we decide how we want to manage
	shardMetadata, err := m.processorsService.GetLatestProcessorShardMetadata()
	if err != nil {
		return errors.Wrapf(err, "Failed to get latest Shard Metadata for Processor %s", processor.ID)
	}

	var status = dbapi.ProcessorStatus{}
	status.ID = processor.ID
	status.NamespaceID = namespace.ID
	status.Phase = dbapi.ProcessorStatusPhasePrepared
	if err = m.processorsService.SaveStatus(ctx, status); err != nil {
		return errors.Wrapf(err, "Failed to update Processor status %s with Namespace details", status.ID)
	}

	deployment := dbapi.ProcessorDeployment{
		Model: db.Model{
			ID: api.NewID(),
		},
		ProcessorID:              processor.ID,
		ClusterID:                namespace.ClusterId,
		NamespaceID:              namespace.ID,
		ProcessorVersion:         processor.Version,
		ProcessorShardMetadataID: shardMetadata.ID,
		Status:                   dbapi.ProcessorDeploymentStatus{},
	}

	if err = m.processorDeploymentsService.Create(ctx, &deployment); err != nil {
		return errors.Wrapf(err, "Failed to create ProcessorDeployment for Processor %s", processor.ID)
	}

	return nil
}

func (m *ProcessorManager) reconcileDeprovisioning(ctx context.Context, processor *dbapi.Processor) error {
	// Has the ProcessorDeployment been deleted by the FleetShard
	processorDeployment, err := m.processorDeploymentsService.GetByProcessorId(ctx, processor.ID)
	if processorDeployment.Status.Phase != dbapi.ProcessorStatusPhaseDeleted {
		return nil
	}

	// If the ProcessorDeployment has been deleted clean up the database and mark Processor for deletion
	if err != nil {
		return errors.Wrapf(err, "Failed to find ProcessorDeployment for Processor %s", processor.ID)
	}
	if err := m.processorDeploymentsService.Delete(ctx, processorDeployment.ID); err != nil {
		return errors.Wrapf(err, "Failed to delete ProcessorDeployment for Processor %s", processor.ID)
	}

	// Set Processor status to Deleted
	processor.Status.Phase = dbapi.ProcessorStatusPhaseDeleted
	if err = m.processorsService.SaveStatus(ctx, processor.Status); err != nil {
		return err
	}

	return nil
}

func (m *ProcessorManager) reconcileDeleted(ctx context.Context, processor *dbapi.Processor) error {
	if err := m.processorsService.Delete(ctx, processor.ID); err != nil {
		return err
	}
	return nil
}

func (m *ProcessorManager) reconcileProcessorUpdate(ctx context.Context, processor *dbapi.Processor) (err error) {
	// Get the deployment for the processor...
	deployment, serr := m.processorDeploymentsService.GetByProcessorId(ctx, processor.ID)
	if serr != nil {
		err = serr
	} else {
		// we may need to update the deployment due to processor change.
		if deployment.ProcessorVersion != processor.Version {
			deployment.ProcessorVersion = processor.Version
			if serr = m.processorDeploymentsService.Update(ctx, deployment); serr != nil {
				err = errors.Wrapf(serr, "failed to update processor version in deployment for processor %s", processor.ID)
			}
		}
	}

	if cerr := db.AddPostCommitAction(ctx, func() {
		m.lastVersion = processor.Version
	}); cerr != nil {
		glog.Errorf("Failed to AddPostCommitAction to save lastVersion %d: %v", processor.Version, cerr.Error())
		if err == nil {
			err = cerr
		} else {
			err = errors.Errorf("Multiple errors in reconciling processor %s: %s; %s", processor.ID, err, cerr)
		}
	}

	return err
}

func (m *ProcessorManager) doReconcile(errs *[]error, reconcilePhase string, reconcileFunc func(ctx context.Context, processor *dbapi.Processor) error, query string, args ...interface{}) {
	var count int64
	var serviceErrs []error
	glog.V(5).Infof("Reconciling %s processors...", reconcilePhase)
	if serviceErrs = m.processorsService.ForEach(func(processor *dbapi.Processor) *serviceError.ServiceError {
		return InDBTransaction(m.ctx, func(ctx context.Context) error {
			if err := reconcileFunc(ctx, processor); err != nil {
				glog.Errorf("Failed to reconcile %s processor %s in phase %s: %v", reconcilePhase,
					processor.ID, processor.Status.Phase, err)
				return err
			}
			count++
			return nil
		})
	}, query, args...); len(serviceErrs) > 0 {
		*errs = append(*errs, serviceErrs...)
	}
	if count == 0 && len(serviceErrs) == 0 {
		glog.V(5).Infof("No %s processor", reconcilePhase)
	} else {
		glog.V(5).Infof("Reconciled %d %s processor with %d errors", count, reconcilePhase, len(serviceErrs))
	}
}
