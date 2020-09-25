package workers

import (
	"fmt"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"time"

	"github.com/golang/glog"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

const (
	repeatInterval = 30 * time.Second
)

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	ocmClient      ocm.Client
	clusterService services.ClusterService
	timer          *time.Timer
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService, ocmClient ocm.Client) *ClusterManager {
	return &ClusterManager{
		ocmClient:      ocmClient,
		clusterService: clusterService,
	}
}

// Start initializes the cluster manager to reconcile osd clusters
func (c *ClusterManager) Start() {
	glog.Infoln("Starting cluster manager")

	// start reconcile immediately and then on every repeat interval
	c.reconcile()

	c.timer = time.NewTimer(repeatInterval)
	for range c.timer.C {
		c.reconcile()
		c.reset() // timer reset, should always be the last task
	}
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	c.timer.Stop()
}

// reset resets the timer to ensure that its invoked only after the new interval period elapses.
func (c *ClusterManager) reset() {
	c.timer.Reset(repeatInterval)
}

func (c *ClusterManager) reconcile() {
	glog.Infoln("reconciling clusters")

	// reconcile the status of existing clusters in a non-ready state

	provisioningClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		glog.Errorf("failed to list pending clusters: %s", listErr.Error())
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for _, provisioningCluster := range provisioningClusters {
		reconciledCluster, err := c.reconcileClusterStatus(&provisioningCluster)
		if err != nil {
			glog.Errorf("failed to reconcile cluster %s status: %s", provisioningCluster.ID, err.Error())
			continue
		}
		glog.Infof("reconciled cluster %s state", reconciledCluster.ID)
	}

	// continue processing clusters
}

// reconcileClusterStatus updates the provided clusters stored status to reflect it's current state
func (c *ClusterManager) reconcileClusterStatus(cluster *api.Cluster) (*api.Cluster, error) {
	// get current cluster state, if not pending, update
	clusterStatus, err := c.ocmClient.GetClusterStatus(cluster.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s status: %w", cluster.ClusterID, err)
	}
	needsUpdate := false
	if cluster.Status == "" {
		cluster.Status = api.ClusterProvisioning
		needsUpdate = true
	}
	// if cluster state is ready, update the local cluster state
	if clusterStatus.State() == v1.ClusterStateReady {
		cluster.Status = api.ClusterProvisioned
		needsUpdate = true
	}
	// if cluster state is error, update the local cluster state
	if clusterStatus.State() == v1.ClusterStateError {
		cluster.Status = api.ClusterFailed
		needsUpdate = true
	}
	// if cluster is neither ready nor in an error state, assume it's pending
	if needsUpdate {
		if err = c.clusterService.UpdateStatus(cluster.ID, cluster.Status); err != nil {
			return nil, fmt.Errorf("failed to update local cluster %s status: %w", cluster.ID, err)
		}
	}
	return cluster, nil
}
