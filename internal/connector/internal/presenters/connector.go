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

	return &dbapi.Connector{
		Meta: api.Meta{
			ID: from.Id,
		},
		TargetKind:      from.DeploymentLocation.Kind,
		AddonClusterId:  from.DeploymentLocation.ClusterId,
		Name:            from.Name,
		Owner:           from.Owner,
		KafkaID:         from.Kafka.Id,
		Version:         from.ResourceVersion,
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		DesiredState:    string(from.DesiredState),
		Channel: 		 string(from.Channel),
		Kafka: dbapi.KafkaConnectionSettings{
			BootstrapServer: from.Kafka.Url,
			ClientId:        from.ServiceAccount.ClientId,
			ClientSecret:    from.ServiceAccount.ClientSecret,
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

		DeploymentLocation: public.DeploymentLocation{
			Kind:          from.TargetKind,
			ClusterId:     from.AddonClusterId,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		Connector:   	 spec,
		Status: public.ConnectorStatusStatus{
			State: public.ConnectorState(from.Status.Phase),
		},
		DesiredState: public.ConnectorDesiredState(from.DesiredState),
		Channel: public.Channel(from.Channel),
		Kafka: public.KafkaConnectionSettings{
			Id: from.KafkaID,
			Url: from.Kafka.BootstrapServer,
		},
		ServiceAccount: public.ServiceAccount{
			ClientId:        from.Kafka.ClientId,
			ClientSecret:    from.Kafka.ClientSecret,
		},
	}, nil
}
