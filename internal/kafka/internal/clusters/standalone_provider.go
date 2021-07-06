package clusters

import (
	"context"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	projectv1 "github.com/openshift/api/project/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
)

// TODO make OLM index images configurable?
const (
	strimziOperatorOLMIndexImage                 = "quay.io/osd-addons/managed-kafka:production-82b42db"
	strimziOperatorOLMPackageName                = "managed-kafka"
	strimziOperatorCatalogSourceName             = "managed-kafka-cs"
	strimziOperatorCatalogSourceNamespace        = "openshift-marketplace"
	strimziOperatorOperatorGroupName             = "managed-kafka-og"
	strimziOperatorSubscriptionName              = "managed-kafka-sub"
	strimziOperatorSubscriptionChannelName       = "alpha"
	kasFleetShardOperatorOLMIndexImage           = "quay.io/osd-addons/kas-fleetshard-operator:production-82b42db"
	kasFleetShardOperatorOLMPackageName          = "kas-fleetshard-operator"
	kasFleetShardOperatorCatalogSourceName       = "kas-fleetshard-operator-cs"
	kasFleetShardOperatorcatalogSourceNamespace  = "openshift-marketplace"
	kasFleetShardOperatorOperatorGroupName       = "kas-fleetshard-operator-og"
	kasFleetShardOperatorSubscriptionName        = "kas-fleetshard-operator-sub"
	kasFleetShardOperatorSubscriptionChannelName = "alpha"
	kasFleetShardOperatorParametersSecretName    = "addon-kas-fleetshard-operator-parameters"
)

// fieldManager indicates that the kas-fleet-manager will be used as a field manager for conflict resolution
const fieldManager = "kas-fleet-manager"

// lastAppliedConfigurationAnnotation is an annotation applied in a resources which tracks the last applied configuration of a resource.
// this is used to decide whether a new apply request should be taken into account or not
const lastAppliedConfigurationAnnotation = "kas-fleet-manager/last-applied-resource-configuration"

// kafkaSREOpenIDPSecretName is the secret name holding the clientSecret content
const kafkaSREOpenIDPSecretName = "kafka-sre-idp-secret"

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

// blank assignment to verify that StandaloneProvider implements Provider
var _ Provider = &StandaloneProvider{}

func (s *StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, nil
}

func (s *StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	return true, nil
}

func (s *StandaloneProvider) buildStrimziOperatorNamespace() *projectv1.Project {
	return &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: projectv1.SchemeGroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.StrimziOperatorNamespace,
		},
	}
}

func (s *StandaloneProvider) buildStrimziOperatorCatalogSource() *operatorsv1alpha1.CatalogSource {
	return &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strimziOperatorCatalogSourceName,
			Namespace: strimziOperatorCatalogSourceNamespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			SourceType: operatorsv1alpha1.SourceTypeGrpc,
			Image:      strimziOperatorOLMIndexImage,
		},
	}
}

func (s *StandaloneProvider) buildStrimziOperatorOperatorGroup() *operatorsv1alpha2.OperatorGroup {
	return &operatorsv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strimziOperatorOperatorGroupName,
			Namespace: constants.StrimziOperatorNamespace,
		},
		//Spec.TargetNamespaces intentionally not set, which means "select all namespaces"
		Spec: operatorsv1alpha2.OperatorGroupSpec{},
	}
}

func (s *StandaloneProvider) buildStrimziOperatorSubscription() *operatorsv1alpha1.Subscription {
	return &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strimziOperatorSubscriptionName,
			Namespace: constants.StrimziOperatorNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          strimziOperatorCatalogSourceName,
			Channel:                strimziOperatorSubscriptionChannelName,
			CatalogSourceNamespace: strimziOperatorCatalogSourceNamespace,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                strimziOperatorOLMPackageName,
		},
	}
}

func (s *StandaloneProvider) buildKASFleetShardOperatorNamespace() *projectv1.Project {
	return &projectv1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: projectv1.SchemeGroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.KASFleetShardOperatorNamespace,
		},
	}
}

func (s *StandaloneProvider) buildKASFleetShardOperatorCatalogSource() *operatorsv1alpha1.CatalogSource {
	return &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kasFleetShardOperatorCatalogSourceName,
			Namespace: kasFleetShardOperatorcatalogSourceNamespace,
		},
		Spec: v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
			Image:      kasFleetShardOperatorOLMIndexImage,
		},
	}
}

func (s *StandaloneProvider) buildKASFleetShardOperatorOperatorGroup() *operatorsv1alpha2.OperatorGroup {
	return &operatorsv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kasFleetShardOperatorOperatorGroupName,
			Namespace: constants.KASFleetShardOperatorNamespace,
		},
		//Spec.TargetNamespaces intentionally not set, which means "select all namespaces"
		Spec: operatorsv1alpha2.OperatorGroupSpec{},
	}
}

func (s *StandaloneProvider) buildKASFleetShardOperatorSubscription() *operatorsv1alpha1.Subscription {
	return &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kasFleetShardOperatorSubscriptionName,
			Namespace: constants.KASFleetShardOperatorNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          kasFleetShardOperatorCatalogSourceName,
			Channel:                kasFleetShardOperatorSubscriptionChannelName,
			CatalogSourceNamespace: kasFleetShardOperatorcatalogSourceNamespace,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                kasFleetShardOperatorOLMPackageName,
		},
	}
}

func (s *StandaloneProvider) buildKASFleetShardSyncSecret(params []types.Parameter) *v1.Secret {
	secretStringData := map[string]string{}
	for _, param := range params {
		secretStringData[param.Id] = param.Value
	}

	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kasFleetShardOperatorParametersSecretName,
			Namespace: constants.KASFleetShardOperatorNamespace,
		},
		StringData: secretStringData,
	}
}

func (s *StandaloneProvider) InstallStrimzi(clusterSpec *types.ClusterSpec) (bool, error) {
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			s.buildStrimziOperatorNamespace(),
			s.buildStrimziOperatorCatalogSource(),
			s.buildStrimziOperatorOperatorGroup(),
			s.buildStrimziOperatorSubscription(),
		},
	})

	return true, err
}

func (s *StandaloneProvider) InstallClusterLogging(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return true, nil // NOOP for now
}

func (s *StandaloneProvider) InstallKasFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			s.buildKASFleetShardOperatorNamespace(),
			s.buildKASFleetShardSyncSecret(params),
			s.buildKASFleetShardOperatorCatalogSource(),
			s.buildKASFleetShardOperatorOperatorGroup(),
			s.buildKASFleetShardOperatorSubscription(),
		},
	})

	return true, err
}

func (s *StandaloneProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	spec.Status = api.ClusterProvisioned
	return spec, nil
}

func (s *StandaloneProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	return "", nil // NOOP for now
}

func (s *StandaloneProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	// setup identity provider
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			buildOpenIDPClientSecret(identityProvider),
			buildIdentityProviderResource(identityProvider),
		},
	})

	return &identityProvider, err
}

// buildOpenIDPClientSecret builds the k8s secret which holds OpenIDP clientSecret value
// The clientSecret as indicated in https://docs.openshift.com/container-platform/4.7/authentication/identity_providers/configuring-oidc-identity-provider.html#identity-provider-creating-secret_configuring-oidc-identity-provider
func buildOpenIDPClientSecret(identityProvider types.IdentityProviderInfo) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaSREOpenIDPSecretName,
			Namespace: "openshift-config",
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"clientSecret": identityProvider.OpenID.ClientSecret,
		},
	}
}

// buildIdentityProviderResource builds the identity provider resource to be applied
// The resource is taken from https://docs.openshift.com/container-platform/4.7/authentication/identity_providers/configuring-oidc-identity-provider.html#identity-provider-oidc-CR_configuring-oidc-identity-provider
func buildIdentityProviderResource(identityProvider types.IdentityProviderInfo) map[string]interface{} {
	// Using unstructured type for now.
	// we might want to pull the type information from github.com/openshift/api at a later stage
	return map[string]interface{}{
		"apiVersion": "config.openshift.io/v1",
		"kind":       "OAuth",
		"metadata": map[string]string{
			"name": "cluster",
		},
		"spec": map[string]interface{}{
			"identityProviders": []map[string]interface{}{
				{
					"name":          identityProvider.OpenID.Name,
					"mappingMethod": "claim",
					"type":          "OpenID",
					"openID": map[string]interface{}{
						"clientID": identityProvider.OpenID.ClientID,
						"issuer":   identityProvider.OpenID.Issuer,
						"clientSecret": map[string]string{
							"name": kafkaSREOpenIDPSecretName,
						},
						"claims": map[string][]string{
							"email":             {"email"},
							"preferredUsername": {"preferred_username"},
							"last_name":         {"preferred_username"},
						},
					},
				},
			},
		},
	}
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

func applyResource(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, resource interface{}) (runtime.Object, error) {
	// parse resource obj to unstructure.Unstructered
	data, err := json.Marshal(resource)
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

	desiredObj := &obj
	namespace, err := meta.NewAccessor().Namespace(desiredObj)
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

	name, err := meta.NewAccessor().Name(desiredObj)
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
	return applyChangesFn(dr, desiredObj, existingObj)
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

func applyChangesFn(client dynamic.ResourceInterface, desiredObj *unstructured.Unstructured, existingObj *unstructured.Unstructured) (runtime.Object, error) {
	if existingObj == nil { // create object if it does not exist
		return client.Create(ctx, desiredObj, metav1.CreateOptions{
			FieldManager: fieldManager,
		})
	}

	desiredObj.SetResourceVersion(existingObj.GetResourceVersion())

	// we are replacing the whole object instead of using server-side apply which is in beta
	// the object is set to exactly desired object
	return client.Update(ctx, desiredObj, metav1.UpdateOptions{
		FieldManager: fieldManager,
	})
}
