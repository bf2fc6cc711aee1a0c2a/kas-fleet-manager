package phase

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/looplab/fsm"
)

type ClusterOperation string

const (
	ConnectCluster    ClusterOperation = "connect"
	DisconnectCluster ClusterOperation = "disconnect"
	DeleteCluster     ClusterOperation = "delete"
)

// ClusterFSM handles cluster phase changes
type ClusterFSM struct {
	Cluster *dbapi.ConnectorCluster
	fsm     *fsm.FSM
}

var clusterEvents = []fsm.EventDesc{
	{Name: string(ConnectCluster), Src: []string{string(dbapi.ConnectorClusterPhaseDisconnected), string(dbapi.ConnectorClusterPhaseReady)}, Dst: string(dbapi.ConnectorClusterPhaseReady)},
	{Name: string(ConnectCluster), Src: []string{string(dbapi.ConnectorClusterPhaseDeleting)}, Dst: string(dbapi.ConnectorClusterPhaseDeleting)},
	{Name: string(DisconnectCluster), Src: []string{string(dbapi.ConnectorClusterPhaseDisconnected), string(dbapi.ConnectorClusterPhaseReady)}, Dst: string(dbapi.ConnectorClusterPhaseDisconnected)},
	{Name: string(DeleteCluster), Src: []string{string(dbapi.ConnectorClusterPhaseDisconnected), string(dbapi.ConnectorClusterPhaseReady), string(dbapi.ConnectorClusterPhaseDeleting)}, Dst: string(dbapi.ConnectorClusterPhaseDeleting)},
}

func NewClusterFSM(cluster *dbapi.ConnectorCluster) *ClusterFSM {
	return &ClusterFSM{
		Cluster: cluster,
		fsm:     fsm.NewFSM(string(cluster.Status.Phase), clusterEvents, nil),
	}
}

// Perform tries to perform the given operation and updates the cluster phase,
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in cluster's present phase
func (c *ClusterFSM) Perform(operation ClusterOperation) (bool, *errors.ServiceError) {
	// make sure FSM phase is current
	c.fsm.SetState(string(c.Cluster.Status.Phase))
	if err := c.fsm.Event(string(operation)); err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
			return false, nil
		default:
			return false, errors.BadRequest("Cannot perform Cluster operation [%s] because %s",
				operation, err)
		}
	}

	c.Cluster.Status.Phase = dbapi.ConnectorClusterPhaseEnum(c.fsm.Current())
	return true, nil
}

// PerformClusterOperation is a utility method to change a cluster's phase and call optional functions to save updated phase
// first return value is true if the phase was changed and
// second value is an error if operation is not permitted in cluster's present phase
func PerformClusterOperation(cluster *dbapi.ConnectorCluster, operation ClusterOperation,
	updatePhase ...func(connectorCluster *dbapi.ConnectorCluster) *errors.ServiceError) (updated bool, err *errors.ServiceError) {

	if updated, err = NewClusterFSM(cluster).Perform(operation); len(updatePhase) > 0 && err == nil && updated {
		for _, f := range updatePhase {
			err = f(cluster)
			if err != nil {
				break
			}
		}
	}
	return updated, err
}
