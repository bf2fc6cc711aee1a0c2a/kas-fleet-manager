package workers

import (
	"fmt"
	ingressoperatorv1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/ingressoperator/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/pkg/errors"
	storagev1 "k8s.io/api/storage/v1"
	"strings"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"

	projectv1 "github.com/openshift/api/project/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	userv1 "github.com/openshift/api/user/v1"
)

const (
	observabilityNamespace          = "managed-application-services-observability"
	openshiftIngressNamespace       = "openshift-ingress-operator"
	observabilityDexCredentials     = "observatorium-dex-credentials"
	observabilityCatalogSourceImage = "quay.io/integreatly/observability-operator-index:v3.0.1"
	observabilityOperatorGroupName  = "observability-operator-group-name"
	observabilityCatalogSourceName  = "observability-operator-manifests"
	observabilitySubscriptionName   = "observability-operator"
	syncsetName                     = "ext-managedservice-cluster-mgr"
	imagePullSecretName             = "rhoas-image-pull-secret"
	strimziAddonNamespace           = "redhat-managed-kafka-operator"
	kasFleetshardAddonNamespace     = "redhat-kas-fleetshard-operator"
	openIDIdentityProviderName      = "Kafka_SRE"
	readOnlyGroupName               = "mk-readonly-access"
	mkReadOnlyRoleBindingName       = "mk-dedicated-readers"
	dedicatedReadersRoleBindingName = "dedicated-readers"
	KafkaStorageClass               = "mk-storageclass"
	IngressLabelName                = "ingressType"
	IngressLabelValue               = "sharded"
)

var clusterMetricsStatuses = []api.ClusterStatus{
	api.ClusterAccepted,
	api.ClusterProvisioning,
	api.ClusterProvisioned,
	api.ClusterCleanup,
	api.ClusterWaitingForKasFleetShardOperator,
	api.ClusterReady,
	api.ClusterComputeNodeScalingUp,
	api.ClusterFull,
	api.ClusterFailed,
	api.ClusterDeprovisioning,
}

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	id                         string
	workerType                 string
	isRunning                  bool
	clusterService             services.ClusterService
	cloudProvidersService      services.CloudProvidersService
	imStop                     chan struct{} //a chan used only for cancellation
	syncTeardown               sync.WaitGroup
	reconciler                 Reconciler
	configService              services.ConfigService
	kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	osdIdpKeycloakService      services.KeycloakService
}

type processor func() []error

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService, cloudProvidersService services.CloudProvidersService, configService services.ConfigService, id string, agentOperatorAddon services.KasFleetshardOperatorAddon, osdIdpKeycloakService services.KeycloakService) *ClusterManager {
	return &ClusterManager{
		id:                         id,
		workerType:                 "cluster",
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

func (c *ClusterManager) Reconcile() []error {
	glog.Infoln("reconciling clusters")
	var encounteredErrors []error

	processors := []processor{
		c.processMetrics,
		c.reconcileClusterWithManualConfig,
		c.reconcileClustersForRegions,
		c.processDeprovisioningClusters,
		c.processCleanupClusters,
		c.processAcceptedClusters,
		c.processProvisioningClusters,
		c.processProvisionedClusters,
		c.processReadyClusters,
	}

	for _, p := range processors {
		if errs := p(); len(errs) > 0 {
			encounteredErrors = append(encounteredErrors, errs...)
		}
	}
	return encounteredErrors
}

func (c *ClusterManager) processMetrics() []error {
	if err := c.setClusterStatusCountMetrics(); err != nil {
		return []error{errors.Wrapf(err, "failed to set cluster status count metrics")}
	}

	if err := c.setKafkaPerClusterCountMetrics(); err != nil {
		return []error{errors.Wrapf(err, "failed to set kafka per cluster count metrics")}
	}
	return []error{}
}

func (c *ClusterManager) processDeprovisioningClusters() []error {
	var errs []error
	deprovisioningClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterDeprovisioning)
	if serviceErr != nil {
		errs = append(errs, serviceErr)
		return errs
	} else {
		glog.Infof("deprovisioning clusters count = %d", len(deprovisioningClusters))
	}

	for _, cluster := range deprovisioningClusters {
		glog.V(10).Infof("deprovision cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterDeprovisioning)
		if err := c.reconcileDeprovisioningCluster(&cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile deprovisioning cluster %s", cluster.ID))
		}
	}
	return errs
}

func (c *ClusterManager) processCleanupClusters() []error {
	var errs []error
	cleanupClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterCleanup)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "failed to list of cleaup clusters"))
		return errs
	} else {
		glog.Infof("cleanup clusters count = %d", len(cleanupClusters))
	}

	for _, cluster := range cleanupClusters {
		glog.V(10).Infof("cleanup cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterCleanup)
		if err := c.reconcileCleanupCluster(cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile cleanup cluster %s", cluster.ID))
		}
	}
	return errs
}

func (c *ClusterManager) processAcceptedClusters() []error {
	var errs []error
	acceptedClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterAccepted)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "failed to list accepted clusters"))
		return errs
	} else {
		glog.Infof("accepted clusters count = %d", len(acceptedClusters))
	}

	for _, cluster := range acceptedClusters {
		glog.V(10).Infof("accepted cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterAccepted)
		if err := c.reconcileAcceptedCluster(&cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile accepted cluster %s", cluster.ID))
			continue
		}
	}
	return errs
}

func (c *ClusterManager) processProvisioningClusters() []error {
	var errs []error
	provisioningClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list pending clusters"))
		return errs
	} else {
		glog.Infof("provisioning clusters count = %d", len(provisioningClusters))
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for _, provisioningCluster := range provisioningClusters {
		glog.V(10).Infof("provisioning cluster ClusterID = %s", provisioningCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisioningCluster, api.ClusterProvisioning)
		_, err := c.reconcileClusterStatus(&provisioningCluster)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile cluster %s status", provisioningCluster.ClusterID))
			continue
		}
	}
	return errs
}

func (c *ClusterManager) processProvisionedClusters() []error {
	var errs []error
	/*
	 * Terraforming Provisioned Clusters
	 */
	provisionedClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list provisioned clusters"))
		return errs
	} else {
		glog.Infof("provisioned clusters count = %d", len(provisionedClusters))
	}

	// process each local provisioned cluster and apply necessary terraforming
	for _, provisionedCluster := range provisionedClusters {
		glog.V(10).Infof("provisioned cluster ClusterID = %s", provisionedCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterProvisioned)
		err := c.reconcileProvisionedCluster(provisionedCluster)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile provisioned cluster %s", provisionedCluster.ClusterID))
			continue
		}
	}

	return errs
}

func (c *ClusterManager) processReadyClusters() []error {
	var errs []error
	// Keep SyncSet up to date for clusters that are ready
	readyClusters, listErr := c.clusterService.ListByStatus(api.ClusterReady)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list ready clusters"))
		return errs
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
			errs = append(errs, errors.Wrapf(recErr, "failed to reconcile ready cluster %s", readyCluster.ClusterID))
			continue
		}
	}
	return errs
}

func (c *ClusterManager) reconcileDeprovisioningCluster(cluster *api.Cluster) error {
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
			return c.clusterService.UpdateStatus(*cluster, api.ClusterReady)
		}
	}

	deleted, deleteClusterErr := c.clusterService.RemoveClusterFromProvider(cluster)
	if deleteClusterErr != nil {
		return deleteClusterErr
	}

	if !deleted {
		return nil
	}

	// cluster has been removed from cluster service. Mark it for cleanup
	glog.Infof("Cluster %s  has been removed from cluster service.", cluster.ClusterID)
	updateStatusErr := c.clusterService.UpdateStatus(*cluster, api.ClusterCleanup)
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
	if !c.configService.GetConfig().OSDClusterConfig.IsReadyDataPlaneClustersReconcileEnabled() {
		glog.Infof("Reconcile of dataplane ready clusters is disabled. Skipped reconcile of ready ClusterID '%s'", cluster.ClusterID)
		return nil
	}

	var err error
	// resources update if needed
	if err := c.reconcileClusterResources(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile cluster resources %s ", cluster.ClusterID)
	}

	err = c.reconcileClusterIdentityProvider(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	err = c.reconcileClusterDNS(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	if c.kasFleetshardOperatorAddon != nil {
		if err := c.kasFleetshardOperatorAddon.ReconcileParameters(cluster); err != nil {
			if err.IsBadRequest() {
				glog.Infof("kas-fleetshard operator is not found on cluster %s", cluster.ClusterID)
			} else {
				return errors.WithMessagef(err, "failed to reconcile ready cluster %s: %s", cluster.ClusterID, err.Error())
			}
		}
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

	if err := c.reconcileClusterDNS(cluster); err != nil {
		return err
	}

	// SyncSet creation step
	syncSetErr := c.reconcileClusterResources(cluster) //OSD cluster itself
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

func (c *ClusterManager) reconcileClusterDNS(cluster api.Cluster) error {
	// Return if the clusterDNS is already set
	if cluster.ClusterDNS != "" {
		return nil
	}

	_, dnsErr := c.clusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster %s: GetClusterDNS %s", cluster.ClusterID, dnsErr.Error())
	}

	return nil
}

func (c *ClusterManager) reconcileClusterResources(cluster api.Cluster) error {
	clusterDNS, dnsErr := c.clusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.Wrapf(dnsErr, "failed to reconcile cluster %s: %s", cluster.ClusterID, dnsErr.Error())
	}
	clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)
	resourceSet := c.buildResourceSet(clusterDNS)
	if err := c.clusterService.ApplyResources(&cluster, resourceSet); err != nil {
		return errors.Wrapf(err, "failed to apply resources for cluster %s", cluster.ClusterID)
	}

	return nil
}

func (c *ClusterManager) reconcileAcceptedCluster(cluster *api.Cluster) error {
	_, err := c.clusterService.Create(cluster)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster for request %s", cluster.ID)
	}

	return nil
}

// reconcileClusterStatus updates the provided clusters stored status to reflect it's current state
func (c *ClusterManager) reconcileClusterStatus(cluster *api.Cluster) (*api.Cluster, error) {
	updatedCluster, err := c.clusterService.CheckClusterStatus(cluster)
	if err != nil {
		return nil, err
	}
	if updatedCluster.Status == api.ClusterFailed {
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterFailed)
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
	}
	return updatedCluster, nil
}

func (c *ClusterManager) reconcileAddonOperator(provisionedCluster api.Cluster) error {
	if _, err := c.reconcileStrimziOperator(provisionedCluster); err != nil {
		return err
	}
	glog.Infof("Provisioning kas-fleetshard-operator as it is enabled")
	if _, errs := c.kasFleetshardOperatorAddon.Provision(provisionedCluster); errs != nil {
		return errs
	}
	glog.V(5).Infof("Set cluster status to %s for cluster %s", api.ClusterWaitingForKasFleetShardOperator, provisionedCluster.ClusterID)
	if err := c.clusterService.UpdateStatus(provisionedCluster, api.ClusterWaitingForKasFleetShardOperator); err != nil {
		return errors.Wrapf(err, "failed to update local cluster %s status: %s", provisionedCluster.ClusterID, err.Error())
	}
	metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterWaitingForKasFleetShardOperator)
	return nil
}

// reconcileStrimziOperator installs the Strimzi operator on a provisioned clusters
func (c *ClusterManager) reconcileStrimziOperator(provisionedCluster api.Cluster) (bool, error) {
	strimziOperatorAddonID := c.configService.GetConfig().OCM.StrimziOperatorAddonID
	ready, err := c.clusterService.InstallAddon(&provisionedCluster, strimziOperatorAddonID)
	if err != nil {
		return false, err
	}
	glog.V(5).Infof("ready status of %s addon on cluster %s is %t", strimziOperatorAddonID, provisionedCluster.ClusterID, ready)
	return ready, nil
}

// reconcileClusterWithConfig reconciles clusters within the dataplane-cluster-configuration file.
// New clusters will be registered if it is not yet in the database.
// A cluster will be deprovisioned if it is in the database but not in the config file.
func (c *ClusterManager) reconcileClusterWithManualConfig() []error {
	if !c.configService.GetConfig().OSDClusterConfig.IsDataPlaneManualScalingEnabled() {
		glog.Infoln("manual cluster configuration reconciliation is skipped as it is disabled")
		return []error{}
	}

	glog.Infoln("reconciling manual cluster configurations")
	allClusterIds, err := c.clusterService.ListAllClusterIds()
	if err != nil {
		return []error{errors.Wrapf(err, "failed to retrieve cluster ids from clusters")}
	}
	clusterIdsMap := make(map[string]api.Cluster)
	for _, v := range allClusterIds {
		clusterIdsMap[v.ClusterID] = v
	}

	//Create all missing clusters
	for _, p := range c.configService.GetConfig().OSDClusterConfig.ClusterConfig.MissingClusters(clusterIdsMap) {
		clusterRequest := api.Cluster{
			CloudProvider: p.CloudProvider,
			Region:        p.Region,
			MultiAZ:       p.MultiAZ,
			ClusterID:     p.ClusterId,
			Status:        p.Status,
			ProviderType:  p.ProviderType,
		}
		if err := c.clusterService.RegisterClusterJob(&clusterRequest); err != nil {
			return []error{errors.Wrapf(err, "Failed to register new cluster %s with config file", p.ClusterId)}
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
		return []error{errors.Wrapf(err, "Failed to find kafka count a cluster: %s", excessClusterIds)}
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

	err = c.clusterService.UpdateMultiClusterStatus(idsOfClustersToDeprovision, api.ClusterDeprovisioning)
	if err != nil {
		return []error{errors.Wrapf(err, "Failed to deprovisioning a cluster: %s", idsOfClustersToDeprovision)}
	} else {
		glog.Infof("Deprovisioning clusters: not found in config file: %s ", idsOfClustersToDeprovision)
	}

	return []error{}
}

// reconcileClustersForRegions creates an OSD cluster for each supported cloud provider and region where no cluster exists.
func (c *ClusterManager) reconcileClustersForRegions() []error {
	var errs []error
	if !c.configService.GetConfig().OSDClusterConfig.IsDataPlaneAutoScalingEnabled() {
		return errs
	}
	glog.Infoln("reconcile cloud providers and regions")
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
		errs = append(errs, errors.Wrapf(err, "failed to find cluster with criteria"))
		return errs
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
					ProviderType:  api.ClusterProviderOCM,
				}
				if err := c.clusterService.RegisterClusterJob(&clusterRequest); err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to auto-create cluster request in %s, region: %s", p.Name, v.Name))
					return errs
				} else {
					glog.Infof("Auto-created cluster request in %s, region: %s, Id: %s ", p.Name, v.Name, clusterRequest.ID)
				}
			} //
		} //region
	} //provider
	return errs
}

func (c *ClusterManager) buildResourceSet(ingressDNS string) types.ResourceSet {
	r := []interface{}{
		c.buildStorageClass(),
		c.buildIngressController(ingressDNS),
		c.buildObservabilityNamespaceResource(),
		c.buildObservabilityDexSecretResource(),
		c.buildObservabilityCatalogSourceResource(),
		c.buildObservabilityOperatorGroupResource(),
		c.buildObservabilitySubscriptionResource(),
		c.buildReadOnlyGroupResource(),
		c.buildDedicatedReaderClusterRoleBindingResource(),
	}

	if s := c.buildImagePullSecret(strimziAddonNamespace); s != nil {
		r = append(r, s)
	}
	if s := c.buildImagePullSecret(kasFleetshardAddonNamespace); s != nil {
		r = append(r, s)
	}
	return types.ResourceSet{
		Name:      syncsetName,
		Resources: r,
	}
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

// buildReadOnlyGroupResource creates a group to which read-only cluster users are added.
func (c *ClusterManager) buildReadOnlyGroupResource() *userv1.Group {
	return &userv1.Group{
		TypeMeta: metav1.TypeMeta{
			APIVersion: userv1.SchemeGroupVersion.String(),
			Kind:       "Group",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: readOnlyGroupName,
		},
		Users: c.configService.GetConfig().OSDClusterConfig.ReadOnlyUserList,
	}
}

// buildDedicatedReaderClusterRoleBindingResource creates a cluster role binding, associates it with the mk-readonly-access group, and attaches the dedicated-reader cluster role.
func (c *ClusterManager) buildDedicatedReaderClusterRoleBindingResource() *authv1.ClusterRoleBinding {
	return &authv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mkReadOnlyRoleBindingName,
		},
		Subjects: []k8sCoreV1.ObjectReference{
			{
				Kind:       "Group",
				APIVersion: "rbac.authorization.k8s.io",
				Name:       readOnlyGroupName,
			},
		},
		RoleRef: k8sCoreV1.ObjectReference{
			Kind:       "ClusterRole",
			Name:       dedicatedReadersRoleBindingName,
			APIVersion: "rbac.authorization.k8s.io",
		},
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

	idpInfo := types.IdentityProviderInfo{
		OpenID: &types.OpenIDIdentityProviderInfo{
			Name:         openIDIdentityProviderName,
			ClientID:     cluster.ClusterID,
			ClientSecret: clientSecret,
			Issuer:       c.osdIdpKeycloakService.GetRealmConfig().ValidIssuerURI,
		},
	}
	if _, err := c.clusterService.ConfigureAndSaveIdentityProvider(&cluster, idpInfo); err != nil {
		return err
	}
	glog.Infof("Identity provider is set up for cluster %s", cluster.ClusterID)
	return nil
}

func (c *ClusterManager) setClusterStatusCountMetrics() error {
	counters, err := c.clusterService.CountByStatus(clusterMetricsStatuses)
	if err != nil {
		return err
	}
	for _, c := range counters {
		metrics.UpdateClusterStatusCountMetric(c.Status, c.Count)
	}
	return nil
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
