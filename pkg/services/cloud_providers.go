package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/patrickmn/go-cache"
	"time"
)

const keyCloudProvidersWithRegions = "cloudProviderWithRegions"

//go:generate moq -out cloud_providers_moq.go . CloudProvidersService
type CloudProvidersService interface {
	GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	ListCloudProviders() ([]api.CloudProvider, *errors.ServiceError)
	ListCloudProviderRegions(id string) ([]api.CloudRegion, *errors.ServiceError)
}

func NewCloudProvidersService(ocmClient ocm.Client) CloudProvidersService {
	return &cloudProvidersService{
		ocmClient: ocmClient,
		cache:     cache.New(5*time.Minute, 10*time.Minute),
	}
}

type cloudProvidersService struct {
	ocmClient ocm.Client
	cache     *cache.Cache
}

type CloudProviderWithRegions struct {
	ID         string
	RegionList *clustersmgmtv1.CloudRegionList
}

func (p cloudProvidersService) GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	cloudProviderWithRegions := []CloudProviderWithRegions{}
	var regionErr error
	providerList, err := p.ocmClient.GetCloudProviders()
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider list")
	}
	providerList.Each(func(provider *clustersmgmtv1.CloudProvider) bool {
		var regions *clustersmgmtv1.CloudRegionList
		regions, regionErr = p.ocmClient.GetRegions(provider)
		if regionErr != nil {
			return false
		}

		cloudProviderWithRegions = append(cloudProviderWithRegions, CloudProviderWithRegions{
			ID:         provider.ID(),
			RegionList: regions,
		})

		return true
	})

	if regionErr != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, regionErr, "failed to retrieve cloud provider list")
	}

	return cloudProviderWithRegions, nil
}

func (p cloudProvidersService) GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	cachedCloudProviderWithRegions, cached := p.cache.Get(keyCloudProvidersWithRegions)
	if cached {
		return convertToCloudProviderWithRegionsType(cachedCloudProviderWithRegions)
	}
	cloudProviderWithRegions, err := p.GetCloudProvidersWithRegions()
	if err != nil {
		return nil, err
	}
	p.cache.Set(keyCloudProvidersWithRegions, cloudProviderWithRegions, cache.DefaultExpiration)
	return cloudProviderWithRegions, nil
}

func convertToCloudProviderWithRegionsType(cachedCloudProviderWithRegions interface{}) ([]CloudProviderWithRegions, *errors.ServiceError) {
	cloudProviderWithRegions, ok := cachedCloudProviderWithRegions.([]CloudProviderWithRegions)
	if ok {
		return cloudProviderWithRegions, nil
	}
	return nil, nil
}

func (p cloudProvidersService) ListCloudProviders() ([]api.CloudProvider, *errors.ServiceError) {

	cloudProviderList := []api.CloudProvider{}
	providerList, err := p.ocmClient.GetCloudProviders()
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider list")
	}

	providerList.Each(func(cloudProvider *clustersmgmtv1.CloudProvider) bool {

		cloudProviderList = append(cloudProviderList, api.CloudProvider{
			Id:          cloudProvider.ID(),
			Name:        cloudProvider.Name(),
			DisplayName: setDisplayName(cloudProvider.ID(), cloudProvider.DisplayName()),
		})

		return true
	})

	return cloudProviderList, nil
}

func (p cloudProvidersService) ListCloudProviderRegions(id string) ([]api.CloudRegion, *errors.ServiceError) {

	cloudRegionList := []api.CloudRegion{}
	cloudProviders, err := p.GetCloudProvidersWithRegions()
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider regions")
	}

	for _, cloudProvider := range cloudProviders {
		if cloudProvider.ID == id {
			cloudProvider.RegionList.Each(func(region *clustersmgmtv1.CloudRegion) bool {

				cloudRegionList = append(cloudRegionList, api.CloudRegion{
					Id:            region.ID(),
					CloudProvider: cloudProvider.ID,
					DisplayName:   region.DisplayName(),
				})

				return true
			})
		}
	}

	return cloudRegionList, nil
}

func setDisplayName(providerId string, defaultDisplayName string) string {

	var displayName string
	switch providerId {
	case "aws":
		displayName = "Amazon Web Services"
	case "azure":
		displayName = "Microsoft Azure"
	case "gcp":
		displayName = "Google Cloud Platform"
	default:
		displayName = defaultDisplayName
	}

	return displayName
}
