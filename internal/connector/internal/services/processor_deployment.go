package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"gorm.io/gorm"
)

func deleteProcessorDeployment(dbConn *gorm.DB, id string) *errors.ServiceError {
	// no err, deployment existed..
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ProcessorDeployment{}).Error; err != nil {
		err := services.HandleDeleteError("ProcessorDeployment", "id", id, err)
		if err != nil {
			return err
		}
	}
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ProcessorDeploymentStatus{}).Error; err != nil {
		err := services.HandleDeleteError("ProcessorDeploymentStatus", "id", id, err)
		if err != nil {
			return err
		}
	}
	return nil
}
