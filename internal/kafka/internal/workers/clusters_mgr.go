package workers

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	kafkaConstants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"

	"strings"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/goava/di"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"

	authv1 "github.com/openshift/api/authorization/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	"github.com/pkg/errors"

	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	observabilityNamespace          = "managed-application-services-observability"
	observabilityCatalogSourceImage = "quay.io/rhoas/observability-operator-index:v3.0.10"
	observabilityOperatorGroupName  = "observability-operator-group-name"
	observabilityCatalogSourceName  = "observability-operator-manifests"
	observabilitySubscriptionName   = "observability-operator"
	observatoriumDexSecretName      = "observatorium-configuration-dex"
	observatoriumSSOSecretName      = "observatorium-configuration-red-hat-sso"
	syncsetName                     = "ext-managedservice-cluster-mgr"
	strimziAddonNamespace           = constants.StrimziOperatorNamespace
	strimziQEAddonNamespace         = "redhat-managed-kafka-operator-qe"
	kasFleetshardAddonNamespace     = constants.KASFleetShardOperatorNamespace
	kasFleetshardQEAddonNamespace   = "redhat-kas-fleetshard-operator-qe"
	openIDIdentityProviderName      = "Kafka_SRE"
	mkReadOnlyGroupName             = "mk-readonly-access"
	mkSREGroupName                  = "kafka-sre"
	mkReadOnlyRoleBindingName       = "mk-dedicated-readers"
	mkSRERoleBindingName            = "kafka-sre-cluster-admin"
	dedicatedReadersRoleBindingName = "dedicated-readers"
	clusterAdminRoleName            = "cluster-admin"
)

var clusterMetricsStatuses = []api.ClusterStatus{
	api.ClusterAccepted,
	api.ClusterProvisioning,
	api.ClusterProvisioned,
	api.ClusterCleanup,
	api.ClusterWaitingForKasFleetShardOperator,
	api.ClusterReady,
	api.ClusterFull,
	api.ClusterFailed,
	api.ClusterDeprovisioning,
}

type Worker = workers.Worker

var clusterLoggingOperatorAddonParams = []types.Parameter{
	{
		Id:    "use-cloudwatch",
		Value: "true",
	},
	{
		Id:    "use-app-logs",
		Value: "true",
	},
	{
		Id:    "use-infra-logs",
		Value: "false",
	},
	{
		Id:    "use-audit-logs",
		Value: "false",
	},
}

// ClusterManager represents a cluster manager that periodically reconciles osd clusters.

type ClusterManager struct {
	id           string
	workerType   string
	isRunning    bool
	imStop       chan struct{} //a chan used only for cancellation.
	syncTeardown sync.WaitGroup
	ClusterManagerOptions
}

type ClusterManagerOptions struct {
	di.Inject
	Reconciler                 workers.Reconciler
	OCMConfig                  *ocm.OCMConfig
	ObservabilityConfiguration *observatorium.ObservabilityConfiguration
	DataplaneClusterConfig     *config.DataplaneClusterConfig
	SupportedProviders         *config.ProviderConfig
	ClusterService             services.ClusterService
	CloudProvidersService      services.CloudProvidersService
	KasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	OsdIdpKeycloakService      sso.OsdKeycloakService
	ProviderFactory            clusters.ProviderFactory
}

type processor func() []error

// NewClusterManager creates a new cluster manager.
func NewClusterManager(o ClusterManagerOptions) *ClusterManager {
	return &ClusterManager{
		id:                    uuid.New().String(),
		workerType:            "cluster",
		ClusterManagerOptions: o,
	}
}

func (c *ClusterManager) GetStopChan() *chan struct{} {
	return &c.imStop
}

func (c *ClusterManager) GetSyncGroup() *sync.WaitGroup {
	return &c.syncTeardown
}

// GetID returns the ID that represents this worker.
func (c *ClusterManager) GetID() string {
	return c.id
}

func (c *ClusterManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the cluster manager to reconcile osd clusters.
func (c *ClusterManager) Start() {
	metrics.SetLeaderWorkerMetric(c.workerType, true)
	c.Reconciler.Start(c)
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	c.Reconciler.Stop(c)
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
		c.processWaitingForKasFleetshardOperatorClusters,
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
	deprovisioningClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterDeprovisioning)
	if serviceErr != nil {
		errs = append(errs, serviceErr)
		return errs
	} else {
		glog.Infof("deprovisioning clusters count = %d", len(deprovisioningClusters))
	}

	for i := range deprovisioningClusters {
		cluster := deprovisioningClusters[i]
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
	cleanupClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterCleanup)
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
	acceptedClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterAccepted)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "failed to list accepted clusters"))
		return errs
	} else {
		glog.Infof("accepted clusters count = %d", len(acceptedClusters))
	}

	for i := range acceptedClusters {
		cluster := acceptedClusters[i]
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
	provisioningClusters, listErr := c.ClusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list pending clusters"))
		return errs
	} else {
		glog.Infof("provisioning clusters count = %d", len(provisioningClusters))
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for i := range provisioningClusters {
		provisioningCluster := provisioningClusters[i]
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
	provisionedClusters, listErr := c.ClusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list provisioned clusters"))
		return errs
	} else {
		glog.Infof("provisioned clusters count = %d", len(provisionedClusters))
	}

	// process each local provisioned cluster and apply necessary terraforming.
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
	// Keep SyncSet up to date for clusters that are ready.
	readyClusters, listErr := c.ClusterService.ListByStatus(api.ClusterReady)
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
		if c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
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

func (c *ClusterManager) processWaitingForKasFleetshardOperatorClusters() []error {
	var errs []error
	waitingClusters, listErr := c.ClusterService.ListByStatus(api.ClusterWaitingForKasFleetShardOperator)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list waiting for Kas Fleetshard Operator clusters"))
		return errs
	} else {
		glog.Infof("waiting for Kas Fleetshard Operator clusters count = %d", len(waitingClusters))
	}

	// process each local waiting cluster and apply necessary terraforming.
	for _, waitingCluster := range waitingClusters {
		glog.V(10).Infof("waiting for Kas Fleetshard Operator cluster ClusterID = %s", waitingCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(waitingCluster, api.ClusterWaitingForKasFleetShardOperator)
		err := c.reconcileWaitingForKasFleetshardOperatorCluster(waitingCluster)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile waiting for Kas Fleetshard Operator cluster %s", waitingCluster.ClusterID))
		}
	}

	return errs
}

func (c *ClusterManager) reconcileDeprovisioningCluster(cluster *api.Cluster) error {
	if c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		siblingCluster, findClusterErr := c.ClusterService.FindCluster(services.FindClusterCriteria{
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
			return c.ClusterService.UpdateStatus(*cluster, api.ClusterReady)
		}
	}

	deleted, deleteClusterErr := c.ClusterService.Delete(cluster)
	if deleteClusterErr != nil {
		return deleteClusterErr
	}

	if !deleted {
		return nil
	}

	// cluster has been removed from cluster service. Mark it for cleanup.
	glog.Infof("Cluster %s  has been removed from cluster service.", cluster.ClusterID)
	updateStatusErr := c.ClusterService.UpdateStatus(*cluster, api.ClusterCleanup)
	if updateStatusErr != nil {
		return errors.Wrapf(updateStatusErr, "Failed to update deprovisioning cluster %s status to 'cleanup'", cluster.ClusterID)
	}

	return nil
}

func (c *ClusterManager) reconcileCleanupCluster(cluster api.Cluster) error {
	glog.Infof("Removing Dataplane cluster %s IDP client", cluster.ClusterID)
	keycloakDeregistrationErr := c.OsdIdpKeycloakService.DeRegisterClientInSSO(cluster.ID)
	if keycloakDeregistrationErr != nil {
		return errors.Wrapf(keycloakDeregistrationErr, "Failed to removed Dataplance cluster %s IDP client", cluster.ClusterID)
	}
	glog.Infof("Removing Dataplane cluster %s fleetshard service account", cluster.ClusterID)
	serviceAcountRemovalErr := c.KasFleetshardOperatorAddon.RemoveServiceAccount(cluster)
	if serviceAcountRemovalErr != nil {
		return errors.Wrapf(serviceAcountRemovalErr, "Failed to removed Dataplance cluster %s fleetshard service account", cluster.ClusterID)
	}

	glog.Infof("Soft deleting the Dataplane cluster %s from the database", cluster.ClusterID)
	deleteError := c.ClusterService.DeleteByClusterID(cluster.ClusterID)
	if deleteError != nil {
		return errors.Wrapf(deleteError, "Failed to soft delete Dataplance cluster %s from the database", cluster.ClusterID)
	}
	return nil
}

func (c *ClusterManager) reconcileReadyCluster(cluster api.Cluster) error {
	if !c.DataplaneClusterConfig.IsReadyDataPlaneClustersReconcileEnabled() {
		glog.Infof("Reconcile of dataplane ready clusters is disabled. Skipped reconcile of ready ClusterID '%s'", cluster.ClusterID)
		return nil
	}

	var err error

	err = c.reconcileClusterInstanceType(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile instance type ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	err = c.reconcileDynamicCapacityInfo(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile dynamic capacity info for ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	// resources update if needed
	if err := c.reconcileClusterResources(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile ready cluster resources %s ", cluster.ClusterID)
	}

	err = c.reconcileClusterIdentityProvider(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile identity provider of ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	err = c.reconcileClusterDNS(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile cluster dns of ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	err = c.reconcileKasFleetshardOperator(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile Kas Fleetshard Operator of ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	return nil
}

// reconcileClusterInstanceType checks wether a cluster has an instance type, if not, set to the instance type provided in the manual cluster configuration.
// If the cluster does not exists, assume the cluster supports both instance types.
func (c *ClusterManager) reconcileClusterInstanceType(cluster api.Cluster) error {
	logger.Logger.Infof("reconciling cluster = %s instance type", cluster.ClusterID)
	supportedInstanceType := api.AllInstanceTypeSupport.String()
	manualScalingEnabled := c.DataplaneClusterConfig.IsDataPlaneManualScalingEnabled()
	if manualScalingEnabled {
		supportedType, found := c.DataplaneClusterConfig.ClusterConfig.GetClusterSupportedInstanceType(cluster.ClusterID)
		if !found && cluster.SupportedInstanceType != "" {
			logger.Logger.Infof("cluster instance type already set for cluster = %s", cluster.ClusterID)
			return nil
		} else if found {
			supportedInstanceType = supportedType
		}
	}

	if cluster.SupportedInstanceType != "" && !manualScalingEnabled {
		logger.Logger.Infof("cluster instance type already set for cluster = %s and scaling type is not manual", cluster.ClusterID)
		return nil
	}

	if cluster.SupportedInstanceType != supportedInstanceType {
		cluster.SupportedInstanceType = supportedInstanceType
		err := c.ClusterService.Update(cluster)
		if err != nil {
			return errors.Wrapf(err, "failed to update instance type in database for cluster %s", cluster.ClusterID)
		}
	}

	logger.Logger.Infof("supported instance type for cluster = %s successful updated", cluster.ClusterID)
	return nil
}

func (c *ClusterManager) reconcileDynamicCapacityInfo(cluster api.Cluster) error {
	updatedDynamicCapacityInfo := map[string]api.DynamicCapacityInfo{}

	if c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		previousDynamicCapacityInfo, err := cluster.RetrieveDynamicCapacityInfo()
		if err != nil {
			return err
		}

		if len(previousDynamicCapacityInfo) > 0 {
			return nil
		}

		supportedInstanceTypes := cluster.GetSupportedInstanceTypes()
		for _, supportedInstanceType := range supportedInstanceTypes {
			config, ok := c.DataplaneClusterConfig.DynamicScalingConfig.ForInstanceType(supportedInstanceType)
			if !ok {
				continue
			}

			updatedDynamicCapacityInfo[supportedInstanceType] = api.DynamicCapacityInfo{
				MaxNodes: int32(config.ComputeNodesConfig.MaxComputeNodes),
			}
		}
	}

	_ = cluster.SetDynamicCapacityInfo(updatedDynamicCapacityInfo)
	if err := c.ClusterService.Update(cluster); err != nil {
		return errors.Wrapf(err, "failed to update instance type in database for cluster %s", cluster.ClusterID)
	}

	logger.Logger.Infof("reconciling cluster = %s dynamic config type", cluster.ClusterID)

	return nil
}

// reconcileEmptyCluster checks wether a cluster is empty and mark it for deletion.
func (c *ClusterManager) reconcileEmptyCluster(cluster api.Cluster) (bool, error) {
	glog.V(10).Infof("check if cluster is empty, ClusterID = %s", cluster.ClusterID)
	clusterFromDb, err := c.ClusterService.FindNonEmptyClusterById(cluster.ClusterID)
	if err != nil {
		return false, err
	}
	if clusterFromDb != nil {
		glog.V(10).Infof("cluster is not empty, ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	clustersByRegionAndCloudProvider, findSiblingClusterErr := c.ClusterService.ListGroupByProviderAndRegion(
		[]string{cluster.CloudProvider},
		[]string{cluster.Region},
		[]string{api.ClusterReady.String()})

	if findSiblingClusterErr != nil || len(clustersByRegionAndCloudProvider) == 0 {
		return false, findSiblingClusterErr
	}

	siblingClusterCount := clustersByRegionAndCloudProvider[0]
	if siblingClusterCount.Count <= 1 { // sibling cluster not found.
		glog.V(10).Infof("no valid sibling found for cluster ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	updateStatusErr := c.ClusterService.UpdateStatus(cluster, api.ClusterDeprovisioning)
	return updateStatusErr == nil, updateStatusErr
}

func (c *ClusterManager) reconcileWaitingForKasFleetshardOperatorCluster(cluster api.Cluster) error {
	if err := c.reconcileClusterResources(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile  waiting for Kas Fleetshard Operator cluster resources '%s'", cluster.ClusterID)
	}

	if err := c.reconcileClusterIdentityProvider(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile identity provider of waiting for Kas Fleetshard Operator cluster %s: %s", cluster.ClusterID, err.Error())
	}

	if err := c.reconcileKasFleetshardOperator(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile Kas Fleetshard Operator of waiting for Kas Fleetshard Operator cluster %s: %s", cluster.ClusterID, err.Error())
	}

	return nil
}

func (c *ClusterManager) reconcileProvisionedCluster(cluster api.Cluster) error {
	machinePoolsReconciled, err := c.reconcileClusterMachinePools(cluster)
	if err != nil {
		return err
	}
	glog.V(10).Infof("status of Machine Pools reconciling is %v", machinePoolsReconciled)

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

	addonsReconciled, addOnErr := c.reconcileAddonOperator(cluster)
	if addOnErr != nil {
		return errors.WithMessagef(addOnErr, "failed to reconcile cluster %s addon operator: %s", cluster.ClusterID, addOnErr.Error())
	}

	if machinePoolsReconciled && addonsReconciled {
		glog.V(0).Infof("Set cluster status to %s for cluster %s", api.ClusterWaitingForKasFleetShardOperator, cluster.ClusterID)
		if err := c.ClusterService.
			UpdateStatus(cluster, api.ClusterWaitingForKasFleetShardOperator); err != nil {
			return errors.Wrapf(err, "failed to update local cluster %s status: %s", cluster.ClusterID, err.Error())
		}
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterWaitingForKasFleetShardOperator)
	}

	return nil
}

func (c *ClusterManager) reconcileClusterDNS(cluster api.Cluster) error {
	// Return if the clusterDNS is already set.
	if cluster.ClusterDNS != "" {
		return nil
	}

	_, dnsErr := c.ClusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster %s: GetClusterDNS %s", cluster.ClusterID, dnsErr.Error())
	}

	return nil
}

func (c *ClusterManager) reconcileKasFleetshardOperator(cluster api.Cluster) error {
	if params, err := c.KasFleetshardOperatorAddon.ReconcileParameters(cluster); err != nil {
		return errors.WithMessagef(err, "failed to reconcile kas-fleet-shard parameters of %s cluster %s: %s", cluster.Status, cluster.ClusterID, err.Error())
	} else {
		if cluster.ClientID == "" || cluster.ClientSecret == "" {
			cluster.ClientID = params.GetParam(services.KasFleetshardOperatorParamServiceAccountId)
			cluster.ClientSecret = params.GetParam(services.KasFleetshardOperatorParamServiceAccountSecret)
			if err := c.ClusterService.Update(cluster); err != nil {
				return errors.WithMessagef(err, "failed to reconcile clientID of %s cluster %s: %s", cluster.Status, cluster.ClusterID, err.Error())
			}
		}
	}
	return nil
}

func (c *ClusterManager) reconcileClusterResources(cluster api.Cluster) error {
	resourceSet := c.buildResourceSet(cluster)
	if err := c.ClusterService.ApplyResources(&cluster, resourceSet); err != nil {
		return errors.Wrapf(err, "failed to apply resources for cluster %s", cluster.ClusterID)
	}

	return nil
}

func (c *ClusterManager) reconcileAcceptedCluster(cluster *api.Cluster) error {
	_, err := c.ClusterService.Create(cluster)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster for request %s", cluster.ID)
	}

	return nil
}

// reconcileClusterStatus updates the provided clusters stored status to reflect it's current state.
func (c *ClusterManager) reconcileClusterStatus(cluster *api.Cluster) (*api.Cluster, error) {
	updatedCluster, err := c.ClusterService.CheckClusterStatus(cluster)
	if err != nil {
		return nil, err
	}
	if updatedCluster.Status == api.ClusterFailed {
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterFailed)
		metrics.IncreaseClusterTotalOperationsCountMetric(kafkaConstants.ClusterOperationCreate)
	}
	return updatedCluster, nil
}

func (c *ClusterManager) reconcileAddonOperator(provisionedCluster api.Cluster) (bool, error) {
	strimziOperatorIsReady, err := c.reconcileStrimziOperator(provisionedCluster)
	if err != nil {
		return false, err
	}

	clusterLoggingOperatorIsReady := false

	if c.OCMConfig.ClusterLoggingOperatorAddonID != "" {
		ready, err := c.reconcileClusterLoggingOperator(provisionedCluster)
		if err != nil {
			return false, err
		}
		clusterLoggingOperatorIsReady = ready
	}

	glog.Infof("Provisioning kas-fleetshard-operator as it is enabled")
	kasFleetshardOperatorIsReady, params, errs := c.KasFleetshardOperatorAddon.Provision(provisionedCluster)
	if errs != nil {
		return false, errs
	}

	if provisionedCluster.ClientID == "" || provisionedCluster.ClientSecret == "" {
		provisionedCluster.ClientID = params.GetParam(services.KasFleetshardOperatorParamServiceAccountId)
		provisionedCluster.ClientSecret = params.GetParam(services.KasFleetshardOperatorParamServiceAccountSecret)
		if err := c.ClusterService.Update(provisionedCluster); err != nil {
			return false, errors.WithMessagef(err, "failed to reconcile clientID of %s cluster %s: %s", provisionedCluster.Status, provisionedCluster.ClusterID, err.Error())
		}
	}

	if strimziOperatorIsReady && kasFleetshardOperatorIsReady && (clusterLoggingOperatorIsReady || c.OCMConfig.ClusterLoggingOperatorAddonID == "") {
		return true, nil
	}

	return false, nil
}

// reconcileStrimziOperator installs the Strimzi operator on a provisioned clusters
func (c *ClusterManager) reconcileStrimziOperator(provisionedCluster api.Cluster) (bool, error) {
	ready, err := c.ClusterService.InstallStrimzi(&provisionedCluster)
	if err != nil {
		return false, err
	}
	glog.V(5).Infof("ready status of strimzi installation on cluster %s is %t", provisionedCluster.ClusterID, ready)
	return ready, nil
}

// reconcileClusterLoggingOperator installs the cluster logging operator on provisioned clusters
func (c *ClusterManager) reconcileClusterLoggingOperator(provisionedCluster api.Cluster) (bool, error) {
	ready, err := c.ClusterService.InstallClusterLogging(&provisionedCluster, clusterLoggingOperatorAddonParams)
	if err != nil {
		return false, err
	}
	glog.V(5).Infof("ready status of cluster logging installation on cluster %s is %t", provisionedCluster.ClusterID, ready)
	return ready, nil
}

// reconcileClusterWithConfig reconciles clusters within the dataplane-cluster-configuration file.
// New clusters will be registered if it is not yet in the database.
// A cluster will be deprovisioned if it is in the database but not in the coreConfig file.
func (c *ClusterManager) reconcileClusterWithManualConfig() []error {
	if !c.DataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
		glog.Infoln("manual cluster configuration reconciliation is skipped as it is disabled")
		return []error{}
	}

	glog.Infoln("reconciling manual cluster configurations")
	allClusterIds, err := c.ClusterService.ListAllClusterIds()
	if err != nil {
		return []error{errors.Wrapf(err, "failed to retrieve cluster ids from clusters")}
	}
	clusterIdsMap := make(map[string]api.Cluster)
	for _, v := range allClusterIds {
		clusterIdsMap[v.ClusterID] = v
	}

	//Create all missing clusters
	for _, p := range c.DataplaneClusterConfig.ClusterConfig.MissingClusters(clusterIdsMap) {
		clusterRequest := api.Cluster{
			CloudProvider:         p.CloudProvider,
			Region:                p.Region,
			MultiAZ:               p.MultiAZ,
			ClusterID:             p.ClusterId,
			Status:                p.Status,
			ProviderType:          p.ProviderType,
			ClusterDNS:            p.ClusterDNS,
			SupportedInstanceType: p.SupportedInstanceType,
		}
		if err := c.ClusterService.RegisterClusterJob(&clusterRequest); err != nil {
			return []error{errors.Wrapf(err, "Failed to register new cluster %s with config file", p.ClusterId)}
		} else {
			glog.Infof("Registered a new cluster with config file: %s ", p.ClusterId)
		}
	}

	// Remove all clusters that are not in the config file.
	excessClusterIds := c.DataplaneClusterConfig.ClusterConfig.ExcessClusters(clusterIdsMap)
	if len(excessClusterIds) == 0 {
		return nil
	}

	kafkaInstanceCount, err := c.ClusterService.FindKafkaInstanceCount(excessClusterIds)
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

	err = c.ClusterService.UpdateMultiClusterStatus(idsOfClustersToDeprovision, api.ClusterDeprovisioning)
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
	if !c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		return errs
	}
	glog.Infoln("reconcile cloud providers and regions")
	var providers []string
	var regions []string
	status := api.StatusForValidCluster
	//gather the supported providers and regions
	providerList := c.SupportedProviders.ProvidersConfig.SupportedProviders
	for _, v := range providerList {
		providers = append(providers, v.Name)
		for _, r := range v.Regions {
			regions = append(regions, r.Name)
		}
	}

	// get a list of clusters in Map group by their provider and region.
	grpResult, err := c.ClusterService.ListGroupByProviderAndRegion(providers, regions, status)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to find cluster with criteria"))
		return errs
	}

	grpResultMap := make(map[string]*services.ResGroupCPRegion)
	for _, v := range grpResult {
		grpResultMap[v.Provider+"."+v.Region] = v
	}

	// create all the missing clusters in the supported provider and regions.
	for _, p := range providerList {
		for _, v := range p.Regions {
			if _, exist := grpResultMap[p.Name+"."+v.Name]; !exist {
				clusterRequest := api.Cluster{
					CloudProvider:         p.Name,
					Region:                v.Name,
					MultiAZ:               true,
					Status:                api.ClusterAccepted,
					ProviderType:          api.ClusterProviderOCM,
					SupportedInstanceType: api.AllInstanceTypeSupport.String(), // TODO - make sure we use the appropriate instance type.
				}
				if err := c.ClusterService.RegisterClusterJob(&clusterRequest); err != nil {
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

func (c *ClusterManager) buildResourceSet(cluster api.Cluster) types.ResourceSet {
	r := []interface{}{
		c.buildReadOnlyGroupResource(),
		c.buildDedicatedReaderClusterRoleBindingResource(),
		c.buildKafkaSREGroupResource(),
		c.buildKafkaSreClusterRoleBindingResource(),
		c.buildObservabilityNamespaceResource(),
		c.buildObservatoriumDexSecretResource(),
		c.buildObservatoriumSSOSecretResource(),
		c.buildObservabilityCatalogSourceResource(),
		c.buildObservabilityOperatorGroupResource(),
		c.buildObservabilitySubscriptionResource(),
	}
	strimziNamespace := strimziAddonNamespace
	if c.OCMConfig.StrimziOperatorAddonID == "managed-kafka-qe" {
		strimziNamespace = strimziQEAddonNamespace
	}
	kasFleetshardNamespace := kasFleetshardAddonNamespace
	if c.OCMConfig.KasFleetshardAddonID == "kas-fleetshard-operator-qe" {
		kasFleetshardNamespace = kasFleetshardQEAddonNamespace
	}

	// For standalone clusters, make sure that the namespaces is read from the config
	// and that they are created before the pull secrets that references them
	if cluster.ProviderType == api.ClusterProviderStandalone {
		strimziNamespace = c.DataplaneClusterConfig.StrimziOperatorOLMConfig.Namespace
		kasFleetshardNamespace = c.DataplaneClusterConfig.KasFleetshardOperatorOLMConfig.Namespace
		r = append(r, &k8sCoreV1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strimziNamespace,
			},
		}, &k8sCoreV1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: kasFleetshardNamespace,
			},
		})
	}

	if s := c.buildImagePullSecret(strimziNamespace); s != nil {
		r = append(r, s)
	}
	if s := c.buildImagePullSecret(kasFleetshardNamespace); s != nil {
		r = append(r, s)
	}
	return types.ResourceSet{
		Name:      syncsetName,
		Resources: r,
	}
}

func (c *ClusterManager) buildObservabilityNamespaceResource() *k8sCoreV1.Namespace {
	return &k8sCoreV1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: observabilityNamespace,
		},
	}
}

func (c *ClusterManager) buildObservatoriumDexSecretResource() *k8sCoreV1.Secret {
	observabilityConfig := c.ObservabilityConfiguration
	stringDataMap := map[string]string{
		"authType":    observatorium.AuthTypeDex,
		"gateway":     observabilityConfig.ObservatoriumGateway,
		"tenant":      observabilityConfig.ObservatoriumTenant,
		"dexUrl":      observabilityConfig.DexUrl,
		"dexPassword": observabilityConfig.DexPassword,
		"dexSecret":   observabilityConfig.DexSecret,
		"dexUsername": observabilityConfig.DexUsername,
	}
	return &k8sCoreV1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observatoriumDexSecretName,
			Namespace: observabilityNamespace,
		},
		Type:       k8sCoreV1.SecretTypeOpaque,
		StringData: stringDataMap,
	}
}

func (c *ClusterManager) buildObservatoriumSSOSecretResource() *k8sCoreV1.Secret {
	observabilityConfig := c.ObservabilityConfiguration
	stringDataMap := map[string]string{
		"authType":               observatorium.AuthTypeSso,
		"gateway":                observabilityConfig.RedHatSsoGatewayUrl,
		"tenant":                 observabilityConfig.RedHatSsoTenant,
		"redHatSsoAuthServerUrl": observabilityConfig.RedHatSsoAuthServerUrl,
		"redHatSsoRealm":         observabilityConfig.RedHatSsoRealm,
		"metricsClientId":        observabilityConfig.MetricsClientId,
		"metricsSecret":          observabilityConfig.MetricsSecret,
		"logsClientId":           observabilityConfig.LogsClientId,
		"logsSecret":             observabilityConfig.LogsSecret,
	}
	return &k8sCoreV1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observatoriumSSOSecretName,
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
			StartingCSV:            "observability-operator.v3.0.10",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
			Package:                observabilitySubscriptionName,
		},
	}
}

func (c *ClusterManager) buildImagePullSecret(namespace string) *k8sCoreV1.Secret {
	content := c.DataplaneClusterConfig.ImagePullDockerConfigContent
	if strings.TrimSpace(content) == "" {
		return nil
	}

	dataMap := map[string][]byte{
		k8sCoreV1.DockerConfigJsonKey: []byte(content),
	}

	return &k8sCoreV1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaConstants.ImagePullSecretName,
			Namespace: namespace,
		},
		Type: k8sCoreV1.SecretTypeDockerConfigJson,
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
			Name: mkReadOnlyGroupName,
		},
		Users: c.DataplaneClusterConfig.ReadOnlyUserList,
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
				Name:       mkReadOnlyGroupName,
			},
		},
		RoleRef: k8sCoreV1.ObjectReference{
			Kind:       "ClusterRole",
			Name:       dedicatedReadersRoleBindingName,
			APIVersion: "rbac.authorization.k8s.io",
		},
	}
}

// buildReadOnlyGroupResource creates a group to which read-only cluster users are added.
func (c *ClusterManager) buildKafkaSREGroupResource() *userv1.Group {
	return &userv1.Group{
		TypeMeta: metav1.TypeMeta{
			APIVersion: userv1.SchemeGroupVersion.String(),
			Kind:       "Group",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mkSREGroupName,
		},
		Users: c.DataplaneClusterConfig.KafkaSREUsers,
	}
}

// buildClusterAdminClusterRoleBindingResource creates a cluster role binding, associates it with the kafka-sre group, and attaches the cluster-admin role.
func (c *ClusterManager) buildKafkaSreClusterRoleBindingResource() *authv1.ClusterRoleBinding {
	return &authv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mkSRERoleBindingName,
		},
		Subjects: []k8sCoreV1.ObjectReference{
			{
				Kind:       "Group",
				APIVersion: "rbac.authorization.k8s.io",
				Name:       mkSREGroupName,
			},
		},
		RoleRef: k8sCoreV1.ObjectReference{
			Kind:       "ClusterRole",
			Name:       clusterAdminRoleName,
			APIVersion: "rbac.authorization.k8s.io",
		},
	}
}

func (c *ClusterManager) buildMachinePoolRequest(machinePoolID string, supportedInstanceType string, cluster api.Cluster) (*types.MachinePoolRequest, error) {
	dynamicScalingConfig, found := c.DataplaneClusterConfig.DynamicScalingConfig.ForInstanceType(supportedInstanceType)
	if !found {
		return nil, fmt.Errorf("No dynamic scaling configuration found for instance type '%s'", supportedInstanceType)
	}
	machinePoolLabels := map[string]string{
		"bf2.org/kafkaInstanceProfile": supportedInstanceType,
	}
	machinePoolTaint := types.CluserNodeTaint{
		Effect: "NoExecute",
		Key:    "bf2.org/kafkaInstanceProfileType",
		Value:  supportedInstanceType,
	}
	machinePoolTaints := []types.CluserNodeTaint{machinePoolTaint}
	machinePool := &types.MachinePoolRequest{
		ID:                 machinePoolID,
		InstanceSize:       c.DataplaneClusterConfig.ComputeMachineType,
		MultiAZ:            cluster.MultiAZ,
		AutoScalingEnabled: true,
		AutoScaling: types.MachinePoolAutoScaling{
			MinNodes: 0,
			MaxNodes: dynamicScalingConfig.ComputeNodesConfig.MaxComputeNodes,
		},
		ClusterID:  cluster.ClusterID,
		NodeLabels: machinePoolLabels,
		NodeTaints: machinePoolTaints,
	}

	return machinePool, nil
}

func (c *ClusterManager) reconcileClusterMachinePools(cluster api.Cluster) (bool, error) {
	if !c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		return true, nil
	}
	// TODO should we implement machinepool creation of the additional 'kafka'
	// machinepool when cluster scaling is manual or none???

	providerClient, err := c.ProviderFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, err
	}

	glog.V(10).Infof("Reconciling MachinePools for clusterID '%s'", cluster.ClusterID)
	supportedInstanceTypes := cluster.GetSupportedInstanceTypes()
	// Ensure a MachinePool is created for each supported instance type
	dynamicCapacityInfo := map[string]api.DynamicCapacityInfo{}

	for _, supportedInstanceType := range supportedInstanceTypes {
		machinePoolID := fmt.Sprintf("kafka-%s", supportedInstanceType)
		existingMachinePool, err := providerClient.GetMachinePool(cluster.ClusterID, machinePoolID)
		if err != nil {
			return false, err
		}
		if existingMachinePool != nil {
			dynamicCapacityInfo[supportedInstanceType] = api.DynamicCapacityInfo{
				MaxNodes: int32(existingMachinePool.AutoScaling.MaxNodes),
			}
			glog.V(10).Infof("MachinePool '%s' for clusterID '%s' already created. No further reconciling of it needed.", machinePoolID, cluster.ClusterID)
			continue
		}

		machinePoolRequest, err := c.buildMachinePoolRequest(machinePoolID, supportedInstanceType, cluster)
		if err != nil {
			return false, err
		}

		glog.Infof("MachinePool '%s' for clusterID '%s' does not exist. Creating it...", cluster.ClusterID, machinePoolID)
		_, err = providerClient.CreateMachinePool(machinePoolRequest)
		if err != nil {
			return false, err
		}
		dynamicCapacityInfo[supportedInstanceType] = api.DynamicCapacityInfo{
			MaxNodes: int32(machinePoolRequest.AutoScaling.MaxNodes),
		}
	}

	// update the dyamic capacity info for them to be stored in the database
	_ = cluster.SetDynamicCapacityInfo(dynamicCapacityInfo)
	srvErr := c.ClusterService.Update(cluster)
	if srvErr != nil {
		return false, errors.Wrapf(srvErr, "failed to update cluster")
	}
	return true, nil
}

func (c *ClusterManager) reconcileClusterIdentityProvider(cluster api.Cluster) error {
	if !c.DataplaneClusterConfig.EnableKafkaSreIdentityProviderConfiguration {
		glog.Infof("Configuration of data plane identity providers is disabled. Skipping configuring the identity provider for ClusterID '%s'", cluster.ClusterID)
		return nil
	}

	if cluster.IdentityProviderID != "" {
		return nil
	}

	// identity provider not yet created, let's create a new one.
	glog.Infof("Setting up the identity provider for cluster %s", cluster.ClusterID)
	clusterDNS, dnsErr := c.ClusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster identity provider %s: %s", cluster.ClusterID, dnsErr.Error())
	}

	callbackUri := fmt.Sprintf("https://oauth-openshift.%s/oauth2callback/%s", clusterDNS, openIDIdentityProviderName)
	clientSecret, ssoErr := c.OsdIdpKeycloakService.RegisterClientInSSO(cluster.ID, callbackUri)
	if ssoErr != nil {
		return errors.WithMessagef(ssoErr, "failed to reconcile cluster identity provider %s: %s", cluster.ClusterID, ssoErr.Error())
	}

	idpInfo := types.IdentityProviderInfo{
		OpenID: &types.OpenIDIdentityProviderInfo{
			Name:         openIDIdentityProviderName,
			ClientID:     cluster.ID,
			ClientSecret: clientSecret,
			Issuer:       c.OsdIdpKeycloakService.GetRealmConfig().ValidIssuerURI,
		},
	}
	if _, err := c.ClusterService.ConfigureAndSaveIdentityProvider(&cluster, idpInfo); err != nil {
		return err
	}
	glog.Infof("Identity provider is set up for cluster %s", cluster.ClusterID)
	return nil
}

func (c *ClusterManager) setClusterStatusCountMetrics() error {
	counters, err := c.ClusterService.CountByStatus(clusterMetricsStatuses)
	if err != nil {
		return err
	}
	for _, c := range counters {
		metrics.UpdateClusterStatusCountMetric(c.Status, c.Count)
	}
	return nil
}

func (c *ClusterManager) setKafkaPerClusterCountMetrics() error {
	if counters, err := c.ClusterService.FindKafkaInstanceCount([]string{}); err != nil {
		return err
	} else {
		for _, counter := range counters {
			clusterExternalID, err := c.ClusterService.GetExternalID(counter.Clusterid)
			if err != nil {
				return err
			}
			metrics.UpdateKafkaPerClusterCountMetric(counter.Clusterid, clusterExternalID, counter.Count)
		}
	}
	return nil
}
