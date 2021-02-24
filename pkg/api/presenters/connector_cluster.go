package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertConnectorCluster(from openapi.ConnectorCluster) (*api.ConnectorCluster, *errors.ServiceError) {

	return &api.ConnectorCluster{
		Meta: api.Meta{
			ID:        from.Id,
			CreatedAt: from.Metadata.CreatedAt,
			UpdatedAt: from.Metadata.UpdatedAt,
		},
		Owner:      from.Metadata.Owner,
		Name:       from.Metadata.Name,
		AddonGroup: from.Metadata.Group,
		Status:     from.Status,
	}, nil
}

func PresentConnectorCluster(from *api.ConnectorCluster) (openapi.ConnectorCluster, *errors.ServiceError) {
	reference := PresentReference(from.ID, from)
	return openapi.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorClusterAllOfMetadata{
			Owner:     from.Owner,
			Name:      from.Name,
			Group:     from.AddonGroup,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		Status: from.Status,
	}, nil
}
