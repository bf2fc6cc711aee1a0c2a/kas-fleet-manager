package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
)

func PresentAddonParameter(from ocm.AddonParameter) public.AddonParameter {
	return public.AddonParameter{
		Id:    from.Id,
		Value: from.Value,
	}
}
