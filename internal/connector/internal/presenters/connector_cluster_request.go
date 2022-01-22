package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
)

func ConvertConnectorClusterRequest(from public.ConnectorClusterRequest) dbapi.ConnectorCluster {
	return dbapi.ConnectorCluster{
		Name:  from.Name,
	}
}
