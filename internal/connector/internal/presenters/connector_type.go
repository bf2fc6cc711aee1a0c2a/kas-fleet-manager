package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertConnectorType(from public.ConnectorType) *dbapi.ConnectorType {

	return &dbapi.ConnectorType{
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

func PresentConnectorType(from *dbapi.ConnectorType) public.ConnectorType {
	reference := PresentReference(from.ID, from)
	return public.ConnectorType{
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
