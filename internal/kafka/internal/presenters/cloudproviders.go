package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func PresentCloudProvider(cloudProvider *api.CloudProvider) public.CloudProvider {

	reference := PresentReference(cloudProvider.Id, cloudProvider)
	return public.CloudProvider{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Name:        cloudProvider.Name,
		DisplayName: cloudProvider.DisplayName,
		Enabled:     cloudProvider.Enabled,
	}
}
func PresentCloudRegion(cloudRegion *api.CloudRegion) public.CloudRegion {
	reference := PresentReference(cloudRegion.Id, cloudRegion)
	return public.CloudRegion{
		Id:                               reference.Id,
		Kind:                             reference.Kind,
		DisplayName:                      cloudRegion.DisplayName,
		Enabled:                          cloudRegion.Enabled,
		DeprecatedSupportedInstanceTypes: getSupportedInstanceTypes(cloudRegion.SupportedInstanceTypes),
		Capacity:                         GetRegionCapacityItems(cloudRegion.Capacity),
	}
}

func GetRegionCapacityItems(capacityItems []api.RegionCapacityListItem) []public.RegionCapacityListItem {
	items := make([]public.RegionCapacityListItem, 0)
	for _, c := range capacityItems {
		items = append(items, public.RegionCapacityListItem{InstanceType: c.InstanceType, DeprecatedMaxCapacityReached: c.DeprecatedMaxCapacityReached})
	}
	return items
}

func getSupportedInstanceTypes(instTypes []string) []string {
	if len(instTypes) > 0 {
		return instTypes
	}
	return make([]string, 0)
}
