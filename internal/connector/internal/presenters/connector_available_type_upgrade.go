package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
)

func PresentConnectorAvailableTypeUpgrade(req *dbapi.ConnectorDeploymentTypeUpgrade) *public.ConnectorAvailableTypeUpgrade {
	return &public.ConnectorAvailableTypeUpgrade{
		ConnectorId:     req.ConnectorID,
		ConnectorTypeId: req.ConnectorTypeId,
		Channel:         req.Channel,
		ShardMetadata: public.ConnectorAvailableTypeUpgradeShardMetadata{
			AssignedId:  req.ShardMetadata.AssignedId,
			AvailableId: req.ShardMetadata.AvailableId,
		},
	}
}

func ConvertConnectorAvailableTypeUpgrade(req *public.ConnectorAvailableTypeUpgrade) *dbapi.ConnectorDeploymentTypeUpgrade {
	return &dbapi.ConnectorDeploymentTypeUpgrade{
		ConnectorID:     req.ConnectorId,
		ConnectorTypeId: req.ConnectorTypeId,
		Channel:         req.Channel,
		ShardMetadata: &dbapi.ConnectorTypeUpgrade{
			AssignedId:  req.ShardMetadata.AssignedId,
			AvailableId: req.ShardMetadata.AvailableId,
		},
	}
}
