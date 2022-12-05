package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

func ConvertConnectorClusterRequest(id string, from public.ConnectorClusterRequest, userID, orgID string) *dbapi.ConnectorCluster {
	return &dbapi.ConnectorCluster{
		Model: db.Model{
			ID: id,
		},
		Owner:          userID,
		OrganisationId: orgID,
		Name:           from.Name,
		Annotations:    ConvertClusterAnnotations(id, from.Annotations),
		Status: dbapi.ConnectorClusterStatus{
			Phase: dbapi.ConnectorClusterPhaseDisconnected,
		},
	}
}
