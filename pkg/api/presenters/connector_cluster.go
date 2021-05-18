package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
)

func ConvertConnectorCluster(from openapi.ConnectorCluster) api.ConnectorCluster {
	return api.ConnectorCluster{
		Meta: api.Meta{
			ID:        from.Id,
			CreatedAt: from.Metadata.CreatedAt,
			UpdatedAt: from.Metadata.UpdatedAt,
		},
		Owner: from.Metadata.Owner,
		Name:  from.Metadata.Name,
		Status: api.ConnectorClusterStatus{
			Phase: from.Status,
		},
	}
}

func PresentConnectorCluster(from api.ConnectorCluster) openapi.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return openapi.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorClusterAllOfMetadata{
			Owner:     from.Owner,
			Name:      from.Name,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		Status: from.Status.Phase,
	}
}
