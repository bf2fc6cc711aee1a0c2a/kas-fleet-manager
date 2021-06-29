package clusters

import (
	"encoding/json"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	patchTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// fieldManager indicates that the kas-fleet-manager will be used as a field manager for conflict resolution
const fieldManager = "kas-fleet-manager"

// lastAppliedConfigurationAnnotation is an annotation applied in a resources which tracks the last applied configuration of a resource.
// this is used to decide whether a new apply request should be taken into account or not
const lastAppliedConfigurationAnnotation = "kas-fleet-manager/last-applied-resource-configuration"

// force is a patch option to force applying new changes when conflicts occurs
var force = true

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
	clientConfig, err := clientcmd.NewNonInteractiveClientConfig(*s.dataplaneClusterConfig.RawKubernetesConfig, override.CurrentContext,
		override, &clientcmd.ClientConfigLoadingRules{}).
		ClientConfig()

	if err != nil {
		return nil, err
	}

	for _, resource := range resources.Resources {
		_, err = applyResource(clientConfig, resource)
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

func applyResource(restConfig *rest.Config, resourceObj interface{}) (runtime.Object, error) {
	// parse resource obj
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

	kubeClientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	// Create a REST mapper that tracks information about the available resources in the cluster.
	groupResources, err := restmapper.GetAPIGroupResources(kubeClientset.Discovery())
	if err != nil {
		return nil, err
	}
	rm := restmapper.NewDiscoveryRESTMapper(groupResources)

	// Get some metadata needed to make the REST request.
	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}

	namespace, err := meta.NewAccessor().Namespace(&obj)
	if err != nil {
		return nil, err
	}

	// Create a client specifically for creating the object.
	restClient, err := newRestClient(restConfig, mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return nil, err
	}

	restHelper := resource.NewHelper(restClient, mapping)
	name, err := meta.NewAccessor().Name(&obj)
	if err != nil {
		return nil, err
	}

	// check if resources needs to be applied
	applyChanges, err := shouldApplyChanges(restHelper, namespace, name, newConfiguration)
	if err != nil {
		return &obj, err
	}

	if !applyChanges { // no need to apply changes as resource has not changed
		return &obj, nil
	}

	// apply new changes which will lead to creation of new resources
	return applyChangesFn(restHelper, namespace, name, obj)
}

func shouldApplyChanges(restHelper *resource.Helper, namespace string, name string, newConfiguration string) (bool, error) {
	existingObj, err := restHelper.Get(namespace, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return true, nil
		} else {
			return false, err
		}
	}

	existing, err := runtime.DefaultUnstructuredConverter.ToUnstructured(existingObj)
	if err != nil {
		return false, err
	}
	original := unstructured.Unstructured{
		Object: existing,
	}
	originalAnnotations := original.GetAnnotations()
	if originalAnnotations != nil {
		lastApplied, ok := originalAnnotations[lastAppliedConfigurationAnnotation]
		if !ok {
			return true, nil // new object, create it
		} else {
			return newConfiguration != lastApplied, nil // check if configuration has changed before applying changes
		}
	}

	return true, nil
}

func applyChangesFn(restHelper *resource.Helper, namespace string, name string, obj unstructured.Unstructured) (runtime.Object, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	patchOptions := &metav1.PatchOptions{
		FieldManager: fieldManager,
		Force:        &force,
	}

	// try to apply new update using server-side apply
	result, err := restHelper.Patch(namespace, name, patchTypes.ApplyPatchType, data, patchOptions)

	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to create typed patch object")) {
		result, err = restHelper.CreateWithOptions(namespace, true, &obj, &metav1.CreateOptions{
			FieldManager: fieldManager,
		})
	}

	// replace if it already exist
	if err != nil && strings.Contains(err.Error(), "already exist") {
		result, err = restHelper.Replace(namespace, name, true, &obj)
	}

	// ignore immutability failure to apply patch
	if err != nil && strings.Contains(err.Error(), "field is immutable") {
		return &obj, nil
	}
	return result, err
}

func newRestClient(restConfig *rest.Config, gv schema.GroupVersion) (rest.Interface, error) {
	restConfig.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	restConfig.GroupVersion = &gv
	if len(gv.Group) == 0 {
		restConfig.APIPath = "/api"
	} else {
		restConfig.APIPath = "/apis"
	}
	return rest.RESTClientFor(restConfig)
}
