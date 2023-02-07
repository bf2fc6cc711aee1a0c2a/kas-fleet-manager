package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/pkg/errors"
)

//go:generate moq -out provider_moq.go . Provider
type Provider interface {
	// Create using the information provided to request a new OpenShift/k8s cluster from the provider
	Create(request *types.ClusterRequest) (*types.ClusterSpec, error)
	// Delete delete the cluster from the provider
	Delete(spec *types.ClusterSpec) (bool, error)
	// CheckClusterStatus check the status of the cluster. This will be called periodically during cluster provisioning phase to see if the cluster is ready.
	// It should set the status in the returned `ClusterSpec` to either `provisioning`, `ready` or `failed`.
	// If there is additional data that needs to be preserved and passed between checks, add it to the returned `ClusterSpec` and it will be saved to the database and passed into this function again next time it is called.
	CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error)
	// AddIdentityProvider add an identity provider to the cluster
	AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error)
	// ApplyResources apply openshift/k8s resources to the cluster
	ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error)
	// RemoveResources uninstalls resources from a cluster
	RemoveResources(clusterSpec *types.ClusterSpec, syncSetName string) error
	// GetClusterDNS Get the dns of the cluster
	GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error)
	// GetClusterSpec returns the details of the cluster from the cluster provider
	// It should set the status in the returned `ClusterSpec` to either `provisioning`, `ready` or `failed`.
	GetClusterSpec(clusterID string) (types.ClusterSpec, error)
	// GetCloudProviders Get the information about supported cloud providers from the cluster provider
	GetCloudProviders() (*types.CloudProviderInfoList, error)
	// GetCloudProviderRegions Get the regions information for the given cloud provider from the cluster provider
	GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error)
	// Install the strimzi operator in a given cluster
	InstallStrimzi(clusterSpec *types.ClusterSpec) (bool, error)
	// Install the cluster logging operator for a given cluster
	InstallClusterLogging(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error)
	// Install the cluster logging operator for a given cluster
	InstallKasFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error)
	GetMachinePool(clusterID string, id string) (*types.MachinePoolInfo, error)
	CreateMachinePool(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error)
	// GetClusterResourceQuotaCosts returns a list of quota cost information related to resources used for the provisioning and
	// terraforming of data plane clusters for the authenticated user.
	GetClusterResourceQuotaCosts() ([]types.QuotaCost, error)
}

// ProviderFactory used to return an instance of Provider implementation
//
//go:generate moq -out provider_factory_moq.go . ProviderFactory
type ProviderFactory interface {
	GetProvider(providerType api.ClusterProviderType) (Provider, error)
}

// DefaultProviderFactory the default implementation for ProviderFactory
type DefaultProviderFactory struct {
	providerContainer map[api.ClusterProviderType]Provider
}

func NewDefaultProviderFactory(
	ocmClient ocm.ClusterManagementClient,
	connectionFactory *db.ConnectionFactory,
	ocmConfig *ocm.OCMConfig,
	awsConfig *config.AWSConfig,
	gcpConfig *config.GCPConfig,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
) *DefaultProviderFactory {

	clusterBuilder := NewClusterBuilder(awsConfig, gcpConfig, dataplaneClusterConfig)
	ocmProvider := newOCMProvider(ocmClient, clusterBuilder, ocmConfig)
	standaloneProvider := newStandaloneProvider(connectionFactory, dataplaneClusterConfig)
	return &DefaultProviderFactory{
		providerContainer: map[api.ClusterProviderType]Provider{
			api.ClusterProviderStandalone: standaloneProvider,
			api.ClusterProviderOCM:        ocmProvider,
		},
	}

}

func (d *DefaultProviderFactory) GetProvider(providerType api.ClusterProviderType) (Provider, error) {
	if providerType == "" {
		providerType = api.ClusterProviderOCM
	}

	provider, ok := d.providerContainer[providerType]
	if !ok {
		return nil, errors.Errorf("invalid provider type: %v", providerType)
	}

	return provider, nil
}
