package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"gorm.io/gorm"
)

// TODO: convert this as deployment service and move here relevant methods from services/connector_cluster.go
func deleteConnectorDeployment(dbConn *gorm.DB, id string) *errors.ServiceError {
	// no err, deployment existed..
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ConnectorDeployment{}).Error; err != nil {
		err := services.HandleDeleteError("ConnectorDeployment", "id", id, err)
		if err != nil {
			return err
		}
	}
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ConnectorDeploymentStatus{}).Error; err != nil {
		err := services.HandleDeleteError("ConnectorDeploymentStatus", "id", id, err)
		if err != nil {
			return err
		}
	}
	return nil
}
