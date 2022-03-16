package phase

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/looplab/fsm"
)

type ConnectorOperation string

const (
	CreateConnector   ConnectorOperation = "create"
	AssignConnector   ConnectorOperation = "assign"
	UnassignConnector ConnectorOperation = "unassign"
	UpdateConnector   ConnectorOperation = "update"
	StopConnector     ConnectorOperation = "stop"
	DeleteConnector   ConnectorOperation = "delete"
)

// ConnectorFSM handles connector phase changes within it's cluster's current phase
type ConnectorFSM struct {
	namespacePhase dbapi.ConnectorNamespacePhaseEnum
	Connector      *dbapi.Connector
	fsm            *fsm.FSM
}

var connectorEvents = map[dbapi.ConnectorNamespacePhaseEnum][]fsm.EventDesc{
	dbapi.ConnectorNamespacePhaseDisconnected: {
		{Name: string(CreateConnector), Src: []string{string(dbapi.ConnectorUnassigned),string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(AssignConnector), Src: []string{string(dbapi.ConnectorUnassigned)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(UnassignConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped)}, Dst: string(dbapi.ConnectorUnassigned)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorUnassigned)}, Dst: string(dbapi.ConnectorUnassigned)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorStopped)}, Dst: string(dbapi.ConnectorStopped)},
		{Name: string(StopConnector), Src: []string{string(dbapi.ConnectorStopped), string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorStopped)},
		{Name: string(DeleteConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped), string(dbapi.ConnectorDeleted)}, Dst: string(dbapi.ConnectorDeleted)},
	},
	dbapi.ConnectorNamespacePhaseReady:    {
		{Name: string(CreateConnector), Src: []string{string(dbapi.ConnectorUnassigned),string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(AssignConnector), Src: []string{string(dbapi.ConnectorUnassigned)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(UnassignConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped)}, Dst: string(dbapi.ConnectorUnassigned)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorUnassigned)}, Dst: string(dbapi.ConnectorUnassigned)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorReady)},
		{Name: string(UpdateConnector), Src: []string{string(dbapi.ConnectorStopped)}, Dst: string(dbapi.ConnectorStopped)},
		{Name: string(StopConnector), Src: []string{string(dbapi.ConnectorStopped), string(dbapi.ConnectorReady)}, Dst: string(dbapi.ConnectorStopped)},
		{Name: string(DeleteConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped), string(dbapi.ConnectorDeleted)}, Dst: string(dbapi.ConnectorDeleted)},
	},
	dbapi.ConnectorNamespacePhaseDeleting: {
		{Name: string(UnassignConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped)}, Dst: string(dbapi.ConnectorUnassigned)},
		{Name: string(DeleteConnector), Src: []string{string(dbapi.ConnectorUnassigned), string(dbapi.ConnectorReady), string(dbapi.ConnectorStopped), string(dbapi.ConnectorDeleted)}, Dst: string(dbapi.ConnectorDeleted)},
	},
}

var startingPhase = map[dbapi.ConnectorDesiredState]dbapi.ConnectorStatusPhase{
	dbapi.ConnectorUnassigned: dbapi.ConnectorStatusPhaseAssigning,
	dbapi.ConnectorDeleted:    dbapi.ConnectorStatusPhaseDeleting,
}

func NewConnectorFSM(namespace *dbapi.ConnectorNamespace, connector *dbapi.Connector) *ConnectorFSM {
	return &ConnectorFSM{
		Connector:      connector,
		namespacePhase: namespace.Status.Phase,
		fsm:            fsm.NewFSM(string(connector.DesiredState), connectorEvents[namespace.Status.Phase], nil),
	}
}

// Perform tries to perform the given operation and updates the connector desired state,
// first return value is true if the state was changed and
// second value is an error if operation is not permitted in connector's present state
func (c *ConnectorFSM) Perform(operation ConnectorOperation) (bool, *errors.ServiceError) {
	if err := c.fsm.Event(string(operation)); err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
			return false, nil
		default:
			return false, errors.BadRequest("Cannot perform Connector operation [%s] in Namespace phase [%s] because %s",
				operation, c.namespacePhase, err)
		}
	}

	c.Connector.DesiredState = dbapi.ConnectorDesiredState(c.fsm.Current())
	if phase, ok := startingPhase[c.Connector.DesiredState]; ok {
		c.Connector.Status.Phase = phase
	}
	return true, nil
}

// PerformConnectorOperation is a utility method to change a connector's phase
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in connector's present phase
func PerformConnectorOperation(namespace *dbapi.ConnectorNamespace, connector *dbapi.Connector, operation ConnectorOperation,
	updatePhase ...func(connector *dbapi.Connector) *errors.ServiceError) (updated bool, err *errors.ServiceError) {

	if updated, err = NewConnectorFSM(namespace, connector).Perform(operation); len(updatePhase) > 0 && err == nil && updated {
		for _, f := range updatePhase {
			err = f(connector)
			if err != nil {
				break
			}
		}
	}
	return updated, err
}
