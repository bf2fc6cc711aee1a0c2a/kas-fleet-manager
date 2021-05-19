package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
)

type OCMProvider struct {
	ocmClient ocm.Client
	clusterBuilder ocm.ClusterBuilder
}

func (o *OCMProvider) Create(request *ClusterRequest) (*ClusterSpec, error) {
	// Build a new OSD cluster object
	newCluster, err := o.clusterBuilder.NewOCMClusterFromCluster(request)
	if err != nil {
		return nil, err
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := o.ocmClient.CreateCluster(newCluster)
	if err != nil {
		return nil, err
	}
	
	result := &ClusterSpec{
		Status: api.ClusterProvisioning,
	}
	if createdCluster.ID() != "" {
		result.InternalID = createdCluster.ID()
	}
	if createdCluster.ExternalID() != "" {
		result.ExternalID = createdCluster.ExternalID()
	}
	return result, nil
}

func (o *OCMProvider) Get(spec *ClusterSpec) (*ClusterSpec, error) {
	panic("implement me")
}

func (o *OCMProvider) GetClusterDNS(clusterSpec *ClusterSpec) (string, error) {
	panic("implement me")
}

func (O OCMProvider) AddIdentityProvider(clusterSpec *ClusterSpec, identityProvider IdentityProvider) (*IdentityProvider, error) {
	panic("implement me")
}

func (O OCMProvider) ListIdentityProviders(clusterSpec *ClusterSpec) (*IdentityProviderList, error) {
	panic("implement me")
}

func (O OCMProvider) ApplyResources(clusterSpec *ClusterSpec, resources ResourceSet) (*ResourceSet, *error) {
	panic("implement me")
}

func (O OCMProvider) DeleteResources(clusterSpec *ClusterSpec, resources ResourceSet) error {
	panic("implement me")
}

func (O OCMProvider) ScaleUp(clusterSpec *ClusterSpec, increment int) (*ClusterSpec, error) {
	panic("implement me")
}

func (O OCMProvider) ScaleDown(clusterSpec *ClusterSpec, decrement int) (*ClusterSpec, error) {
	panic("implement me")
}

func (O OCMProvider) SetComputeNodes(clusterSpec *ClusterSpec, numNodes int) (*ClusterSpec, error) {
	panic("implement me")
}

func (O OCMProvider) CreateAddon(clusterSpec *ClusterSpec, info AddonInfo) (*ClusterSpec, error) {
	panic("implement me")
}

func (O OCMProvider) UpdateAddon(clusterSpec *ClusterSpec, info AddonInfo) (*ClusterSpec, error) {
	panic("implement me")
}

// ensure OCMProvider implements Provider interface
var _ Provider = &OCMProvider{}

func newOCMProvider(ocmClient ocm.Client, clusterBuilder ocm.ClusterBuilder) *OCMProvider {
	return &OCMProvider{
		ocmClient: ocmClient,
		clusterBuilder: clusterBuilder,
	}
}
