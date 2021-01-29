package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func ConvertConnectorType(from openapi.ConnectorType) *api.ConnectorType {

	return &api.ConnectorType{
		Meta: api.Meta{
			ID: from.Id,
		},
		Name:        from.Name,
		Version:     from.Version,
		Description: from.Description,
		JsonSchema:  from.JsonSchema,
	}
}

func PresentConnectorType(from *api.ConnectorType) openapi.ConnectorType {
	reference := PresentReference(from.ID, from)
	return openapi.ConnectorType{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		Name:        from.Name,
		Version:     from.Version,
		Description: from.Description,
		JsonSchema:  from.JsonSchema,
	}
}
