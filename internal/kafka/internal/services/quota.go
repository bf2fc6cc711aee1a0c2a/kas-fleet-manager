package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	// CheckQuota checks if the user can reserve quota, return an error otherwise
	CheckQuota(kafka *dbapi.KafkaRequest) *errors.ServiceError
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionId string) *errors.ServiceError
}
