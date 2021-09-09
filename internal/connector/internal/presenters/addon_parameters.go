package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
)

func PresentAddonParameter(from ocm.Parameter) public.AddonParameter {
	return public.AddonParameter{
		Id:    from.Id,
		Value: from.Value,
	}
}
