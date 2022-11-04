package quota

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

const (
	billingModelStandard = "standard"
	defaultBillingModel  = billingModelStandard
)

type QuotaManagementListService struct {
	connectionFactory   *db.ConnectionFactory
	quotaManagementList *quota_management.QuotaManagementListConfig
	kafkaConfig         *config.KafkaConfig
}

var _ services.QuotaService = &QuotaManagementListService{}

// ValidateBillingAccount - don't validate billing accounts when using the quota list
func (q QuotaManagementListService) ValidateBillingAccount(organisationId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
	return nil
}

// CheckIfQuotaIsDefinedForInstanceType - returns if there is any quota configuration for the given instanceType/billingAccount pair (either organization or service account)
func (q QuotaManagementListService) CheckIfQuotaIsDefinedForInstanceType(username string, organisationId string, instanceType types.KafkaInstanceType, kafkaBillingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
	orgId := organisationId
	var account quota_management.Account
	org, orgFound := q.quotaManagementList.QuotaList.Organisations.GetById(orgId)
	userIsRegistered := false
	serviceAccountIsRegistered := false

	if orgFound && org.IsUserRegistered(username) {
		userIsRegistered = true
	} else {
		account, serviceAccountIsRegistered = q.quotaManagementList.QuotaList.ServiceAccounts.GetByUsername(username)
	}

	// if the user is registered, check that he has quota defined for the desired instance type
	if userIsRegistered && org.HasQuotaConfigurationFor(instanceType.String(), kafkaBillingModel.ID) {
		return true, nil
	}

	// if the serviceAccount is registered, check that he has quota defined for the desired instance type
	if serviceAccountIsRegistered && account.HasQuotaConfigurationFor(instanceType.String(), kafkaBillingModel.ID) {
		return true, nil
	}

	// if the user is not listed, he can create only DEVELOPER instances
	if !userIsRegistered && !serviceAccountIsRegistered && instanceType.String() == types.DEVELOPER.String() { // allow user who are not in quota list to create developer instances
		return true, nil
	}

	return false, nil
}

// ReserveQuota - tries to reserve the quota for the received kafka request
func (q QuotaManagementListService) ReserveQuota(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
	billingModelID, err := q.detectBillingModel(kafka)
	if err != nil {
		return "", err
	}
	// TODO: find a better place to set this instead of this side effect
	kafka.DesiredKafkaBillingModel = billingModelID

	if !q.quotaManagementList.EnableInstanceLimitControl {
		// TODO: find a better place to set this instead of this side effect
		kafka.ActualKafkaBillingModel = kafka.DesiredKafkaBillingModel
		return "", nil
	}

	username := kafka.Owner
	orgId := kafka.OrganisationId
	var quotaManagementListItem quota_management.QuotaManagementListItem
	message := fmt.Sprintf("user '%s' has reached a maximum number of %d allowed streaming units", username, quota_management.GetDefaultMaxAllowedInstances())
	org, orgFound := q.quotaManagementList.QuotaList.Organisations.GetById(orgId)
	filterByOrg := false
	if orgFound && org.IsUserRegistered(username) {
		quotaManagementListItem = org
		message = fmt.Sprintf("organization '%s' has reached a maximum number of %d allowed streaming units", orgId, org.GetMaxAllowedInstances(kafka.InstanceType, kafka.DesiredKafkaBillingModel))
		filterByOrg = true
	} else {
		user, userFound := q.quotaManagementList.QuotaList.ServiceAccounts.GetByUsername(username)
		if userFound {
			quotaManagementListItem = user
			message = fmt.Sprintf("user '%s' has reached a maximum number of %d allowed streaming units", username, user.GetMaxAllowedInstances(kafka.InstanceType, kafka.DesiredKafkaBillingModel))
		}
	}

	errMessage := fmt.Sprintf("failed to check kafka capacity for instance type '%s'", kafka.InstanceType)
	var totalInstanceCount int

	var kafkas []*dbapi.KafkaRequest
	dbConn := q.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("instance_type = ?", kafka.InstanceType).
		Where("actual_kafka_billing_model = ? or desired_kafka_billing_model = ?", kafka.DesiredKafkaBillingModel, kafka.DesiredKafkaBillingModel)

	if kafka.InstanceType != types.DEVELOPER.String() && filterByOrg {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}

	if err := dbConn.Model(&dbapi.KafkaRequest{}).
		Scan(&kafkas).Error; err != nil {
		return "", errors.GeneralError(errMessage)
	}

	for _, kafka := range kafkas {
		kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
		if e != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, e, errMessage)
		}
		totalInstanceCount += kafkaInstanceSize.CapacityConsumed
	}

	if quotaManagementListItem != nil && kafka.InstanceType != types.DEVELOPER.String() {
		kafkaInstanceSize, e := q.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
		if e != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, e, "error reserving quota")
		}
		if quotaManagementListItem.IsInstanceCountWithinLimit(kafka.InstanceType, billingModelID, totalInstanceCount+kafkaInstanceSize.CapacityConsumed) {
			// TODO: find a better place to set this
			kafka.ActualKafkaBillingModel = kafka.DesiredKafkaBillingModel
			return "", nil
		} else {
			return "", errors.MaximumAllowedInstanceReached(message)
		}
	}

	if kafka.InstanceType == types.DEVELOPER.String() && quotaManagementListItem == nil {
		if totalInstanceCount >= quota_management.GetDefaultMaxAllowedInstances() {
			return "", errors.MaximumAllowedInstanceReached(message)
		}
		// TODO: find a better place to set this
		kafka.ActualKafkaBillingModel = kafka.DesiredKafkaBillingModel
		return "", nil
	}

	return "", errors.InsufficientQuotaError("Insufficient quota")
}

// detectBillingModel - tries to detect the billing model. The flow is as follows:
// 1) If the user specified a 'desired' billing model, that billing model will be returned
// 2) If there is quota for the received organisation, this quota definition is used
// 3) If there is no quota defined for the received organisation, checks if there is quota defined for the received user (service account)
// 4) If both 2 & 3 fails, returns an error
// 5) Check that a quota configuration exists for the received instance type
// 6) If a quota configuration exists for the received instance type with billing model `STANDARD`, `STANDARD` billing model will be returned
// 7) if [6] fails, return the first defined billing model
func (q QuotaManagementListService) detectBillingModel(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
	if kafka.DesiredKafkaBillingModel != "" {
		return kafka.DesiredKafkaBillingModel, nil
	}
	if kafka.InstanceType == types.DEVELOPER.String() {
		return billingModelStandard, nil
	}

	var grantedQuota []quota_management.Quota

	org, orgFound := q.quotaManagementList.QuotaList.Organisations.GetById(kafka.OrganisationId)
	username := kafka.Owner
	if orgFound {
		grantedQuota = org.GetGrantedQuota()
	} else {
		user, userFound := q.quotaManagementList.QuotaList.ServiceAccounts.GetByUsername(username)
		if userFound {
			grantedQuota = user.GetGrantedQuota()
		} else {
			return "", errors.InsufficientQuotaError("unable to detect any valid billing model for organisation '%s' and user '%s'", kafka.OrganisationId, kafka.Owner)
		}
	}

	idx, quota := arrays.FindFirst(grantedQuota, func(q quota_management.Quota) bool {
		return shared.StringEqualsIgnoreCase(q.InstanceTypeID, kafka.InstanceType)
	})

	if idx == -1 {
		return "", errors.InsufficientQuotaError("no quota assigned for instance type: %s", kafka.InstanceType)
	}

	if kafka.Marketplace != "" {
		return "", errors.InsufficientQuotaError("marketplace is not supported when using QUOTA-LIST")
	}

	// check if the user has quota for STANDARD/STANDARD
	if bm, ok := quota.GetKafkaBillingModelByID(defaultBillingModel); ok {
		return bm.Id, nil
	}

	// The user has no quota defined for standard: returning the first available billing model
	// GetBillingModel always returns at least one element, so we can safely reference element
	return quota.GetKafkaBillingModels()[0].Id, nil
}

func (q QuotaManagementListService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	return nil // NOOP
}
