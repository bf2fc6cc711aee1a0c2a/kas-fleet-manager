package services

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

const (
	AWSCloudProviderID = "aws"
)

type CloudProvidersService interface {
	GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, error)
}

func NewCloudProvidersService(ocmClient ocm.Client) CloudProvidersService {
	return &cloudProvidersService{
		ocmClient: ocmClient,
	}
}

type cloudProvidersService struct {
	ocmClient ocm.Client
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
		// TODO add "|| provider.ID() == GcpCloudProviderID" to suport GCP in the future
		if provider.ID() == AWSCloudProviderID {
			var regions *clustersmgmtv1.CloudRegionList
			regions, regionErr = p.ocmClient.GetRegions(provider)
			if regionErr != nil {
				return false
			}

			cloudProviderWithRegions = append(cloudProviderWithRegions, CloudProviderWithRegions{
				ID:         provider.ID(),
				RegionList: regions,
			})
		}

		return true
	})

	return cloudProviderWithRegions, regionErr
}
