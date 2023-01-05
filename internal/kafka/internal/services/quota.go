package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	// CheckIfQuotaIsDefinedForInstanceType checks if quota is defined for the given instance type
	CheckIfQuotaIsDefinedForInstanceType(username string, externalID string, instanceTypeID types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError)
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError)
	// ReserveQuotaIfNotAlreadyReserved reserves a quota for the specified request if the desired quota
	// has not been already reserved. Returns the id of the newly reserved quota or the id of the existing one
	ReserveQuotaIfNotAlreadyReserved(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionId string) *errors.ServiceError
	// DeleteQuotaForBillingModel deletes a reserved quota only if it is related to the specified billing model, otherwise exits with no error
	DeleteQuotaForBillingModel(subscriptionId string, kafkaBillingModel config.KafkaBillingModel) *errors.ServiceError
	// ValidateBillingAccount validates if a billing account is contained in the quota cost response
	ValidateBillingAccount(organisationId string, instanceType types.KafkaInstanceType, billingModelID string, billingCloudAccountId string, marketplace *string) *errors.ServiceError
}
