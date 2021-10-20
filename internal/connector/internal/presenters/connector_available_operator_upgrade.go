package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
)

func PresentConnectorAvailableOperatorUpgrade(req *dbapi.ConnectorDeploymentOperatorUpgrade) *public.ConnectorAvailableOperatorUpgrade {
	return &public.ConnectorAvailableOperatorUpgrade{
		ConnectorId:     req.ConnectorID,
		ConnectorTypeId: req.ConnectorTypeId,
		Channel:         req.Channel,
		Operator: public.ConnectorAvailableOperatorUpgradeOperator{
			AssignedId:  req.Operator.Assigned.Id,
			AvailableId: req.Operator.Available.Id,
		},
	}
}

func ConvertConnectorAvailableOperatorUpgrade(req *public.ConnectorAvailableOperatorUpgrade) *dbapi.ConnectorDeploymentOperatorUpgrade {
	return &dbapi.ConnectorDeploymentOperatorUpgrade{
		ConnectorID:     req.ConnectorId,
		ConnectorTypeId: req.ConnectorTypeId,
		Channel:         req.Channel,
		Operator:        &dbapi.ConnectorOperatorUpgrade{
			Assigned:  dbapi.ConnectorOperator{
				Id:      req.Operator.AssignedId,
			},
			Available: dbapi.ConnectorOperator{
				Id: req.Operator.AvailableId,
			},
		},
	}
}
