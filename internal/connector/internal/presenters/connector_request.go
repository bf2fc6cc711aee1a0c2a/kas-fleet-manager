package presenters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertConnectorRequest(from public.ConnectorRequest) (*dbapi.Connector, *errors.ServiceError) {

	spec, err := json.Marshal(from.Connector)
	if err != nil {
		return nil, errors.BadRequest("invalid connector spec: %v", err)
	}

	namespaceId := &from.DeploymentLocation.NamespaceId
	if *namespaceId == "" {
		namespaceId = nil
	}
	return &dbapi.Connector{
		TargetKind:      dbapi.AddonTargetKind,
		NamespaceId:     namespaceId,
		Name:            from.Name,
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
	}, nil
}
