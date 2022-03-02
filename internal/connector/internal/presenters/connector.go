package presenters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertConnector(from public.Connector) (*dbapi.Connector, *errors.ServiceError) {

	spec, err := json.Marshal(from.Connector)
	if err != nil {
		return nil, errors.BadRequest("invalid connector spec: %v", err)
	}

	var namespaceId *string
	if from.NamespaceId != "" {
		namespaceId = &from.NamespaceId
	}
	return &dbapi.Connector{
		Meta: api.Meta{
			ID: from.Id,
		},
		NamespaceId:     namespaceId,
		Name:            from.Name,
		Owner:           from.Owner,
		Version:         from.ResourceVersion,
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		DesiredState:    string(from.DesiredState),
		Channel:         string(from.Channel),
		Kafka: dbapi.KafkaConnectionSettings{
			KafkaID:         from.Kafka.Id,
			BootstrapServer: from.Kafka.Url,
		},
		SchemaRegistry: dbapi.SchemaRegistryConnectionSettings{
			SchemaRegistryID: from.SchemaRegistry.Id,
			Url:              from.SchemaRegistry.Url,
		},
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
		Status: dbapi.ConnectorStatus{
			Phase: string(from.Status.State),
		},
	}, nil
}

func PresentConnector(from *dbapi.Connector) (public.Connector, *errors.ServiceError) {
	spec := map[string]interface{}{}
	err := json.Unmarshal([]byte(from.ConnectorSpec), &spec)
	if err != nil {
		return public.Connector{}, errors.BadRequest("invalid connector spec: %v", err)
	}

	namespaceId := ""
	if from.NamespaceId != nil {
		namespaceId = *from.NamespaceId
	}

	reference := PresentReference(from.ID, from)
	return public.Connector{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		Owner:           from.Owner,
		Name:            from.Name,
		CreatedAt:       from.CreatedAt,
		ModifiedAt:      from.UpdatedAt,
		ResourceVersion: from.Version,
		NamespaceId:     namespaceId,
		ConnectorTypeId: from.ConnectorTypeId,
		Connector:       spec,
		Status: public.ConnectorStatusStatus{
			State: public.ConnectorState(from.Status.Phase),
		},
		DesiredState: public.ConnectorDesiredState(from.DesiredState),
		Channel:      public.Channel(from.Channel),
		Kafka: public.KafkaConnectionSettings{
			Id:  from.Kafka.KafkaID,
			Url: from.Kafka.BootstrapServer,
		},
		SchemaRegistry: public.SchemaRegistryConnectionSettings{
			Id:  from.SchemaRegistry.SchemaRegistryID,
			Url: from.SchemaRegistry.Url,
		},
		ServiceAccount: public.ServiceAccount{
			ClientId:     from.ServiceAccount.ClientId,
			ClientSecret: from.ServiceAccount.ClientSecret,
		},
	}, nil
}
