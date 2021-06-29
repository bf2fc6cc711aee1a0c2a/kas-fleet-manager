package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/presenters"
)

func PresentCloudProvider(cloudProvider *api.CloudProvider) public.CloudProvider {

	reference := presenters.PresentReference(cloudProvider.Id, cloudProvider)
	return public.CloudProvider{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Name:        cloudProvider.Name,
		DisplayName: cloudProvider.DisplayName,
		Enabled:     cloudProvider.Enabled,
	}
}
func PresentCloudRegion(cloudRegion *api.CloudRegion) public.CloudRegion {
	reference := presenters.PresentReference(cloudRegion.Id, cloudRegion)
	return public.CloudRegion{
		Id:          reference.Id,
		Kind:        reference.Kind,
		DisplayName: cloudRegion.DisplayName,
		Enabled:     cloudRegion.Enabled,
	}
}
