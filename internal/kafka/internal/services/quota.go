package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	// CheckIfQuotaIsDefinedForInstanceType checks if quota is defined for the given instance type
	CheckIfQuotaIsDefinedForInstanceType(username string, externalId string, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError)
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionId string) *errors.ServiceError
	// GetMarketplaceFromBillingAccountInformation gets the marketplace associated
	// to the provided billing account information.
	// If marketplace is not nil only accounts that belong to the provided
	// marketplace for the provided billingCloudAccountId are considered.
	// An error is returned if:
	// - There is more than one cloud account that matches the provided
	//   billing account information
	// - There is no cloud account that matches billing account information
	GetMarketplaceFromBillingAccountInformation(externalId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) (string, *errors.ServiceError)
}
