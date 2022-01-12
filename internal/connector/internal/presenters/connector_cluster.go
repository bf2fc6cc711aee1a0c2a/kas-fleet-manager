package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
)

func ConvertConnectorCluster(from public.ConnectorCluster) dbapi.ConnectorCluster {
	return dbapi.ConnectorCluster{
		Meta: api.Meta{
			ID:        from.Id,
			CreatedAt: from.Metadata.CreatedAt,
			UpdatedAt: from.Metadata.UpdatedAt,
		},
		Owner: from.Metadata.Owner,
		Name:  from.Metadata.Name,
		Status: dbapi.ConnectorClusterStatus{
			Phase: from.Status,
		},
	}
}

func PresentConnectorCluster(from dbapi.ConnectorCluster) public.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return public.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: public.ConnectorClusterAllOfMetadata{
			Owner:     from.Owner,
			Name:      from.Name,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		Status: from.Status.Phase,
	}
}

func PresentConnectorClusterAdmin(from dbapi.ConnectorCluster) private.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return private.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: private.ConnectorClusterAllOfMetadata{
			Owner:     from.Owner,
			Name:      from.Name,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		Status: from.Status.Phase,
	}
}
