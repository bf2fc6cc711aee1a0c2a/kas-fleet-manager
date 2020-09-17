package workers

import (
	"time"

	"github.com/golang/glog"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

const (
	repeatInterval = 30 * time.Second
)

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	clusterService services.ClusterService
	timer          *time.Timer
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService) *ClusterManager {
	return &ClusterManager{
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

	// Add logic here for reconciling clusters
}
