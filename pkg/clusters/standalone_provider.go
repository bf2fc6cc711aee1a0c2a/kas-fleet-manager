package clusters

import (
	"context"
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// fieldManager indicates that the kas-fleet-manager will be used as a field manager for conflict resolution
const fieldManager = "kas-fleet-manager"

// lastAppliedConfigurationAnnotation is an annotation applied in a resources which tracks the last applied configuration of a resource.
// this is used to decide whether a new apply request should be taken into account or not
const lastAppliedConfigurationAnnotation = "kas-fleet-manager/last-applied-resource-configuration"

var ctx = context.Background()

type StandaloneProvider struct {
	connectionFactory      *db.ConnectionFactory
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider(connectionFactory *db.ConnectionFactory, dataplaneClusterConfig *config.DataplaneClusterConfig) *StandaloneProvider {
	return &StandaloneProvider{
		connectionFactory:      connectionFactory,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

func (s *StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, nil
}

func (s *StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	return true, nil
}

func (s *StandaloneProvider) InstallStrimzi(spec *types.ClusterSpec) (bool, error) {
	return true, nil // NOOP for now. TODO See kas-installer repo on how to install strimzi
}

func (s *StandaloneProvider) InstallClusterLogging(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return true, nil // NOOP for now
}

func (s *StandaloneProvider) InstallKasFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return true, nil // NOOP for now. TODO see kas-installer repo on how to install kas-fleet-shard
}

func (s *StandaloneProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	spec.Status = api.ClusterProvisioned
	return spec, nil
}

func (s *StandaloneProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	return "", nil // NOOP for now
}

func (s *StandaloneProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	return &identityProvider, nil // NOOP
}

func (s *StandaloneProvider) ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
	if s.dataplaneClusterConfig.RawKubernetesConfig == nil {
		return &resources, nil // no kubeconfig read, do nothing.
	}

	contextName := s.dataplaneClusterConfig.FindClusterNameByClusterId(clusterSpec.InternalID)
	override := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	config := *s.dataplaneClusterConfig.RawKubernetesConfig
	restConfig, err := clientcmd.NewNonInteractiveClientConfig(config, override.CurrentContext, override, &clientcmd.ClientConfigLoadingRules{}).
		ClientConfig()

	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	// Create a REST mapper that tracks information about the available resources in the cluster.
	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	discoveryCachedClient := memory.NewMemCacheClient(dc)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryCachedClient)

	for _, resource := range resources.Resources {
		_, err = applyResource(dynamicClient, mapper, resource)
		if err != nil {
			return nil, err
		}
	}

	return &resources, nil
}

func (s *StandaloneProvider) ScaleUp(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

func (s *StandaloneProvider) ScaleDown(clusterSpec *types.ClusterSpec, decrement int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

func (s *StandaloneProvider) SetComputeNodes(clusterSpec *types.ClusterSpec, numNodes int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

func (s *StandaloneProvider) GetComputeNodes(spec *types.ClusterSpec) (*types.ComputeNodesInfo, error) {
	return &types.ComputeNodesInfo{}, nil // NOOP
}

func (s *StandaloneProvider) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	type Cluster struct {
		CloudProvider string
	}
	dbConn := s.connectionFactory.New().
		Model(&Cluster{}).
		Distinct("cloud_provider").
		Where("provider_type = ?", api.ClusterProviderStandalone.String()).
		Where("status NOT IN (?)", api.ClusterDeletionStatuses)

	var results []Cluster
	err := dbConn.Find(&results).Error
	if err != nil {
		return nil, err
	}

	items := []types.CloudProviderInfo{}
	for _, result := range results {
		items = append(items, types.CloudProviderInfo{
			ID:          result.CloudProvider,
			Name:        result.CloudProvider,
			DisplayName: result.CloudProvider,
		})
	}

	return &types.CloudProviderInfoList{Items: items}, nil
}

func (s *StandaloneProvider) GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	type Cluster struct {
		Region  string
		MultiAZ bool
	}
	dbConn := s.connectionFactory.New().
		Model(&Cluster{}).
		Distinct("region", "multi_az").
		Where("cloud_provider = ?", providerInf.ID).
		Where("provider_type = ?", api.ClusterProviderStandalone.String()).
		Where("status NOT IN (?)", api.ClusterDeletionStatuses)

	var results []Cluster
	err := dbConn.Find(&results).Error
	if err != nil {
		return nil, err
	}

	items := []types.CloudProviderRegionInfo{}
	for _, result := range results {
		items = append(items, types.CloudProviderRegionInfo{
			ID:              result.Region,
			Name:            result.Region,
			DisplayName:     result.Region,
			SupportsMultiAZ: result.MultiAZ,
			CloudProviderID: providerInf.ID,
		})
	}

	return &types.CloudProviderRegionInfoList{Items: items}, nil
}

func applyResource(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, resourceObj interface{}) (runtime.Object, error) {
	// parse resource obj to unstructure.Unstructered
	data, err := json.Marshal(resourceObj)
	if err != nil {
		return nil, err
	}

	var obj unstructured.Unstructured
	err = json.Unmarshal(data, &obj)

	if err != nil {
		return nil, err
	}

	newConfiguration := string(data)
	newAnnotations := obj.GetAnnotations()
	if newAnnotations == nil {
		newAnnotations = map[string]string{}
		obj.SetAnnotations(newAnnotations)
	}
	// add last configuration annotation with contents pointing to latest marshalled resources
	// this is needed to see if new changes will need to be applied during reconciliation
	newAnnotations[lastAppliedConfigurationAnnotation] = newConfiguration
	obj.SetAnnotations(newAnnotations)

	// Find Group Version resource for rest mapping
	gvk := obj.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	resourceToApply := &obj
	namespace, err := meta.NewAccessor().Namespace(resourceToApply)
	if err != nil {
		return nil, err
	}

	var dr dynamic.ResourceInterface
	if namespace != "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = dynamicClient.Resource(mapping.Resource)
	}

	name, err := meta.NewAccessor().Name(resourceToApply)
	if err != nil {
		return nil, err
	}

	// check if resources needs to be applied
	existingObj, _ := dr.Get(ctx, name, metav1.GetOptions{})
	applyChanges := shouldApplyChanges(dr, existingObj, newConfiguration)

	if !applyChanges { // no need to apply changes as resource has not changed
		return existingObj, nil
	}

	// apply new changes which will lead to creation of new resources
	return applyChangesFn(dr, name, resourceToApply, existingObj)
}

func shouldApplyChanges(dynamicClient dynamic.ResourceInterface, existingObj *unstructured.Unstructured, newConfiguration string) bool {
	if existingObj == nil {
		return true
	}

	originalAnnotations := existingObj.GetAnnotations()
	if originalAnnotations != nil {
		lastApplied, ok := originalAnnotations[lastAppliedConfigurationAnnotation]
		if !ok {
			return true // new object, create it
		} else {
			return newConfiguration != lastApplied // check if configuration has changed before applying changes
		}
	}

	return true
}

func applyChangesFn(client dynamic.ResourceInterface, name string, obj *unstructured.Unstructured, existingObj *unstructured.Unstructured) (runtime.Object, error) {
	if existingObj == nil { // create object if it does not exist
		return client.Create(ctx, obj, metav1.CreateOptions{
			FieldManager: fieldManager,
		})
	}

	obj.SetResourceVersion(existingObj.GetResourceVersion())
	return client.Update(ctx, obj, metav1.UpdateOptions{
		FieldManager: fieldManager,
	})
}
