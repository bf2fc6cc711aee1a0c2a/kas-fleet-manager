package presenters

import (
	"encoding/json"
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
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
		Model: db.Model{
			ID: from.Id,
		},
		NamespaceId:     namespaceId,
		Name:            from.Name,
		Owner:           from.Owner,
		Version:         from.ResourceVersion,
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		DesiredState:    dbapi.ConnectorDesiredState(from.DesiredState),
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
			Phase: dbapi.ConnectorStatusPhase(from.Status.State),
		},
	}, nil
}

func PresentConnectorWithError(from *dbapi.ConnectorWithConditions) (public.Connector, *errors.ServiceError) {
	connector, err := PresentConnector(&from.Connector)
	if err != nil {
		return connector, err
	}

	var conditions []private.MetaV1Condition
	if from.Conditions != nil {
		err := json.Unmarshal(from.Conditions, &conditions)
		if err != nil {
			return public.Connector{}, errors.GeneralError("invalid conditions: %v", err)
		}
		connector.Status.Error = getStatusError(conditions)
	}

	return connector, nil
}

func PresentConnectorAdminView(from *dbapi.ConnectorWithConditions) (admin.ConnectorAdminView, *errors.ServiceError) {

	namespaceId := ""
	if from.NamespaceId != nil {
		namespaceId = *from.NamespaceId
	}

	connector := admin.ConnectorAdminView{
		Id:              from.ID,
		Owner:           from.Owner,
		Name:            from.Name,
		CreatedAt:       from.CreatedAt,
		ModifiedAt:      from.UpdatedAt,
		ResourceVersion: from.Version,
		NamespaceId:     namespaceId,
		ConnectorTypeId: from.ConnectorTypeId,
		Status: admin.ConnectorStatusStatus{
			State: admin.ConnectorState(from.Status.Phase),
		},
		DesiredState: admin.ConnectorDesiredState(from.DesiredState),
		Channel:      admin.Channel(from.Channel),
	}
	reference := PresentReference(connector.Id, connector)
	connector.Kind = reference.Kind
	connector.Href = reference.Href

	var conditions []private.MetaV1Condition
	if from.Conditions != nil {
		err := json.Unmarshal(from.Conditions, &conditions)
		if err != nil {
			return admin.ConnectorAdminView{}, errors.GeneralError("invalid conditions: %v", err)
		}
		connector.Status.Error = getStatusError(conditions)
	}

	return connector, nil
}

func getStatusError(conditions []private.MetaV1Condition) string {
	var result string
	for _, c := range conditions {
		if c.Type == "Ready" {
			if c.Status == "False" {
				finalError := c.Message
				start := strings.Index(c.Message, "error.message")
				end := strings.Index(c.Message, "failure.count")
				if start > -1 && end > -1 {
					finalError = c.Message[start+14 : end]
				}
				result = c.Reason + ": " + finalError
			}
			break
		}
	}
	return result
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
