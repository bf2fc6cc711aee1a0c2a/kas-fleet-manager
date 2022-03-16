package phase

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/looplab/fsm"
)

type NamespaceOperation string

const (
	CreateNamespace     NamespaceOperation = "create"
	ConnectNamespace    NamespaceOperation = "connect"
	DisconnectNamespace NamespaceOperation = "disconnect"
	DeleteNamespace     NamespaceOperation = "delete"
)

// NamespaceFSM handles namespace phase changes within it's cluster's current phase
type NamespaceFSM struct {
	clusterPhase dbapi.ConnectorClusterPhaseEnum
	Namespace    *dbapi.ConnectorNamespace
	fsm          *fsm.FSM
}

var namespaceEvents = map[dbapi.ConnectorClusterPhaseEnum][]fsm.EventDesc{
	dbapi.ConnectorClusterPhaseDisconnected: {
		{Name: string(CreateNamespace),Src: []string{""},Dst: string(dbapi.ConnectorNamespacePhaseDisconnected)},
		{Name: string(DisconnectNamespace),Src: []string{string(dbapi.ConnectorNamespacePhaseDisconnected), string(dbapi.ConnectorNamespacePhaseReady)},Dst: string(dbapi.ConnectorNamespacePhaseDisconnected)},
		{Name: string(DeleteNamespace),Src: []string{string(dbapi.ConnectorNamespacePhaseDisconnected), string(dbapi.ConnectorNamespacePhaseDeleting)},Dst: string(dbapi.ConnectorNamespacePhaseDeleting)},
	},
	dbapi.ConnectorClusterPhaseReady: {
		{Name: string(CreateNamespace),Src: []string{""},Dst: string(dbapi.ConnectorNamespacePhaseDisconnected)},
		{Name: string(ConnectNamespace),Src: []string{string(dbapi.ConnectorNamespacePhaseDisconnected), string(dbapi.ConnectorNamespacePhaseReady)},Dst: string(dbapi.ConnectorNamespacePhaseReady)},
		{Name: string(ConnectNamespace),Src: []string{string(dbapi.ConnectorNamespacePhaseDeleting)}, Dst: string(dbapi.ConnectorNamespacePhaseDeleting)},
		{Name: string(DisconnectNamespace), Src: []string{string(dbapi.ConnectorNamespacePhaseDisconnected), string(dbapi.ConnectorNamespacePhaseReady)}, Dst: string(dbapi.ConnectorNamespacePhaseDisconnected)},
		{Name: string(DeleteNamespace), Src: []string{string(dbapi.ConnectorNamespacePhaseDisconnected), string(dbapi.ConnectorNamespacePhaseReady), string(dbapi.ConnectorNamespacePhaseDeleting)}, Dst: string(dbapi.ConnectorNamespacePhaseDeleting)},
	},
	dbapi.ConnectorClusterPhaseDeleting: {
		{Name: string(ConnectNamespace), Src: []string{string(dbapi.ConnectorClusterPhaseDeleting)}, Dst: string(dbapi.ConnectorNamespacePhaseDeleting)},
		{Name: string(DeleteNamespace), Src: []string{string(dbapi.ConnectorNamespacePhaseDeleting)}, Dst: string(dbapi.ConnectorNamespacePhaseDeleting)},
	},
}

func NewNamespaceFSM(cluster *dbapi.ConnectorCluster, namespace *dbapi.ConnectorNamespace) *NamespaceFSM {
	return &NamespaceFSM{
		Namespace:    namespace,
		clusterPhase: cluster.Status.Phase,
		fsm:          fsm.NewFSM(string(namespace.Status.Phase), namespaceEvents[cluster.Status.Phase], nil),
	}
}

// Perform tries to perform the given operation and updates the namespace phase,
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in namespace's present phase
func (c *NamespaceFSM) Perform(operation NamespaceOperation) (bool, *errors.ServiceError) {
	if err := c.fsm.Event(string(operation)); err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
			return false, nil
		default:
			return false, errors.BadRequest("Cannot perform Namespace operation [%s] in Cluster phase [%s] because %s",
				operation, c.clusterPhase, err)
		}
	}

	c.Namespace.Status.Phase = dbapi.ConnectorNamespacePhaseEnum(c.fsm.Current())
	return true, nil
}

// PerformNamespaceOperation is a utility method to change a namespace's phase
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in namespace's present phase
func PerformNamespaceOperation(cluster *dbapi.ConnectorCluster, namespace *dbapi.ConnectorNamespace, operation NamespaceOperation,
	updatePhase ...func(namespace *dbapi.ConnectorNamespace) *errors.ServiceError) (updated bool, err *errors.ServiceError) {

	if updated, err = NewNamespaceFSM(cluster, namespace).Perform(operation); len(updatePhase) > 0 && err == nil && updated {
		for _, f := range updatePhase {
			err = f(namespace)
			if err != nil {
				break
			}
		}
	}

	return updated, err
}
