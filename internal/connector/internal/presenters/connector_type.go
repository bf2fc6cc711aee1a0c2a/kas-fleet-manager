package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertConnectorType(from public.ConnectorType) (*dbapi.ConnectorType, error) {

	ct := &dbapi.ConnectorType{
		Meta: api.Meta{
			ID: from.Id,
		},
		Name:        from.Name,
		Version:     from.Version,
		Description: from.Description,
		IconHref:    from.IconHref,
	}

	ct.SetLabels(from.Labels)
	ct.SetChannels(from.Channels)
	if err := ct.SetSchema(from.JsonSchema); err != nil {
		return nil, err
	}
	return ct, nil
}

func PresentConnectorType(from *dbapi.ConnectorType) (*public.ConnectorType, error) {
	schemaDom, err := from.JsonSchemaAsMap()
	if err != nil {
		return nil, err
	}
	reference := PresentReference(from.ID, from)
	return &public.ConnectorType{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		Name:        from.Name,
		Version:     from.Version,
		Description: from.Description,
		JsonSchema:  schemaDom,
		IconHref:    from.IconHref,
		Labels:      from.LabelNames(),
		Channels:    from.ChannelNames(),
	}, nil
}
