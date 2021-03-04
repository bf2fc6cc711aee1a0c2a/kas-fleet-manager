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
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   string(spec),
		TargetKind:      from.DeploymentLocation.Kind,
		AddonGroup:      from.DeploymentLocation.Group,
		Region:          from.DeploymentLocation.Region,
		CloudProvider:   from.DeploymentLocation.CloudProvider,
		MultiAZ:         from.DeploymentLocation.MultiAz,
		Name:            from.Metadata.Name,
		Status:          from.Status,
		Owner:           from.Metadata.Owner,
		KafkaID:         from.Metadata.KafkaId,
		Version:         from.Metadata.ResourceVersion,
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
			Owner:           from.Owner,
			KafkaId:         from.KafkaID,
			Name:            from.Name,
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		DeploymentLocation: openapi.ClusterTarget{
			Kind:          from.TargetKind,
			Group:         from.AddonGroup,
			CloudProvider: from.CloudProvider,
			Region:        from.Region,
			MultiAz:       from.MultiAZ,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		Status:          from.Status,
	}, nil
}
