package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertConnectorClusterInstance(from public.ConnectorClusterInstance) dbapi.ConnectorCluster {
	return dbapi.ConnectorCluster{
		Meta: api.Meta{
			ID:        from.Id,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.ModifiedAt,
		},
		Owner: from.Owner,
		Name:  from.Name,
		Status: dbapi.ConnectorClusterStatus{
			Phase: dbapi.ConnectorClusterPhaseEnum(from.Status.State),
		},
	}
}

func PresentConnectorClusterInstance(from dbapi.ConnectorCluster) public.ConnectorClusterInstance {
	reference := PresentReference(from.ID, from)
	return public.ConnectorClusterInstance{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Owner:     from.Owner,
		Name:      from.Name,
		CreatedAt: from.CreatedAt,
		ModifiedAt: from.UpdatedAt,
		Status: public.ConnectorClusterInstanceStatusStatus{
			State: public.ConnectorClusterState(from.Status.Phase),
		},
	}
}
