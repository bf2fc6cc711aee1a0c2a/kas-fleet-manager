package workers

import (
	"fmt"
	"sync"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"

	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

const (
	repeatInterval = 30 * time.Second
)

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	ocmClient             ocm.Client
	clusterService        services.ClusterService
	cloudProvidersService services.CloudProvidersService
	timer                 *time.Timer
	imStop                chan struct{}
	syncTeardown          sync.WaitGroup
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService, cloudProvidersService services.CloudProvidersService, ocmClient ocm.Client) *ClusterManager {
	return &ClusterManager{
		ocmClient:             ocmClient,
		clusterService:        clusterService,
		cloudProvidersService: cloudProvidersService,
	}
}

// Start initializes the cluster manager to reconcile osd clusters
func (c *ClusterManager) Start() {
	c.imStop = make(chan struct{})
	c.syncTeardown.Add(1)
	glog.V(1).Infoln("Starting cluster manager")
	// start reconcile immediately and then on every repeat interval
	c.reconcile()
	ticker := time.NewTicker(repeatInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(1).Infoln("Reconciling OCM clusters")
				c.reconcile()
			case <-c.imStop:
				ticker.Stop()
				defer c.syncTeardown.Done()
				glog.V(1).Infoln("Stopping reconcile loop")
				return
			}
		}
	}()
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	select {
	case <-c.imStop:
		return
	default:
		close(c.imStop)
		c.syncTeardown.Wait()
	}
}

// reset resets the timer to ensure that its invoked only after the new interval period elapses.
func (c *ClusterManager) reset() {
	c.timer.Reset(repeatInterval)
}

func (c *ClusterManager) reconcile() {
	glog.V(5).Infoln("reconciling clusters")

	// reconcile the status of existing clusters in a non-ready state
	cloudProviders, err := c.cloudProvidersService.GetCloudProvidersWithRegions()
	if err != nil {
		glog.Error("Error retrieving cloud providers and regions", err)
	}

	for _, cloudProvider := range cloudProviders {
		cloudProvider.RegionList.Each(func(region *clustersmgmtv1.CloudRegion) bool {
			regionName := region.ID()
			glog.V(10).Infoln("Provider:", cloudProvider.ID, "=>", "Region:", regionName)
			return true
		})

	}

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
		glog.V(5).Infof("reconciled cluster %s state", reconciledCluster.ID)
	}

	/*
	 * Terraforming Provisioned Clusters
	 */
	provisionedClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		glog.Errorf("failed to list provisioned clusters: %s", listErr.Error())
	}

	// process each local provisioned cluster and apply nessecary terraforming
	for _, provisionedCluster := range provisionedClusters {
		addonInstallation, err := c.reconcileStrimziOperator(provisionedCluster.ID)
		if err != nil {
			glog.Errorf("failed to reconcile cluster %s strimzi operator: %s", provisionedCluster.ID, err.Error())
			continue
		}

		// The cluster is ready when the state reports ready
		if addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
			if err = c.clusterService.UpdateStatus(provisionedCluster.ID, api.ClusterReady); err != nil {
				glog.Errorf("failed to update local cluster %s status: %s", provisionedCluster.ID, err.Error())
				continue
			}
		}
		glog.V(5).Infof("reconciled cluster %s terraforming", provisionedCluster.ID)
	}
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
	if clusterStatus.State() == clustersmgmtv1.ClusterStateReady {
		cluster.Status = api.ClusterProvisioned
		needsUpdate = true
	}
	// if cluster state is error, update the local cluster state
	if clusterStatus.State() == clustersmgmtv1.ClusterStateError {
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

// reconcileStrimziOperator installs the Strimzi operator on a provisioned clusters
func (c *ClusterManager) reconcileStrimziOperator(clusterID string) (*clustersmgmtv1.AddOnInstallation, error) {

	addonInstallation, err := c.ocmClient.GetManagedKafkaAddon(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s addon: %w", clusterID, err)
	}

	// Addon needs to be installed if addonInstallation doesn't exist
	if addonInstallation.ID() == "" {
		// Install the Stimzi operator
		addonInstallation, err = c.ocmClient.CreateManagedKafkaAddon(clusterID)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster %s addon: %w", clusterID, err)
		}
	}

	return addonInstallation, nil
}
