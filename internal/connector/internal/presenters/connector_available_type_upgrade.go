package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
)

func PresentConnectorAvailableTypeUpgrade(req *dbapi.ConnectorDeploymentTypeUpgrade) *private.ConnectorAvailableTypeUpgrade {
	return &private.ConnectorAvailableTypeUpgrade{
		ConnectorId:     req.ConnectorID,
		ConnectorTypeId: req.ConnectorTypeId,
		Namespace:       req.Namespace,
		Channel:         req.Channel,
		ShardMetadata: private.ConnectorAvailableTypeUpgradeShardMetadata{
			AssignedId:  req.ShardMetadata.AssignedId,
			AvailableId: req.ShardMetadata.AvailableId,
		},
	}
}

func ConvertConnectorAvailableTypeUpgrade(req *private.ConnectorAvailableTypeUpgrade) *dbapi.ConnectorDeploymentTypeUpgrade {
	return &dbapi.ConnectorDeploymentTypeUpgrade{
		ConnectorID:     req.ConnectorId,
		ConnectorTypeId: req.ConnectorTypeId,
		Namespace:       req.Namespace,
		Channel:         req.Channel,
		ShardMetadata: &dbapi.ConnectorTypeUpgrade{
			AssignedId:  req.ShardMetadata.AssignedId,
			AvailableId: req.ShardMetadata.AvailableId,
		},
	}
}
