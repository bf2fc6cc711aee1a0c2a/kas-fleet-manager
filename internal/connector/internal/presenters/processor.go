package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentProcessor(from *dbapi.Processor) (public.Processor, *errors.ServiceError) {
	spec := map[string]interface{}{}
	err := from.ProcessorSpec.Unmarshal(&spec)
	if err != nil {
		return public.Processor{}, errors.BadRequest("invalid connector spec: %v", err)
	}

	namespaceId := ""
	if from.NamespaceId != nil {
		namespaceId = *from.NamespaceId
	}

	reference := PresentReference(from.ID, from)
	return public.Processor{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		Owner:           from.Owner,
		Name:            from.Name,
		CreatedAt:       from.CreatedAt,
		ModifiedAt:      from.UpdatedAt,
		ResourceVersion: from.Version,
		NamespaceId:     namespaceId,
		Processor:       spec,
		Annotations:     PresentProcessorAnnotations(from.Annotations),
		Status: public.ProcessorStatusStatus{
			State: public.ProcessorState(from.Status.Phase),
		},
		DesiredState: public.ProcessorDesiredState(from.DesiredState),
		ServiceAccount: public.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
	}, nil
}

func PresentProcessorWithError(from *dbapi.ProcessorWithConditions) (public.Processor, *errors.ServiceError) {
	processor, err := PresentProcessor(&from.Processor)
	if err != nil {
		return processor, err
	}

	var conditions []private.MetaV1Condition
	if from.Conditions != nil {
		err := json.Unmarshal(from.Conditions, &conditions)
		if err != nil {
			return public.Processor{}, errors.GeneralError("invalid conditions: %v", err)
		}
		processor.Status.Error = getStatusError(conditions)
	}

	return processor, nil
}

func PresentProcessorAnnotations(annotations []dbapi.ProcessorAnnotation) map[string]string {
	res := make(map[string]string, len(annotations))
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}
func ConvertProcessorAnnotations(id string, annotations map[string]string) []dbapi.ProcessorAnnotation {
	res := make([]dbapi.ProcessorAnnotation, len(annotations))
	i := 0
	for k, v := range annotations {
		res[i].ProcessorID = id
		res[i].Key = k
		res[i].Value = v

		i++
	}

	return res
}
