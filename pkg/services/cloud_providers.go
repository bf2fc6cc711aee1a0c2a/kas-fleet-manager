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
	GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, error)
	GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, error)
	ListCloudProviders() ([]api.CloudProvider, *errors.ServiceError)
	ListCloudProviderRegions(id string) ([]api.CloudRegion, *errors.ServiceError)
}

func NewCloudProvidersService(ocmClient ocm.Client) CloudProvidersService {
	return &cloudProvidersService{
		ocmClient: ocmClient,
		cache:     cache.New(5*time.Hour, 10*time.Hour),
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

func (p cloudProvidersService) GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, error) {
	cloudProviderWithRegions := []CloudProviderWithRegions{}
	var regionErr error
	providerList, err := p.ocmClient.GetCloudProviders()
	if err != nil {
		return nil, err
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
	return cloudProviderWithRegions, regionErr
}

func (p cloudProvidersService) GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, error) {
	cloudProviderWithRegions, cached := p.cache.Get(keyCloudProvidersWithRegions)
	if cached {
		return convertToCloudProviderType(cloudProviderWithRegions)
	}
	cloudProviderWithRegions, err := p.GetCloudProvidersWithRegions()
	if err != nil {
		return nil, err
	}
	p.cache.Set(keyCloudProvidersWithRegions, cloudProviderWithRegions, cache.DefaultExpiration)
	return convertToCloudProviderType(cloudProviderWithRegions)
}

func convertToCloudProviderType(cachedCloudProviderWithRegions interface{}) ([]CloudProviderWithRegions, error) {
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
		return nil, errors.GeneralError("error retrieving cloud provider list: %v", err)
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
		return nil, errors.GeneralError("error retrieving cloud provider list: %v", err)
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
