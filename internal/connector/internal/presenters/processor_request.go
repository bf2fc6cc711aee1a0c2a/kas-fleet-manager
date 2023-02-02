package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertProcessorRequest(id string, from public.ProcessorRequest) (*dbapi.Processor, *errors.ServiceError) {

	spec, err := json.Marshal(from.Processor)
	if err != nil {
		return nil, errors.BadRequest("invalid processor spec: %v", err)
	}

	namespaceId := &from.NamespaceId
	if *namespaceId == "" {
		namespaceId = nil
	}
	return &dbapi.Processor{
		Model: db.Model{
			ID: id,
		},
		NamespaceId:   namespaceId,
		Name:          from.Name,
		ProcessorSpec: spec,
		DesiredState:  dbapi.ProcessorDesiredState(from.DesiredState),
		Annotations:   ConvertProcessorAnnotations(id, from.Annotations),
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
	}, nil
}
