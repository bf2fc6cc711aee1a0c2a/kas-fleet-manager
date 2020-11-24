package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
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
