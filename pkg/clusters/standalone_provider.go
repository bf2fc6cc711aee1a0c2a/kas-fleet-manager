package clusters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	coreV1 "k8s.io/api/core/v1"
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

const fieldManager = "kas-fleet-manager"

type StandaloneProvider struct {
	connectionFactory *db.ConnectionFactory
	osdClusterConfig  *config.OSDClusterConfig
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider(connectionFactory *db.ConnectionFactory, osdClusterConfig *config.OSDClusterConfig) *StandaloneProvider {
	return &StandaloneProvider{
		connectionFactory: connectionFactory,
		osdClusterConfig:  osdClusterConfig,
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
	contextName := "change-me" // change this to read the context name of the given cluster
	override := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	clientConfig, err := clientcmd.NewNonInteractiveClientConfig(s.osdClusterConfig.RawKubernetesConfig, override.CurrentContext,
		override, &clientcmd.ClientConfigLoadingRules{}).
		ClientConfig()

	if err != nil {
		return nil, err
	}

	// change this to be an element of resourceset
	configMap := coreV1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"jira.ticket": "https://issues.redhat.com/browse/MGDSTRM-3814",
		},
	}

	data, err := json.Marshal(configMap)
	if err != nil {
		return nil, err
	}

	var resource unstructured.Unstructured
	err = json.Unmarshal(data, &resource)

	if err != nil {
		return nil, err
	}
	_, err = applyResource(clientConfig, &resource)

	if err != nil {
		return nil, err
	}

	return &resources, nil // NOOP for now
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

func applyResource(restConfig *rest.Config, resourceObj runtime.Object) (runtime.Object, error) {
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
	gvk := resourceObj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}

	namespace, err := meta.NewAccessor().Namespace(resourceObj)
	if err != nil {
		return nil, err
	}

	// Create a client specifically for creating the object.
	restClient, err := newRestClient(restConfig, mapping.GroupVersionKind.GroupVersion())
	if err != nil {
		return nil, err
	}

	restHelper := resource.NewHelper(restClient, mapping)
	name, err := meta.NewAccessor().Name(resourceObj)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resourceObj)
	if err != nil {
		return nil, err
	}

	return restHelper.Patch(namespace, name, patchTypes.ApplyPatchType, data, &metav1.PatchOptions{
		FieldManager: fieldManager,
	})
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

func newStandaloneProvider(connectionFactory *db.ConnectionFactory, osdClusterConfig *config.OSDClusterConfig) *StandaloneProvider {
	return &StandaloneProvider{
		connectionFactory: connectionFactory,
		osdClusterConfig:  osdClusterConfig,
	}
}
