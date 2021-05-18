package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
	connector_catalog "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector_catalog/openapi"
)

func ConvertConnectorType(from connector_catalog.ConnectorType) *api.ConnectorType {

	return &api.ConnectorType{
		Meta: api.Meta{
			ID: from.Id,
		},
		Name:        from.Name,
		Version:     from.Version,
		Description: from.Description,
		JsonSchema:  from.JsonSchema,
		IconHref:    from.IconHref,
		Labels:      from.Labels,
		Channels:    from.Channels,
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
		IconHref:    from.IconHref,
		Labels:      from.Labels,
		Channels:    from.Channels,
	}
}
