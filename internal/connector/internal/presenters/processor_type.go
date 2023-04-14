package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

func ConvertProcessorType(from public.ProcessorType) (*dbapi.ProcessorType, error) {

	ct := &dbapi.ProcessorType{
		Model: db.Model{
			ID: from.Id,
		},
		Name:         from.Name,
		Version:      from.Version,
		Deprecated:   from.Deprecated,
		Description:  from.Description,
		FeaturedRank: from.FeaturedRank,
		IconHref:     from.IconHref,
		Annotations:  ConvertProcessorTypeAnnotations(from.Id, from.Annotations),
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

func ConvertProcessorTypeAnnotations(processorTypeId string, annotations map[string]string) []dbapi.ProcessorTypeAnnotation {
	res := make([]dbapi.ProcessorTypeAnnotation, len(annotations))
	i := 0
	for k, v := range annotations {
		res[i].ProcessorTypeID = processorTypeId
		res[i].Key = k
		res[i].Value = v

		i++
	}

	return res
}

func PresentProcessorType(from *dbapi.ProcessorType) (*public.ProcessorType, error) {
	schemaDom, err := from.JsonSchemaAsMap()
	if err != nil {
		return nil, err
	}
	reference := PresentReference(from.ID, from)
	return &public.ProcessorType{
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
		Annotations:  PresentProcessorTypeAnnotations(from.Annotations),
	}, nil
}

func PresentProcessorTypeAnnotations(annotations []dbapi.ProcessorTypeAnnotation) map[string]string {
	res := make(map[string]string, len(annotations))
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}
