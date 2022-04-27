package mocks

import (
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
)

func BuildApiRegionCapacityListItemSlice(modifyFn func(regionCapacityListItems []api.RegionCapacityListItem)) []api.RegionCapacityListItem {
	apiRegionCapacityListItem := []api.RegionCapacityListItem{
		{
			InstanceType:                 types.STANDARD.String(),
			DeprecatedMaxCapacityReached: false,
			AvailableSizes:               []string{},
		},
	}
	if modifyFn != nil {
		modifyFn(apiRegionCapacityListItem)
	}
	return apiRegionCapacityListItem
}

func BuildRegionCapacityListItemSlice(modifyFn func(regionCapacityListItems []public.RegionCapacityListItem)) []public.RegionCapacityListItem {
	regionCapacityListItem := []public.RegionCapacityListItem{
		{
			InstanceType:                 types.STANDARD.String(),
			DeprecatedMaxCapacityReached: false,
			AvailableSizes:               []string{},
		},
	}
	if modifyFn != nil {
		modifyFn(regionCapacityListItem)
	}
	return regionCapacityListItem
}

func BuildApiCloudProvider(modifyFn func(apiCloudProvider *api.CloudProvider)) *api.CloudProvider {
	apiCloudProvider := api.CloudProvider{
		Kind:        mocks.MockCloudProvider.Kind(),
		Id:          mocks.MockCloudProvider.ID(),
		DisplayName: mocks.MockCloudRegionDisplayName,
		Name:        mocks.MockCloudRegionDisplayName,
		Enabled:     true,
	}

	if modifyFn != nil {
		modifyFn(&apiCloudProvider)
	}
	return &apiCloudProvider
}

func BuildCloudProvider(modifyFn func(cloudProvider public.CloudProvider)) public.CloudProvider {
	cloudProvider := public.CloudProvider{
		Kind:        mocks.MockCloudProvider.Kind(),
		Id:          mocks.MockCloudProvider.ID(),
		DisplayName: mocks.MockCloudRegionDisplayName,
		Name:        mocks.MockCloudRegionDisplayName,
		Enabled:     true,
	}
	if modifyFn != nil {
		modifyFn(cloudProvider)
	}
	return cloudProvider
}

func BuildApiCloudRegion(modifyFn func(apiCloudRegion *api.CloudRegion)) *api.CloudRegion {
	apiCloudRegion := api.CloudRegion{
		Id:            mocks.MockCloudProviderRegion.ID(),
		Kind:          mocks.MockCloudProviderRegion.Kind(),
		DisplayName:   mocks.MockCloudProviderRegion.DisplayName(),
		CloudProvider: mocks.MockCloudProvider.DisplayName(),
		Enabled:       true,
	}
	if modifyFn != nil {
		modifyFn(&apiCloudRegion)
	}
	return &apiCloudRegion
}

func BuildCloudRegion(modifyFn func(apiCloudRegion *public.CloudRegion)) *public.CloudRegion {
	cloudRegion := public.CloudRegion{
		Id:          mocks.MockCloudProviderRegion.ID(),
		Kind:        mocks.MockCloudProviderRegion.Kind(),
		DisplayName: mocks.MockCloudProviderRegion.DisplayName(),
		Enabled:     true,
	}
	if modifyFn != nil {
		modifyFn(&cloudRegion)
	}
	return &cloudRegion
}

func GetAllSupportedInstancetypes() []string {
	return strings.Split(api.AllInstanceTypeSupport.String(), ",")
}
