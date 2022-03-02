package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
)

func PresentConnectorAvailableOperatorUpgrade(req *dbapi.ConnectorDeploymentOperatorUpgrade) *private.ConnectorAvailableOperatorUpgrade {
	return &private.ConnectorAvailableOperatorUpgrade{
		ConnectorId:     req.ConnectorID,
		ConnectorTypeId: req.ConnectorTypeId,
		NamespaceId:       req.NamespaceID,
		Channel:         req.Channel,
		Operator: private.ConnectorAvailableOperatorUpgradeOperator{
			AssignedId:  req.Operator.Assigned.Id,
			AvailableId: req.Operator.Available.Id,
		},
	}
}

func ConvertConnectorAvailableOperatorUpgrade(req *private.ConnectorAvailableOperatorUpgrade) *dbapi.ConnectorDeploymentOperatorUpgrade {
	return &dbapi.ConnectorDeploymentOperatorUpgrade{
		ConnectorID:     req.ConnectorId,
		ConnectorTypeId: req.ConnectorTypeId,
		NamespaceID:     req.NamespaceId,
		Channel:         req.Channel,
		Operator: &dbapi.ConnectorOperatorUpgrade{
			Assigned: dbapi.ConnectorOperator{
				Id: req.Operator.AssignedId,
			},
			Available: dbapi.ConnectorOperator{
				Id: req.Operator.AvailableId,
			},
		},
	}
}
