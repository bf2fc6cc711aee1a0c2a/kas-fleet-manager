package chain

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
)

// ReconcileAction is the interface that must be implemented by each action that the reconciler will execute.
// T - is the type of result the action will generate
type ReconcileAction[T any] interface {
	// PerformJob is the method where the Action implementation will perform the real job
	// * kafkaRequest - the kafka request to reconcile
	// * currentResult - object containing the results that have been produced up to this point into the pipeline
	// Return values
	// * res Result[T] - the result produced by this action
	// * finished bool - returning `true` here will interrupt the pipeline execution
	// * err error - an eventual error. Returning an error always interrupts the pipeline execution
	PerformJob(kafkaRequest *dbapi.KafkaRequest, currentResult ActionResult[T]) (res ActionResult[T], finished bool, err error)
}

// ActionResult contains the result produced by each action
type ActionResult[T any] struct {
	hasValue bool
	value    T
}

func (r *ActionResult[T]) SetValue(value T) {
	r.value = value
	r.hasValue = true
}

func (r *ActionResult[T]) Value() T {
	return r.value
}

func (r *ActionResult[T]) HasValue() bool {
	return r.hasValue
}

// The ReconcileActionRunner is the engine of the action chain. It runs each action one after the other
// passing to each of them the current result so that all the action can concur to create a piece of the
// result. When the chain execution is finished, the result will be complete.
// T is the type of result the ReconcileActionRunner will produce (by running a list of ReconcileAction)
type ReconcileActionRunner[T any] struct {
	actions []ReconcileAction[T]
}

func NewReconcileActionRunner[T any](actions ...ReconcileAction[T]) ReconcileActionRunner[T] {
	return ReconcileActionRunner[T]{
		actions: actions,
	}
}

// Run the chain end return the final result of the execution
// The returned result is the result of the last ReconcileAction executed.
func (r *ReconcileActionRunner[T]) Run(kafkaRequest *dbapi.KafkaRequest) (T, error) {
	finalResult := ActionResult[T]{}
	for _, action := range r.actions {
		res, endPipeline, err := action.PerformJob(kafkaRequest, finalResult)
		if err != nil {
			return finalResult.value, err
		}
		if res.hasValue {
			finalResult.SetValue(res.Value())
		}
		if endPipeline {
			break
		}
	}

	return finalResult.value, nil
}
