package clusters

import "github.com/pkg/errors"

type StandaloneProvider struct {
}

func (s StandaloneProvider) Create(request *ClusterRequest) (*ClusterSpec, error) {
	return nil, errors.Errorf("Create is not supported for StandaloneProvider type")
}

func (s StandaloneProvider) Get(spec *ClusterSpec) (*ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) GetClusterDNS(clusterSpec *ClusterSpec) (string, error) {
	panic("implement me")
}

func (s StandaloneProvider) AddIdentityProvider(clusterSpec *ClusterSpec, identityProvider IdentityProvider) (*IdentityProvider, error) {
	panic("implement me")
}

func (s StandaloneProvider) ListIdentityProviders(clusterSpec *ClusterSpec) (*IdentityProviderList, error) {
	panic("implement me")
}

func (s StandaloneProvider) ApplyResources(clusterSpec *ClusterSpec, resources ResourceSet) (*ResourceSet, *error) {
	panic("implement me")
}

func (s StandaloneProvider) DeleteResources(clusterSpec *ClusterSpec, resources ResourceSet) error {
	panic("implement me")
}

func (s StandaloneProvider) ScaleUp(clusterSpec *ClusterSpec, increment int) (*ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) ScaleDown(clusterSpec *ClusterSpec, decrement int) (*ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) SetComputeNodes(clusterSpec *ClusterSpec, numNodes int) (*ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) CreateAddon(clusterSpec *ClusterSpec, info AddonInfo) (*ClusterSpec, error) {
	panic("implement me")
}

func (s StandaloneProvider) UpdateAddon(clusterSpec *ClusterSpec, info AddonInfo) (*ClusterSpec, error) {
	panic("implement me")
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider() *StandaloneProvider {
	return &StandaloneProvider{}
}
