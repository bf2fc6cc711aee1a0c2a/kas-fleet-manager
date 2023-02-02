package phase

import (
	"context"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/looplab/fsm"
)

type ProcessorOperation string

const (
	CreateProcessor  ProcessorOperation = "create"
	UpdateProcessor  ProcessorOperation = "update"
	StopProcessor    ProcessorOperation = "stop"
	RestartProcessor ProcessorOperation = "restart"
	DeleteProcessor  ProcessorOperation = "delete"
)

// ProcessorFSM handles processor phase changes within its namespace's current phase
type ProcessorFSM struct {
	namespacePhase dbapi.ConnectorNamespacePhaseEnum
	Processor      *dbapi.Processor
	fsm            *fsm.FSM
}

var processorEvents = map[dbapi.ConnectorNamespacePhaseEnum][]fsm.EventDesc{
	dbapi.ConnectorNamespacePhaseDisconnected: {
		{Name: string(CreateProcessor), Src: []string{string(dbapi.ProcessorReady)}, Dst: string(dbapi.ProcessorReady)},
		{Name: string(UpdateProcessor), Src: []string{string(dbapi.ProcessorReady)}, Dst: string(dbapi.ProcessorReady)},
		{Name: string(UpdateProcessor), Src: []string{string(dbapi.ProcessorStopped)}, Dst: string(dbapi.ProcessorStopped)},
		{Name: string(StopProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped)}, Dst: string(dbapi.ProcessorStopped)},
		{Name: string(DeleteProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped), string(dbapi.ProcessorDeleted)}, Dst: string(dbapi.ProcessorDeleted)},
	},
	dbapi.ConnectorNamespacePhaseReady: {
		{Name: string(CreateProcessor), Src: []string{string(dbapi.ProcessorReady)}, Dst: string(dbapi.ProcessorReady)},
		{Name: string(UpdateProcessor), Src: []string{string(dbapi.ProcessorReady)}, Dst: string(dbapi.ProcessorReady)},
		{Name: string(UpdateProcessor), Src: []string{string(dbapi.ProcessorStopped)}, Dst: string(dbapi.ProcessorStopped)},
		{Name: string(RestartProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped)}, Dst: string(dbapi.ProcessorReady)},
		{Name: string(StopProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped)}, Dst: string(dbapi.ProcessorStopped)},
		{Name: string(DeleteProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped), string(dbapi.ProcessorDeleted)}, Dst: string(dbapi.ProcessorDeleted)},
	},
	dbapi.ConnectorNamespacePhaseDeleting: {
		{Name: string(DeleteProcessor), Src: []string{string(dbapi.ProcessorReady), string(dbapi.ProcessorStopped), string(dbapi.ProcessorDeleted)}, Dst: string(dbapi.ProcessorDeleted)},
	},
}

var ProcessorStartingPhase = map[ProcessorOperation]dbapi.ProcessorStatusPhase{
	CreateProcessor:  dbapi.ProcessorStatusPhasePreparing,
	RestartProcessor: dbapi.ProcessorStatusPhasePrepared,
	StopProcessor:    dbapi.ProcessorStatusPhasePrepared,
	UpdateProcessor:  dbapi.ProcessorStatusPhaseUpdating,
	DeleteProcessor:  dbapi.ProcessorStatusPhaseDeleting,
}

func NewProcessorFSM(namespace *dbapi.ConnectorNamespace, processor *dbapi.Processor) *ProcessorFSM {
	return &ProcessorFSM{
		Processor:      processor,
		namespacePhase: namespace.Status.Phase,
		fsm:            fsm.NewFSM(string(processor.DesiredState), processorEvents[namespace.Status.Phase], nil),
	}
}

// Perform tries to perform the given operation and updates the processor desired state,
// first return value is true if the state was changed and
// second value is an error if operation is not permitted in processor's present state
func (c *ProcessorFSM) Perform(operation ProcessorOperation) (bool, *errors.ServiceError) {
	if err := c.fsm.Event(context.TODO(), string(operation)); err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
			return false, nil
		default:
			return false, errors.BadRequest("cannot perform Processor operation [%s] in Namespace phase [%s] because %s",
				operation, c.namespacePhase, err)
		}
	}

	c.Processor.DesiredState = dbapi.ProcessorDesiredState(c.fsm.Current())
	if phase, ok := ProcessorStartingPhase[operation]; ok {
		c.Processor.Status.Phase = phase
	}
	return true, nil
}

// PerformProcessorOperation is a utility method to change a processor's phase
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in processor's present phase
func PerformProcessorOperation(namespace *dbapi.ConnectorNamespace, processor *dbapi.Processor, operation ProcessorOperation,
	updatePhase ...func(processor *dbapi.Processor) *errors.ServiceError) (updated bool, err *errors.ServiceError) {

	if updated, err = NewProcessorFSM(namespace, processor).Perform(operation); len(updatePhase) > 0 && err == nil && updated {
		for _, f := range updatePhase {
			err = f(processor)
			if err != nil {
				break
			}
		}
	}
	return updated, err
}
