package presenters

import (
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func toStringSlice(channels []public.Channel) []string {
	if channels == nil {
		return nil
	}

	strings := make([]string, 0, len(channels))
	for _, channel := range channels {
		strings = append(strings, string(channel))
	}
	return strings
}

func toChannelSlice(strings []string) []public.Channel {
	if strings == nil {
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
		Model: db.Model{
			ID: from.Id,
		},
		Name:         from.Name,
		Version:      from.Version,
		Deprecated:   from.Deprecated,
		Description:  from.Description,
		FeaturedRank: from.FeaturedRank,
		IconHref:     from.IconHref,
		Annotations:  ConvertTypeAnnotations(from.Id, from.Annotations),
	}

	ct.SetLabels(from.Labels)
	ct.SetChannels(toStringSlice(from.Channels))
	ct.SetCapabilities(from.Capabilities)
	schemaToBeSet := from.Schema
	if schemaToBeSet == nil {
		schemaToBeSet = from.Schema
	}
	if err := ct.SetSchema(schemaToBeSet); err != nil {
		return nil, err
	}
	return ct, nil
}

func ConvertTypeAnnotations(id string, annotations map[string]string) []dbapi.ConnectorTypeAnnotation {
	res := make([]dbapi.ConnectorTypeAnnotation, len(annotations))
	i := 0
	for k, v := range annotations {
		res[i].ConnectorTypeID = id
		res[i].Key = k
		res[i].Value = v

		i++
	}

	return res
}

func PresentConnectorType(from *dbapi.ConnectorType) (*public.ConnectorType, error) {
	schemaDom, err := from.JsonSchemaAsMap()
	if err != nil {
		return nil, err
	}
	reference := PresentReference(from.ID, from)
	return &public.ConnectorType{
		Id:           reference.Id,
		Kind:         reference.Kind,
		Href:         reference.Href,
		Name:         from.Name,
		Version:      from.Version,
		Deprecated:   from.Deprecated,
		Description:  from.Description,
		FeaturedRank: from.FeaturedRank,
		Schema:       schemaDom,
		IconHref:     from.IconHref,
		Labels:       from.LabelNames(),
		Channels:     toChannelSlice(from.ChannelNames()),
		Capabilities: from.CapabilitiesNames(),
		Annotations:  PresentTypeAnnotations(from.Annotations),
	}, nil
}

func PresentTypeAnnotations(annotations []dbapi.ConnectorTypeAnnotation) map[string]string {
	res := make(map[string]string, len(annotations))
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}

func PresentConnectorTypeLabelCount(from *dbapi.ConnectorTypeLabelCount) (*public.ConnectorTypeLabelCount, error) {
	return &public.ConnectorTypeLabelCount{
		Label: from.Label,
		Count: from.Count,
	}, nil
}

func PresentConnectorTypeAdminView(from dbapi.ConnectorCatalogEntry) (*admin.ConnectorTypeAdminView, *errors.ServiceError) {
	view := admin.ConnectorTypeAdminView{
		Id:           from.ConnectorType.ID,
		Name:         from.ConnectorType.Name,
		Version:      from.ConnectorType.Version,
		Deprecated:   from.ConnectorType.Deprecated,
		Description:  from.ConnectorType.Description,
		FeaturedRank: from.ConnectorType.FeaturedRank,
		IconHref:     from.ConnectorType.IconHref,
		Labels:       make([]string, len(from.ConnectorType.Labels)),
		Annotations:  PresentTypeAnnotations(from.ConnectorType.Annotations),
		Channels:     make(map[string]admin.ConnectorTypeChannel),
	}

	for i, l := range from.ConnectorType.Labels {
		view.Labels[i] = l.Label
	}

	for name, channel := range from.Channels {
		meta, err := channel.ShardMetadata.Object()
		if err != nil {
			return nil, errors.ToServiceError(err)
		}

		view.Channels[name] = admin.ConnectorTypeChannel{
			ShardMetadata: meta,
		}
	}

	schema, err := from.ConnectorType.JsonSchema.Object()
	if err != nil {
		return nil, errors.ToServiceError(err)
	}

	view.Schema = schema

	reference := PresentReference(view.Id, view)
	view.Kind = reference.Kind
	view.Href = reference.Href

	return &view, nil
}
