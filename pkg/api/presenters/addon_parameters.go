package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
)

func PresentAddonParameter(from ocm.AddonParameter) openapi.AddonParameter {
	return openapi.AddonParameter{
		Id:    from.Id,
		Value: from.Value,
	}
}
