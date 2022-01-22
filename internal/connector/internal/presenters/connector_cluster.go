package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertConnectorCluster(from public.ConnectorCluster) dbapi.ConnectorCluster {
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

func PresentConnectorCluster(from dbapi.ConnectorCluster) public.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return public.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Owner:     from.Owner,
		Name:      from.Name,
		CreatedAt: from.CreatedAt,
		ModifiedAt: from.UpdatedAt,
		Status: public.ConnectorClusterStatusStatus{
			State: public.ConnectorClusterState(from.Status.Phase),
		},
	}
}

func PresentPrivateConnectorCluster(from dbapi.ConnectorCluster) private.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return private.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Owner:     from.Owner,
		Name:      from.Name,
		CreatedAt: from.CreatedAt,
		ModifiedAt: from.UpdatedAt,
		Status: private.ConnectorClusterStatusStatus{
			State: private.ConnectorClusterState(from.Status.Phase),
		},
	}
}