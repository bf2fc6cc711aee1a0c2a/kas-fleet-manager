package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/pkg/errors"
)

type StandaloneProvider struct {
}

func (s StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, errors.Errorf("Create is not supported for StandaloneProvider type")
}

func (s StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	panic("implement me")
}

func (s StandaloneProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	panic("implement me")
}

func (s StandaloneProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	panic("implement me")
}

func (s StandaloneProvider) ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
	panic("implement me")
}

func (s StandaloneProvider) ScaleUp(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) ScaleDown(clusterSpec *types.ClusterSpec, decrement int) (*types.ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) SetComputeNodes(clusterSpec *types.ClusterSpec, numNodes int) (*types.ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) GetComputeNodes(spec *types.ClusterSpec) (*types.ComputeNodesInfo, error) {
	panic("implement me")
}

func (s StandaloneProvider) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	panic("implement me")
}

func (s StandaloneProvider) GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	panic("implement me")
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider() *StandaloneProvider {
	return &StandaloneProvider{}
}
