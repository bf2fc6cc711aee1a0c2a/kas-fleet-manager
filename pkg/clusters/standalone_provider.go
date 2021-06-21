package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

type StandaloneProvider struct {
	connectionFactory *db.ConnectionFactory
}

func (s *StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, nil
}

func (s *StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	return true, nil
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

func (s *StandaloneProvider) InstallAddon(clusterSpec *types.ClusterSpec, addonID string) (bool, error) {
	return false, nil
}

func (s *StandaloneProvider) InstallAddonWithParams(clusterSpec *types.ClusterSpec, addonId string, addonParams []ocm.AddonParameter) (bool, error) {
	return false, nil
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider(connectionFactory *db.ConnectionFactory) *StandaloneProvider {
	return &StandaloneProvider{connectionFactory: connectionFactory}
}
