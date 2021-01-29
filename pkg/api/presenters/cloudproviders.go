package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

func PresentCloudProvider(cloudProvider *api.CloudProvider) openapi.CloudProvider {

	reference := PresentReference(cloudProvider.Id, cloudProvider)
	return openapi.CloudProvider{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Name:        cloudProvider.Name,
		DisplayName: cloudProvider.DisplayName,
		Enabled:     cloudProvider.Enabled,
	}
}
func PresentCloudRegion(cloudRegion *api.CloudRegion) openapi.CloudRegion {
	reference := PresentReference(cloudRegion.Id, cloudRegion)
	return openapi.CloudRegion{
		Id:          reference.Id,
		Kind:        reference.Kind,
		DisplayName: cloudRegion.DisplayName,
		Enabled:     cloudRegion.Enabled,
	}
}
