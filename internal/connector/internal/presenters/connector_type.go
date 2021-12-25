package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func toStringSlice(channels []public.Channel) []string {
	if (channels == nil) {
		return nil
	}

	strings := make([]string, 0, len(channels))
	for _, channel := range channels {
		strings = append(strings, string(channel))
	}
	return strings
}

func toChannelSlice(strings []string) []public.Channel {
	if (strings == nil) {
		return nil
	}

	channels := make([]public.Channel, 0, len(strings))
	for _, str := range strings {
		channels = append(channels, public.Channel(str))
	}
	return channels
}

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
	ct.SetChannels(toStringSlice(from.Channels))
	schemaToBeSet := from.Schema
	if schemaToBeSet == nil {
		schemaToBeSet = from.Schema
	}
	if err := ct.SetSchema(schemaToBeSet); err != nil {
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
		Schema:      schemaDom,
		IconHref:    from.IconHref,
		Labels:      from.LabelNames(),
		Channels:    toChannelSlice(from.ChannelNames()),
	}, nil
}
