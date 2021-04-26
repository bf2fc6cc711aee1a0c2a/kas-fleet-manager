package workers

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"k8s.io/apimachinery/pkg/runtime"

	ingressoperatorv1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/ingressoperator/v1"
	"github.com/pkg/errors"
	storagev1 "k8s.io/api/storage/v1"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"

	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	projectv1 "github.com/openshift/api/project/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AWSCloudProviderID              = "aws"
	observabilityNamespace          = "managed-application-services-observability"
	openshiftIngressNamespace       = "openshift-ingress-operator"
	observabilityDexCredentials     = "observatorium-dex-credentials"
	observabilityCatalogSourceImage = "quay.io/integreatly/observability-operator-index:v3.0.1"
	observabilityOperatorGroupName  = "observability-operator-group-name"
	observabilityCatalogSourceName  = "observability-operator-manifests"
	observabilitySubscriptionName   = "observability-operator"
	observabilityKafkaConfiguration = "kafka-observability-configuration"
	syncsetName                     = "ext-managedservice-cluster-mgr"
	imagePullSecretName             = "rhoas-image-pull-secret"
	strimziAddonNamespace           = "redhat-managed-kafka-operator"
	kasFleetshardAddonNamespace     = "redhat-kas-fleetshard-operator"
	openIDIdentityProviderName      = "Kafka_SRE"
	KafkaStorageClass               = "mk-storageclass"
	IngressLabelName                = "ingressType"
	IngressLabelValue               = "sharded"
)

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	id                         string
	workerType                 string
	isRunning                  bool
	ocmClient                  ocm.Client
	clusterService             services.ClusterService
	cloudProvidersService      services.CloudProvidersService
	timer                      *time.Timer
	imStop                     chan struct{} //a chan used only for cancellation
	syncTeardown               sync.WaitGroup
	reconciler                 Reconciler
	configService              services.ConfigService
	kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	osdIdpKeycloakService      services.KeycloakService
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService, cloudProvidersService services.CloudProvidersService, ocmClient ocm.Client,
	configService services.ConfigService, id string, agentOperatorAddon services.KasFleetshardOperatorAddon, osdIdpKeycloakService services.KeycloakService) *ClusterManager {
	return &ClusterManager{
		id:                         id,
		workerType:                 "cluster",
		ocmClient:                  ocmClient,
		clusterService:             clusterService,
		cloudProvidersService:      cloudProvidersService,
		configService:              configService,
		kasFleetshardOperatorAddon: agentOperatorAddon,
		osdIdpKeycloakService:      osdIdpKeycloakService,
	}
}

func (c *ClusterManager) GetStopChan() *chan struct{} {
	return &c.imStop
}

func (c *ClusterManager) GetSyncGroup() *sync.WaitGroup {
	return &c.syncTeardown
}

// GetID returns the ID that represents this worker
func (c *ClusterManager) GetID() string {
	return c.id
}

func (c *ClusterManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the cluster manager to reconcile osd clusters
func (c *ClusterManager) Start() {
	metrics.SetLeaderWorkerMetric(c.workerType, true)
	c.reconciler.Start(c)
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	c.reconciler.Stop(c)
	metrics.ResetMetricsForClusterManagers()
	metrics.SetLeaderWorkerMetric(c.workerType, false)
}

func (c *ClusterManager) IsRunning() bool {
	return c.isRunning
}

func (c *ClusterManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (c *ClusterManager) reconcile() []error {
	glog.Infoln("reconciling clusters")
	var errors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErr := c.setClusterStatusCountMetrics()
	if len(statusErr) > 0 {
		errors = append(errors, statusErr...)
	}

	if err := c.setKafkaPerClusterCountMetrics(); err != nil {
		errors = append(errors, err)
	}

	if err := c.reconcileClusterWithManualConfig(); err != nil {
		glog.Errorf("failed to reconcile clusters with config file: %s", err.Error())
		errors = append(errors, err)
	}

	// reconcile the status of existing clusters in a non-ready state
	glog.Infoln("reconcile cloud providers and regions")
	cloudProviders, err := c.cloudProvidersService.GetCloudProvidersWithRegions()
	if err != nil {
		glog.Error("Error retrieving cloud providers and regions", err)
		errors = append(errors, err)
	}

	for _, cloudProvider := range cloudProviders {
		// TODO add "|| provider.ID() == GcpCloudProviderID" to support GCP in the future
		if cloudProvider.ID == AWSCloudProviderID {
			cloudProvider.RegionList.Each(func(region *clustersmgmtv1.CloudRegion) bool {
				regionName := region.ID()
				glog.V(10).Infoln("Provider:", cloudProvider.ID, "=>", "Region:", regionName)
				return true
			})
		}
	}

	if err := c.reconcileClustersForRegions(); err != nil {
		glog.Errorf("failed to reconcile clusters by Region: %s", err.Error())
		errors = append(errors, err)
	}

	deprovisioningClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterDeprovisioning)
	if serviceErr != nil {
		glog.Errorf("failed to list of deprovisioning clusters: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("deprovisioning clusters count = %d", len(deprovisioningClusters))
	}

	for _, cluster := range deprovisioningClusters {
		glog.V(10).Infof("deprovision cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterDeprovisioning)
		if err := c.reconcileDeprovisioningCluster(cluster); err != nil {
			glog.Errorf("failed to reconcile deprovisioning cluster %s: %s", cluster.ID, err.Error())
			errors = append(errors, err)
		}
	}

	cleanupClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterCleanup)
	if serviceErr != nil {
		glog.Errorf("failed to list of cleaup clusters: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("cleanup clusters count = %d", len(cleanupClusters))
	}

	for _, cluster := range cleanupClusters {
		glog.V(10).Infof("cleanup cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterCleanup)
		if err := c.reconcileCleanupCluster(cluster); err != nil {
			glog.Errorf("failed to reconcile cleanup cluster %s: %s", cluster.ID, err.Error())
			errors = append(errors, err)
		}
	}

	acceptedClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterAccepted)
	if serviceErr != nil {
		glog.Errorf("failed to list accepted clusters: %s", serviceErr.Error())
		errors = append(errors, serviceErr)
	} else {
		glog.Infof("accepted clusters count = %d", len(acceptedClusters))
	}

	for _, cluster := range acceptedClusters {
		glog.V(10).Infof("accepted cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterAccepted)
		if err := c.reconcileAcceptedCluster(&cluster); err != nil {
			glog.Errorf("failed to reconcile accepted cluster %s: %s", cluster.ID, err.Error())
			errors = append(errors, err)
			continue
		}
		if err := c.clusterService.UpdateStatus(cluster, api.ClusterProvisioning); err != nil {
			glog.Errorf("failed to change cluster state to provisioning %s: %s", cluster.ID, err.Error())
			errors = append(errors, err)
		}
	}

	provisioningClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		glog.Errorf("failed to list pending clusters: %s", listErr.Error())
		errors = append(errors, listErr)
	} else {
		glog.Infof("provisioning clusters count = %d", len(provisioningClusters))
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for _, provisioningCluster := range provisioningClusters {
		glog.V(10).Infof("provisioning cluster ClusterID = %s", provisioningCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisioningCluster, api.ClusterProvisioning)
		_, err := c.reconcileClusterStatus(&provisioningCluster)
		if err != nil {
			glog.Errorf("failed to reconcile cluster %s status: %s", provisioningCluster.ClusterID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	/*
	 * Terraforming Provisioned Clusters
	 */
	provisionedClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		glog.Errorf("failed to list provisioned clusters: %s", listErr.Error())
		errors = append(errors, listErr)
	} else {
		glog.Infof("provisioned clusters count = %d", len(provisionedClusters))
	}

	// process each local provisioned cluster and apply necessary terraforming
	for _, provisionedCluster := range provisionedClusters {
		glog.V(10).Infof("provisioned cluster ClusterID = %s", provisionedCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterProvisioned)
		err := c.reconcileProvisionedCluster(provisionedCluster)
		if err != nil {
			glog.Errorf("failed to reconcile provisioned cluster %s: %s", provisionedCluster.ClusterID, err.Error())
			errors = append(errors, err)
			continue
		}
	}

	// Keep SyncSet up to date for clusters that are ready
	readyClusters, listErr := c.clusterService.ListByStatus(api.ClusterReady)
	if listErr != nil {
		glog.Errorf("failed to list ready clusters: %s", listErr.Error())
		errors = append(errors, listErr)
	} else {
		glog.Infof("ready clusters count = %d", len(readyClusters))
	}

	for _, readyCluster := range readyClusters {
		glog.V(10).Infof("ready cluster ClusterID = %s", readyCluster.ClusterID)
		emptyClusterReconciled := false
		var recErr error
		if c.configService.GetConfig().OSDClusterConfig.IsDataPlaneAutoScalingEnabled() {
			emptyClusterReconciled, recErr = c.reconcileEmptyCluster(readyCluster)
		}
		if !emptyClusterReconciled && recErr == nil {
			recErr = c.reconcileReadyCluster(readyCluster)
		}

		if recErr != nil {
			glog.Errorf("failed to reconcile ready cluster %s: %s", readyCluster.ClusterID, recErr.Error())
			errors = append(errors, recErr)
			continue
		}
	}

	return errors
}

func (c *ClusterManager) reconcileDeprovisioningCluster(cluster api.Cluster) error {
	if c.configService.GetConfig().OSDClusterConfig.IsDataPlaneAutoScalingEnabled() {
		siblingCluster, findClusterErr := c.clusterService.FindCluster(services.FindClusterCriteria{
			Region:   cluster.Region,
			Provider: cluster.CloudProvider,
			MultiAZ:  cluster.MultiAZ,
			Status:   api.ClusterReady,
		})

		if findClusterErr != nil {
			return findClusterErr
		}

		//if it is the only cluster left in that region, set it back to ready.
		if siblingCluster == nil {
			return c.clusterService.UpdateStatus(cluster, api.ClusterReady)
		}
	}

	code, deleteClusterErr := c.ocmClient.DeleteCluster(cluster.ClusterID)
	if deleteClusterErr != nil && code != http.StatusNotFound { // only log other errors by ignoring not found errors
		return deleteClusterErr
	}

	if code != http.StatusNotFound { // cluster delete request has been approved
		return nil
	}

	// cluster has been removed from cluster service. Mark it for cleanup
	glog.Infof("Cluster %s  has been removed from cluster service.", cluster.ClusterID)
	updateStatusErr := c.clusterService.UpdateStatus(cluster, api.ClusterCleanup)
	if updateStatusErr != nil {
		return errors.Wrapf(updateStatusErr, "Failed to update deprovisioning cluster %s status to 'cleanup'", cluster.ClusterID)
	}

	return nil
}

func (c *ClusterManager) reconcileCleanupCluster(cluster api.Cluster) error {
	glog.Infof("Removing Dataplane cluster %s IDP client", cluster.ClusterID)
	keycloakDeregistrationErr := c.osdIdpKeycloakService.DeRegisterClientInSSO(cluster.ID)
	if keycloakDeregistrationErr != nil {
		return errors.Wrapf(keycloakDeregistrationErr, "Failed to removed Dataplance cluster %s IDP client", cluster.ClusterID)
	}

	glog.Infof("Removing Dataplane cluster %s fleetshard service account", cluster.ClusterID)
	serviceAcountRemovalErr := c.kasFleetshardOperatorAddon.RemoveServiceAccount(cluster)
	if serviceAcountRemovalErr != nil {
		return errors.Wrapf(serviceAcountRemovalErr, "Failed to removed Dataplance cluster %s fleetshard service account", cluster.ClusterID)
	}

	glog.Infof("Soft deleting the Dataplane cluster %s from the database", cluster.ClusterID)
	deleteError := c.clusterService.DeleteByClusterID(cluster.ClusterID)
	if deleteError != nil {
		return errors.Wrapf(deleteError, "Failed to soft delete Dataplance cluster %s from the database", cluster.ClusterID)
	}
	return nil
}

func (c *ClusterManager) reconcileReadyCluster(cluster api.Cluster) error {
	err := c.reconcileClusterSyncSet(cluster)
	if err == nil {
		err = c.reconcileClusterIdentityProvider(cluster)
	}
	if err == nil && c.kasFleetshardOperatorAddon != nil {
		if e := c.kasFleetshardOperatorAddon.ReconcileParameters(cluster); e != nil {
			if e.IsBadRequest() {
				glog.Infof("kas-fleetshard operator is not found on cluster %s", cluster.ClusterID)
			} else {
				err = e
			}
		}
	}

	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile ready cluster %s: %s", cluster.ClusterID, err.Error())
	}
	return nil
}

// reconcileEmptyCluster checks wether a cluster is empty and mark it for deletion
func (c *ClusterManager) reconcileEmptyCluster(cluster api.Cluster) (bool, error) {
	glog.V(10).Infof("check if cluster is empty, ClusterID = %s", cluster.ClusterID)
	clusterFromDb, err := c.clusterService.FindNonEmptyClusterById(cluster.ClusterID)
	if err != nil {
		return false, err
	}
	if clusterFromDb != nil {
		glog.V(10).Infof("cluster is not empty, ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	clustersByRegionAndCloudProvider, findSiblingClusterErr := c.clusterService.ListGroupByProviderAndRegion(
		[]string{cluster.CloudProvider},
		[]string{cluster.Region},
		[]string{api.ClusterReady.String()})

	if findSiblingClusterErr != nil || len(clustersByRegionAndCloudProvider) == 0 {
		return false, findSiblingClusterErr
	}

	siblingClusterCount := clustersByRegionAndCloudProvider[0]
	if siblingClusterCount.Count <= 1 { // sibling cluster not found
		glog.V(10).Infof("no valid sibling found for cluster ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	updateStatusErr := c.clusterService.UpdateStatus(cluster, api.ClusterDeprovisioning)
	return updateStatusErr == nil, updateStatusErr
}

func (c *ClusterManager) reconcileProvisionedCluster(cluster api.Cluster) error {
	if err := c.reconcileClusterIdentityProvider(cluster); err != nil {
		return err
	}

	// SyncSet creation step
	syncSetErr := c.reconcileClusterSyncSet(cluster) //OSD cluster itself
	if syncSetErr != nil {
		return errors.WithMessagef(syncSetErr, "failed to reconcile cluster %s SyncSet: %s", cluster.ClusterID, syncSetErr.Error())
	}

	// Addon installation step
	// TODO this is currently the responsible of setting the status of the cluster
	// and it is setting it to a different value depending on the addon being
	// installed. The logic to set the status of the cluster should probably done
	// independently of the installation of the addon, and it should use the
	// result of the addon/s reconciliation to set the status of the cluster
	addOnErr := c.reconcileAddonOperator(cluster)
	if addOnErr != nil {
		return errors.WithMessagef(addOnErr, "failed to reconcile cluster %s addon operator: %s", cluster.ClusterID, addOnErr.Error())
	}

	return nil
}

func (c *ClusterManager) reconcileClusterSyncSet(cluster api.Cluster) error {
	clusterDNS, dnsErr := c.clusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster %s: %s", cluster.ClusterID, dnsErr.Error())
	}

	existingSyncset, err := c.ocmClient.GetSyncSet(cluster.ClusterID, syncsetName)
	syncSetFound := true
	if err != nil {
		svcErr := svcErrors.ToServiceError(err)
		if !svcErr.Is404() {
			return err
		}
		syncSetFound = false
	}

	clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)
	if !syncSetFound {
		glog.V(10).Infof("SyncSet for cluster %s not found. Creating it...", cluster.ClusterID)
		_, syncsetErr := c.createSyncSet(cluster.ClusterID, clusterDNS)
		if syncsetErr != nil {
			return errors.WithMessagef(syncsetErr, "failed to create syncset on cluster %s: %s", cluster.ClusterID, syncsetErr.Error())
		}
	} else {
		glog.V(10).Infof("SyncSet for cluster %s already created", cluster.ClusterID)
		_, syncsetErr := c.updateSyncSet(cluster.ClusterID, clusterDNS, existingSyncset)
		if syncsetErr != nil {
			return errors.WithMessagef(syncsetErr, "failed to update syncset on cluster %s: %s", cluster.ClusterID, syncsetErr.Error())
		}
	}

	return nil
}

func (c *ClusterManager) reconcileAcceptedCluster(cluster *api.Cluster) error {
	reconciledCluster, err := c.clusterService.Create(cluster)
	if err != nil {
		return fmt.Errorf("failed to create cluster for request %s: %w", cluster.ID, err)
	}

	// as all fields on OCM structs are internal we cannot perform a standard json marshal as all fields will be empty,
	// instead we need to use the OCM type-specific marshal functions when converting a struct to json
	// declare a buffer to store the resulting json and invoke the OCM type-specific marshal function to populate the
	// buffer with a json string containing the internal cluster values.
	indentedCluster := new(bytes.Buffer)
	if err := clustersmgmtv1.MarshalCluster(reconciledCluster, indentedCluster); err != nil {
		return fmt.Errorf("unable to marshal cluster: %s", err.Error())
	}

	glog.V(10).Infof("%s", indentedCluster.String())
	return nil
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
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterFailed)
	}
	// if cluster is neither ready nor in an error state, assume it's pending
	if needsUpdate {
		if err = c.clusterService.UpdateStatus(*cluster, cluster.Status); err != nil {
			return nil, fmt.Errorf("failed to update local cluster %s status: %w", cluster.ClusterID, err)
		}
	}
	return cluster, nil
}

func (c *ClusterManager) reconcileAddonOperator(provisionedCluster api.Cluster) error {
	isStrimziReady, err := c.reconcileStrimziOperator(provisionedCluster)
	if err != nil {
		return err
	}
	isFleetShardReady := true
	if c.configService.GetConfig().Kafka.EnableManagedKafkaCR || c.configService.GetConfig().Kafka.EnableKasFleetshardSync {
		if c.kasFleetshardOperatorAddon != nil {
			glog.Infof("Provisioning kas-fleetshard-operator as it is enabled")
			if ready, errs := c.kasFleetshardOperatorAddon.Provision(provisionedCluster); errs != nil {
				return errs
			} else {
				isFleetShardReady = ready
			}
		}
	}
	if c.configService.GetConfig().Kafka.EnableKasFleetshardSync {
		// if kas-fleetshard sync is enabled, the cluster status will be reported by the kas-fleetshard
		glog.V(5).Infof("Set cluster status to %s for cluster %s", api.ClusterWaitingForKasFleetShardOperator, provisionedCluster.ClusterID)
		if err := c.clusterService.UpdateStatus(provisionedCluster, api.ClusterWaitingForKasFleetShardOperator); err != nil {
			return errors.WithMessagef(err, "failed to update local cluster %s status: %s", provisionedCluster.ClusterID, err.Error())
		}
		metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterWaitingForKasFleetShardOperator)
	} else if isStrimziReady && isFleetShardReady {
		status := api.ClusterReady
		glog.V(5).Infof("Set cluster status to %s for cluster %s", status, provisionedCluster.ClusterID)
		if err := c.clusterService.UpdateStatus(provisionedCluster, status); err != nil {
			return errors.WithMessagef(err, "failed to update local cluster %s status: %s", provisionedCluster.ClusterID, err.Error())
		}

		// add entry for cluster creation metric
		metrics.UpdateClusterCreationDurationMetric(metrics.JobTypeClusterCreate, time.Since(provisionedCluster.CreatedAt))
		metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterReady)
	}
	return nil
}

// reconcileStrimziOperator installs the Strimzi operator on a provisioned clusters
func (c *ClusterManager) reconcileStrimziOperator(provisionedCluster api.Cluster) (bool, error) {
	clusterId := provisionedCluster.ClusterID
	addonInstallation, err := c.ocmClient.GetAddon(clusterId, api.ManagedKafkaAddonID)
	if err != nil {
		return false, errors.WithMessagef(err, "failed to get cluster %s addon: %s", clusterId, err.Error())
	}

	// Addon needs to be installed if addonInstallation doesn't exist
	if addonInstallation.ID() == "" {
		// Install the Stimzi operator
		addonInstallation, err = c.ocmClient.CreateAddon(clusterId, api.ManagedKafkaAddonID)
		if err != nil {
			return false, errors.WithMessagef(err, "failed to create cluster %s addon: %s", clusterId, err.Error())
		}
	}

	// The cluster is ready when the state reports ready
	if addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		return true, nil
	}

	glog.V(5).Infof("%s addon on cluster %s is not ready yet. State: %s", api.ManagedKafkaAddonID, provisionedCluster.ClusterID, string(addonInstallation.State()))
	return false, nil
}

// reconcileClusterWithConfig reconciles clusters with the config file
// A new clusters will be registered if it is not yet in the database
// A cluster will be deprovisioned if it is in database but not in config file
func (c *ClusterManager) reconcileClusterWithManualConfig() error {
	if !c.configService.GetConfig().OSDClusterConfig.IsDataPlaneManualScalingEnabled() {
		glog.Infoln("manual cluster configuration reconciliation is skipped as it is disabled")
		return nil
	}

	glog.Infoln("reconciling manual cluster configurations")
	clusters, err := c.clusterService.ListAllClusterIds()
	if err != nil {
		return fmt.Errorf("failed to retrieve cluster ids from clusters: %s", err.Error())
	}
	clusterIdsMap := make(map[string]api.Cluster)
	for _, v := range clusters {
		clusterIdsMap[v.ClusterID] = v
	}

	//Create all missing clusters
	for _, p := range c.configService.GetConfig().OSDClusterConfig.ClusterConfig.MissingClusters(clusterIdsMap) {
		clusterRequest := api.Cluster{
			CloudProvider: p.CloudProvider,
			Region:        p.Region,
			MultiAZ:       p.MultiAZ,
			ClusterID:     p.ClusterId,
			Status:        api.ClusterProvisioning,
		}
		if err := c.clusterService.RegisterClusterJob(&clusterRequest); err != nil {
			glog.Errorf("Failed to register new cluster with config file: %s, %s ", p.ClusterId, err.Error())
		} else {
			glog.Infof("Registered a new cluster with config file: %s ", p.ClusterId)
		}
	}

	// Remove all clusters that are not in the config file
	excessClusterIds := c.configService.GetConfig().OSDClusterConfig.ClusterConfig.ExcessClusters(clusterIdsMap)
	if len(excessClusterIds) == 0 {
		return nil
	}

	kafkaInstanceCount, err := c.clusterService.FindKafkaInstanceCount(excessClusterIds)
	if err != nil {
		glog.Errorf("Failed to find kafka count a cluster: %s, %s ", excessClusterIds, err.Error())
		return err
	}

	var idsOfClustersToDeprovision []string
	for _, c := range kafkaInstanceCount {
		if c.Count > 0 {
			glog.Infof("Excess cluster %s is not going to be deleted because it has %d kafka.", c.Clusterid, c.Count)
		} else {
			glog.Infof("Excess cluster is going to be deleted %s", c.Clusterid)
			idsOfClustersToDeprovision = append(idsOfClustersToDeprovision, c.Clusterid)
		}
	}

	if len(idsOfClustersToDeprovision) == 0 {
		return nil
	}

	if err := c.clusterService.UpdateMultiClusterStatus(idsOfClustersToDeprovision, api.ClusterDeprovisioning); err != nil {
		glog.Errorf("Failed to deprovisioning a cluster: %s, %s ", idsOfClustersToDeprovision, err.Error())
		return err
	} else {
		glog.Infof("Deprovisioning clusters: not found in config file: %s ", idsOfClustersToDeprovision)
	}

	return nil
}

// reconcileClustersForRegions creates an OSD cluster for each region where no cluster exists
func (c *ClusterManager) reconcileClustersForRegions() error {
	if !c.configService.GetConfig().OSDClusterConfig.IsDataPlaneAutoScalingEnabled() {
		return nil
	}

	var providers []string
	var regions []string
	status := api.StatusForValidCluster
	//gather the supported providers and regions
	providerList := c.configService.GetSupportedProviders()
	for _, v := range providerList {
		providers = append(providers, v.Name)
		for _, r := range v.Regions {
			regions = append(regions, r.Name)
		}
	}

	//get a list of clusters in Map group by their provider and region
	grpResult, err := c.clusterService.ListGroupByProviderAndRegion(providers, regions, status)
	if err != nil {
		return fmt.Errorf("failed to find cluster with criteria: %s", err.Error())
	}

	grpResultMap := make(map[string]*services.ResGroupCPRegion)
	for _, v := range grpResult {
		grpResultMap[v.Provider+"."+v.Region] = v
	}

	//create all the missing clusters in the supported provider and regions.
	for _, p := range providerList {
		for _, v := range p.Regions {
			if _, exist := grpResultMap[p.Name+"."+v.Name]; !exist {
				clusterRequest := api.Cluster{
					CloudProvider: p.Name,
					Region:        v.Name,
					MultiAZ:       true,
					Status:        api.ClusterAccepted,
				}
				if err := c.clusterService.RegisterClusterJob(&clusterRequest); err != nil {
					glog.Errorf("Failed to auto-create cluster request in %s, region: %s %s", p.Name, v.Name, err.Error())
				} else {
					glog.Infof("Auto-created cluster request in %s, region: %s, Id: %s ", p.Name, v.Name, clusterRequest.ID)
				}
			} //
		} //region
	} //provider
	return nil
}

func (c *ClusterManager) buildSyncSet(ingressDNS string, withId bool) (*clustersmgmtv1.Syncset, error) {
	r := []interface{}{
		c.buildStorageClass(),
		c.buildIngressController(ingressDNS),
		c.buildObservabilityNamespaceResource(),
		c.buildObservabilityDexSecretResource(),
		c.buildObservabilityCatalogSourceResource(),
		c.buildObservabilityOperatorGroupResource(),
		c.buildObservabilitySubscriptionResource(),
	}
	// If kas-fleetshard sync is enabled, the external config for the observability will be delivered to the kas-fleetshard directly
	if !c.configService.GetConfig().Kafka.EnableKasFleetshardSync {
		r = append(r, c.buildObservabilityExternalConfigResource())
	}
	if s := c.buildImagePullSecret(strimziAddonNamespace); s != nil {
		r = append(r, s)
	}
	if s := c.buildImagePullSecret(kasFleetshardAddonNamespace); s != nil {
		r = append(r, s)
	}
	b := clustersmgmtv1.NewSyncset().
		Resources(r...)
	if withId {
		b = b.ID(syncsetName)
	}
	return b.Build()
}

// createSyncSet creates the syncset during cluster terraforming
func (c *ClusterManager) createSyncSet(clusterID string, ingressDNS string) (*clustersmgmtv1.Syncset, error) {
	// terraforming phase
	syncset, sysnsetBuilderErr := c.buildSyncSet(ingressDNS, true)

	if sysnsetBuilderErr != nil {
		return nil, fmt.Errorf("failed to create cluster terraforming sysncset: %s", sysnsetBuilderErr.Error())
	}

	return c.ocmClient.CreateSyncSet(clusterID, syncset)
}

func (c *ClusterManager) updateSyncSet(clusterID string, ingressDNS string, existingSyncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	syncset, sysnsetBuilderErr := c.buildSyncSet(ingressDNS, false)
	if sysnsetBuilderErr != nil {
		return nil, fmt.Errorf("failed to build cluster sysncset: %s", sysnsetBuilderErr.Error())
	}
	if syncsetResourcesChanged(existingSyncset, syncset) {
		glog.V(5).Infof("SyncSet for cluster %s is changed, will update", clusterID)
		return c.ocmClient.UpdateSyncSet(clusterID, syncsetName, syncset)
	}
	glog.V(10).Infof("SyncSet for cluster %s is not changed, no update needed", clusterID)
	return syncset, nil
}

func syncsetResourcesChanged(existing *clustersmgmtv1.Syncset, new *clustersmgmtv1.Syncset) bool {
	if len(existing.Resources()) != len(new.Resources()) {
		return true
	}
	// Here we will convert values in the Resources slice to the same type, and then compare the values.
	// This is needed because when you use ocm.GetSyncset(), the Resources in the returned object only contains a slice of map[string]interface{} objects, because it can't convert them to concrete typed objects.
	// So the compare if there changes, we need to make sure they are the same type first.
	// If the type conversion doesn't work, or the converted values doesn't match, then they are not equal.
	// This assumes that the order of objects in the Resources slice are the same in the exiting and new Syncset (which is the case as the OCM API returns the syncset resources in the same order as they posted)
	for i, r := range new.Resources() {
		obj := reflect.New(reflect.TypeOf(r).Elem()).Interface()
		// Here we convert the unstructured type to the concrete type, as there is a bug in OperatorGroup type to convert it to the unstructured type
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(existing.Resources()[i].(map[string]interface{}), obj)
		// if we can't do the type conversion, it likely means the resource has changed
		if err != nil {
			return true
		}
		if !reflect.DeepEqual(obj, r) {
			return true
		}
	}

	return false
}

func (c *ClusterManager) buildObservabilityNamespaceResource() *projectv1.Project {
	return &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: projectv1.SchemeGroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: observabilityNamespace,
		},
	}
}

func (c *ClusterManager) buildObservabilityDexSecretResource() *k8sCoreV1.Secret {
	observabilityConfig := c.configService.GetObservabilityConfiguration()
	stringDataMap := map[string]string{
		"password": observabilityConfig.DexPassword,
		"secret":   observabilityConfig.DexSecret,
		"username": observabilityConfig.DexUsername,
	}

	return &k8sCoreV1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityDexCredentials,
			Namespace: observabilityNamespace,
		},
		Type:       k8sCoreV1.SecretTypeOpaque,
		StringData: stringDataMap,
	}
}

func (c *ClusterManager) buildObservabilityCatalogSourceResource() *v1alpha1.CatalogSource {
	return &v1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "CatalogSource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityCatalogSourceName,
			Namespace: observabilityNamespace,
		},
		Spec: v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
			Image:      observabilityCatalogSourceImage,
		},
	}
}

func (c *ClusterManager) buildObservabilityOperatorGroupResource() *v1alpha2.OperatorGroup {
	return &v1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha2.SchemeGroupVersion.String(),
			Kind:       "OperatorGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityOperatorGroupName,
			Namespace: observabilityNamespace,
		},
		Spec: v1alpha2.OperatorGroupSpec{
			TargetNamespaces: []string{observabilityNamespace},
		},
	}
}

func (c *ClusterManager) buildObservabilitySubscriptionResource() *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilitySubscriptionName,
			Namespace: observabilityNamespace,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          observabilityCatalogSourceName,
			Channel:                "alpha",
			CatalogSourceNamespace: observabilityNamespace,
			StartingCSV:            "observability-operator.v3.0.1",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
			Package:                observabilitySubscriptionName,
		},
	}
}

func (c *ClusterManager) buildIngressController(ingressDNS string) *ingressoperatorv1.IngressController {
	r := int32(c.configService.GetConfig().OSDClusterConfig.IngressControllerReplicas)
	return &ingressoperatorv1.IngressController{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.openshift.io/v1",
			Kind:       "IngressController",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sharded-nlb",
			Namespace: openshiftIngressNamespace,
		},
		Spec: ingressoperatorv1.IngressControllerSpec{
			Domain: ingressDNS,
			RouteSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					IngressLabelName: IngressLabelValue,
				},
			},
			EndpointPublishingStrategy: &ingressoperatorv1.EndpointPublishingStrategy{
				LoadBalancer: &ingressoperatorv1.LoadBalancerStrategy{
					ProviderParameters: &ingressoperatorv1.ProviderLoadBalancerParameters{
						AWS: &ingressoperatorv1.AWSLoadBalancerParameters{
							Type: ingressoperatorv1.AWSNetworkLoadBalancer,
						},
						Type: ingressoperatorv1.AWSLoadBalancerProvider,
					},
					Scope: ingressoperatorv1.ExternalLoadBalancer,
				},
				Type: ingressoperatorv1.LoadBalancerServiceStrategyType,
			},
			Replicas: &r,
			NodePlacement: &ingressoperatorv1.NodePlacement{
				NodeSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
				},
			},
		},
	}
}

func (c *ClusterManager) buildStorageClass() *storagev1.StorageClass {
	reclaimDelete := k8sCoreV1.PersistentVolumeReclaimDelete
	expansion := true
	consumer := storagev1.VolumeBindingWaitForFirstConsumer

	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: KafkaStorageClass,
		},
		Parameters: map[string]string{
			"encrypted": "false",
			"type":      "gp2",
		},
		Provisioner:          "kubernetes.io/aws-ebs",
		ReclaimPolicy:        &reclaimDelete,
		AllowVolumeExpansion: &expansion,
		VolumeBindingMode:    &consumer,
	}
}

func (c *ClusterManager) buildObservabilityExternalConfigResource() *k8sCoreV1.ConfigMap {
	observabilityConfig := c.configService.GetObservabilityConfiguration()
	return &k8sCoreV1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityKafkaConfiguration,
			Namespace: observabilityNamespace,
			Labels: map[string]string{
				"configures": "observability-operator",
			},
		},
		Data: map[string]string{
			"access_token": observabilityConfig.ObservabilityConfigAccessToken,
			"channel":      observabilityConfig.ObservabilityConfigChannel,
			"repository":   observabilityConfig.ObservabilityConfigRepo,
			"tag":          observabilityConfig.ObservabilityConfigTag,
		},
	}
}

func (c *ClusterManager) buildImagePullSecret(namespace string) *k8sCoreV1.Secret {
	content := c.configService.GetConfig().OSDClusterConfig.ImagePullDockerConfigContent
	if strings.TrimSpace(content) == "" {
		return nil
	}

	dataMap := map[string][]byte{
		k8sCoreV1.DockerConfigKey: []byte(content),
	}

	return &k8sCoreV1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePullSecretName,
			Namespace: namespace,
		},
		Type: k8sCoreV1.SecretTypeDockercfg,
		Data: dataMap,
	}
}

func (c *ClusterManager) reconcileClusterIdentityProvider(cluster api.Cluster) error {
	if cluster.IdentityProviderID != "" {
		return nil
	}
	// identity provider not yet created, let's create a new one
	glog.Infof("Setting up the identity provider for cluster %s", cluster.ClusterID)
	clusterDNS, dnsErr := c.clusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster identity provider %s: %s", cluster.ClusterID, dnsErr.Error())
	}

	callbackUri := fmt.Sprintf("https://oauth-openshift.%s/oauth2callback/%s", clusterDNS, openIDIdentityProviderName)
	clientSecret, ssoErr := c.osdIdpKeycloakService.RegisterOSDClusterClientInSSO(cluster.ID, callbackUri)
	if ssoErr != nil {
		return errors.WithMessagef(ssoErr, "failed to reconcile cluster identity provider %s: %s", cluster.ClusterID, ssoErr.Error())
	}

	identityProvider, buildErr := c.buildIdentityProvider(cluster, clientSecret, true)
	if buildErr != nil {
		return buildErr
	}
	createdIdentityProvider, createIdentityProviderErr := c.ocmClient.CreateIdentityProvider(cluster.ClusterID, identityProvider)
	if createIdentityProviderErr != nil {
		return errors.WithMessagef(createIdentityProviderErr, "failed to create cluster identity provider %s: %s", cluster.ClusterID, createIdentityProviderErr.Error())
	}
	addIdpErr := c.clusterService.AddIdentityProviderID(cluster.ID, createdIdentityProvider.ID())
	if addIdpErr != nil {
		return errors.WithMessagef(addIdpErr, "failed to update cluster identity provider in database %s: %s", cluster.ClusterID, addIdpErr.Error())
	}
	glog.Infof("Identity provider is set up for cluster %s", cluster.ClusterID)
	return nil
}

func (c *ClusterManager) buildIdentityProvider(cluster api.Cluster, clientSecret string, withName bool) (*clustersmgmtv1.IdentityProvider, error) {
	openIdentityBuilder := clustersmgmtv1.NewOpenIDIdentityProvider().
		ClientID(cluster.ID).
		ClientSecret(clientSecret).
		Claims(clustersmgmtv1.NewOpenIDClaims().
			Email("email").
			PreferredUsername("preferred_username").
			Name("last_name", "preferred_username")).
		Issuer(c.osdIdpKeycloakService.GetRealmConfig().ValidIssuerURI)

	identityProviderBuilder := clustersmgmtv1.NewIdentityProvider().
		Type("OpenIDIdentityProvider").
		MappingMethod(clustersmgmtv1.IdentityProviderMappingMethodClaim).
		OpenID(openIdentityBuilder)

	if withName {
		identityProviderBuilder.Name(openIDIdentityProviderName)
	}

	identityProvider, idpBuildErr := identityProviderBuilder.Build()
	if idpBuildErr != nil {
		return nil, errors.WithMessagef(idpBuildErr, "failed to reconcile cluster identity provider %s: %s", cluster.ClusterID, idpBuildErr.Error())
	}

	return identityProvider, nil
}

func (c *ClusterManager) setClusterStatusCountMetrics() []error {
	status := []api.ClusterStatus{
		api.ClusterAccepted,
		api.ClusterProvisioning,
		api.ClusterProvisioned,
		api.ClusterWaitingForKasFleetShardOperator,
		api.ClusterReady,
		api.ClusterComputeNodeScalingUp,
		api.ClusterFull,
		api.ClusterFailed,
		api.ClusterDeprovisioning,
	}

	var errors []error
	if counters, err := c.clusterService.CountByStatus(status); err != nil {
		glog.Errorf("failed to count clusters by status: %s", err.Error())
		errors = append(errors, err)
	} else {
		for _, c := range counters {
			metrics.UpdateClusterStatusCountMetric(c.Status, c.Count)
		}
	}
	return errors
}

func (c *ClusterManager) setKafkaPerClusterCountMetrics() error {
	if counters, err := c.clusterService.FindKafkaInstanceCount([]string{}); err != nil {
		return err
	} else {
		for _, c := range counters {
			metrics.UpdateKafkaPerClusterCountMetric(c.Clusterid, c.Count)
		}
	}
	return nil
}
