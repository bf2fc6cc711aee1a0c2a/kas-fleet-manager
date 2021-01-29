package presenters

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertConnector(from openapi.Connector) (*api.Connector, *errors.ServiceError) {

	spec, err := json.Marshal(from.ConnectorSpec)
	if err != nil {
		return nil, errors.BadRequest("invalid connector spec: %v", err)
	}

	return &api.Connector{
		Meta: api.Meta{
			ID: from.Id,
		},
		Owner:           from.Metadata.Owner,
		KafkaID:         from.Metadata.KafkaId,
		Name:            from.Metadata.Name,
		Region:          from.DeploymentLocation.Region,
		CloudProvider:   from.DeploymentLocation.CloudProvider,
		MultiAZ:         from.DeploymentLocation.MultiAz,
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   string(spec),
		Status:          from.Status,
	}, nil
}

func PresentConnector(from *api.Connector) (openapi.Connector, *errors.ServiceError) {
	spec := map[string]interface{}{}
	err := json.Unmarshal([]byte(from.ConnectorSpec), &spec)
	if err != nil {
		return openapi.Connector{}, errors.BadRequest("invalid connector spec: %v", err)
	}

	reference := PresentReference(from.ID, from)
	return openapi.Connector{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorAllOfMetadata{
			Owner:     from.Owner,
			KafkaId:   from.KafkaID,
			Name:      from.Name,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		DeploymentLocation: openapi.ConnectorAllOfDeploymentLocation{
			CloudProvider: from.CloudProvider,
			MultiAz:       from.MultiAZ,
			Region:        from.Region,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		Status:          from.Status,
	}, nil
}
