package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func ConvertConnector(from public.Connector) (*dbapi.Connector, *errors.ServiceError) {

	spec, err := json.Marshal(from.ConnectorSpec)
	if err != nil {
		return nil, errors.BadRequest("invalid connector spec: %v", err)
	}

	return &dbapi.Connector{
		Meta: api.Meta{
			ID: from.Id,
		},
		TargetKind:      from.DeploymentLocation.Kind,
		AddonClusterId:  from.DeploymentLocation.ClusterId,
		CloudProvider:   from.DeploymentLocation.CloudProvider,
		Region:          from.DeploymentLocation.Region,
		MultiAZ:         from.DeploymentLocation.MultiAz,
		Name:            from.Metadata.Name,
		Owner:           from.Metadata.Owner,
		DinosaurID:         from.Metadata.DinosaurId,
		Version:         from.Metadata.ResourceVersion,
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		DesiredState:    from.DesiredState,
		Channel:         from.Channel,
		Dinosaur: dbapi.DinosaurConnectionSettings{
			BootstrapServer: from.Dinosaur.BootstrapServer,
			ClientId:        from.Dinosaur.ClientId,
			ClientSecret:    from.Dinosaur.ClientSecret,
		},
		Status: dbapi.ConnectorStatus{
			Phase: from.Status,
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
		Metadata: public.ConnectorAllOfMetadata{
			Owner:           from.Owner,
			DinosaurId:         from.DinosaurID,
			Name:            from.Name,
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		DeploymentLocation: public.ClusterTarget{
			Kind:          from.TargetKind,
			ClusterId:     from.AddonClusterId,
			CloudProvider: from.CloudProvider,
			Region:        from.Region,
			MultiAz:       from.MultiAZ,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		Status:          from.Status.Phase,
		DesiredState:    from.DesiredState,
		Channel:         from.Channel,
		Dinosaur: public.DinosaurConnectionSettings{
			BootstrapServer: from.Dinosaur.BootstrapServer,
			ClientId:        from.Dinosaur.ClientId,
			ClientSecret:    from.Dinosaur.ClientSecret,
		},
	}, nil
}
