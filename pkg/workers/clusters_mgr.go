package workers

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	projectv1 "github.com/openshift/api/project/v1"
	observability "gitlab.cee.redhat.com/service/managed-services-api/pkg/api/observability/v1"
	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AWSCloudProviderID              = "aws"
	observabilityNamespace          = "managed-application-services-observability"
	observabilityAuthType           = "dex"
	observabilityDexCredentials     = "observatorium-dex-credentials"
	observabilityCatalogSourceImage = "quay.io/integreatly/observability-operator-index:latest"
	observabilityOperatorGroupName  = "observability-operator-group-name"
	observabilityCatalogSourceName  = "observability-operator-manifests"
	observabilityStackName          = "observability-stack"
	observabilitySubscriptionName   = "observability-operator"
)

var observabilityCanaryPodSelector = map[string]string{
	constants.ObservabilityCanaryPodLabelKey: constants.ObservabilityCanaryPodLabelValue,
}

// ClusterManager represents a cluster manager that periodically reconciles osd clusters
type ClusterManager struct {
	ocmClient             ocm.Client
	clusterService        services.ClusterService
	cloudProvidersService services.CloudProvidersService
	timer                 *time.Timer
	imStop                chan struct{}
	syncTeardown          sync.WaitGroup
	reconciler            Reconciler
	configService         services.ConfigService
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(clusterService services.ClusterService, cloudProvidersService services.CloudProvidersService, ocmClient ocm.Client,
	configService services.ConfigService, serverConfig config.ServerConfig) *ClusterManager {
	return &ClusterManager{
		ocmClient:             ocmClient,
		clusterService:        clusterService,
		cloudProvidersService: cloudProvidersService,
		configService:         configService,
	}
}

func (c *ClusterManager) GetStopChan() *chan struct{} {
	return &c.imStop
}

func (c *ClusterManager) GetSyncGroup() *sync.WaitGroup {
	return &c.syncTeardown
}

// Start initializes the cluster manager to reconcile osd clusters
func (c *ClusterManager) Start() {
	c.reconciler.Start(c)
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	c.reconciler.Stop(c)
}

func (c *ClusterManager) reconcile() {
	glog.V(5).Infoln("reconciling clusters")

	acceptedClusters, serviceErr := c.clusterService.ListByStatus(api.ClusterAccepted)
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list accepted clusters: %s", serviceErr.Error())
	}

	for _, cluster := range acceptedClusters {
		if err := c.reconcileAcceptedCluster(&cluster); err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile accepted cluster %s: %s", cluster.ID, err.Error())
			continue
		}
		if err := c.clusterService.UpdateStatus(cluster, api.ClusterProvisioning); err != nil {
			glog.Errorf("failed to change cluster state to provisioning %s: %s", cluster.ID, err.Error())
		}
	}

	// reconcile the status of existing clusters in a non-ready state
	cloudProviders, err := c.cloudProvidersService.GetCloudProvidersWithRegions()
	if err != nil {
		sentry.CaptureException(err)
		glog.Error("Error retrieving cloud providers and regions", err)
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
	}

	provisioningClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		sentry.CaptureException(listErr)
		glog.Errorf("failed to list pending clusters: %s", listErr.Error())
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for _, provisioningCluster := range provisioningClusters {
		reconciledCluster, err := c.reconcileClusterStatus(&provisioningCluster)
		if err != nil {
			sentry.CaptureException(err)
			glog.Errorf("failed to reconcile cluster %s status: %s", provisioningCluster.ClusterID, err.Error())
			continue
		}
		glog.V(5).Infof("reconciled cluster %s state", reconciledCluster.ClusterID)
	}

	/*
	 * Terraforming Provisioned Clusters
	 */
	provisionedClusters, listErr := c.clusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		sentry.CaptureException(listErr)
		glog.Errorf("failed to list provisioned clusters: %s", listErr.Error())
	}

	// process each local provisioned cluster and apply necessary terraforming
	for _, provisionedCluster := range provisionedClusters {
		addonInstallation, addOnErr := c.reconcileStrimziOperator(provisionedCluster.ClusterID)
		if addOnErr != nil {
			sentry.CaptureException(addOnErr)
			glog.Errorf("failed to reconcile cluster %s strimzi operator: %s", provisionedCluster.ID, addOnErr.Error())
			continue
		}

		// The cluster is ready when the state reports ready
		if addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
			_, syncsetErr := c.createSyncSet(provisionedCluster.ClusterID)
			if syncsetErr != nil {
				sentry.CaptureException(syncsetErr)
				glog.Errorf("failed to create syncset on cluster %s: %s", provisionedCluster.ClusterID, syncsetErr.Error())
				continue
			}

			if err = c.clusterService.UpdateStatus(provisionedCluster, api.ClusterReady); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to update local cluster %s status: %s", provisionedCluster.ClusterID, err.Error())
				continue
			}

			// add entry for cluster creation metric
			metrics.UpdateClusterCreationDurationMetric(metrics.JobTypeClusterCreate, time.Since(provisionedCluster.CreatedAt))
		}
		glog.V(5).Infof("reconciled cluster %s terraforming", provisionedCluster.ClusterID)
	}
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
	}
	// if cluster is neither ready nor in an error state, assume it's pending
	if needsUpdate {
		if err = c.clusterService.UpdateStatus(*cluster, cluster.Status); err != nil {
			return nil, fmt.Errorf("failed to update local cluster %s status: %w", cluster.ClusterID, err)
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

// reconcileClustersForRegions creates an OSD cluster for each region where no cluster exists
func (c *ClusterManager) reconcileClustersForRegions() error {
	if !c.configService.IsAutoCreateOSDEnabled() {
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
				cluster, err := c.clusterService.Create(&api.Cluster{
					CloudProvider: p.Name,
					Region:        v.Name,
				})
				if err != nil {
					glog.Errorf("Failed to auto-create cluster in %s, region: %s %s", p.Name, v.Name, err.Error())
				} else {
					glog.Infof("Auto-created cluster in %s, region: %s, Id: %s ", p.Name, v.Name, cluster.ID())
				}
			} //
		} //region
	} //provider
	return nil
}

// createSyncSet creates the syncset during cluster terraforming
func (c *ClusterManager) createSyncSet(clusterID string) (*clustersmgmtv1.Syncset, error) {
	// observability operator terraforming phase
	syncset, sysnsetBuilderErr := clustersmgmtv1.NewSyncset().
		ID(fmt.Sprintf("ext-%s", observabilityNamespace)).
		Resources(
			[]interface{}{
				c.buildObservabilityNamespaceResource(),
				c.buildObservabilityDexSecretResource(),
				c.buildObservabilityCatalogSourceResource(),
				c.buildObservabilityOperatorGroupResource(),
				c.buildObservabilitySubscriptionResource(),
				c.buildObservabilityStackResource(),
			}...).
		Build()

	if sysnsetBuilderErr != nil {
		return nil, fmt.Errorf("failed to create cluster terraforming sysncset: %s", sysnsetBuilderErr.Error())
	}

	return c.ocmClient.CreateSyncSet(clusterID, syncset)
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
			StartingCSV:            "observability-operator.v0.0.1",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
			Package:                observabilitySubscriptionName,
		},
	}
}

func (c *ClusterManager) buildObservabilityStackResource() *observability.Observability {
	observabilityConfig := c.configService.GetObservabilityConfiguration()

	return &observability.Observability{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "observability.redhat.com/v1",
			Kind:       "Observability",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observabilityStackName,
			Namespace: observabilityNamespace,
		},
		Spec: observability.ObservabilitySpec{
			Grafana: observability.GrafanaConfig{
				Managed: false,
			},
			KafkaNamespaceSelector: metav1.LabelSelector{
				MatchLabels: constants.NamespaceLabels,
			},
			CanaryPodSelector: metav1.LabelSelector{
				MatchLabels: observabilityCanaryPodSelector,
			},
			Observatorium: observability.ObservatoriumConfig{
				Gateway:  observabilityConfig.ObservatoriumGateway,
				Tenant:   observabilityConfig.ObservatoriumTenant,
				AuthType: observabilityAuthType,
				AuthDex: &observability.DexConfig{
					Url:                       observabilityConfig.DexUrl,
					CredentialSecretName:      observabilityDexCredentials,
					CredentialSecretNamespace: observabilityNamespace,
				},
			},
		},
	}
}
