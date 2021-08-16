package quota

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type allowListQuotaService struct {
	connectionFactory *db.ConnectionFactory
	accessControlList *acl.AccessControlListConfig
}

func (q allowListQuotaService) CheckIfQuotaIsDefinedForInstanceType(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
	if !q.accessControlList.EnableInstanceLimitControl {
		return true, nil
	}

	username := kafka.Owner
	orgId := kafka.OrganisationId
	org, orgFound := q.accessControlList.AllowList.Organisations.GetById(orgId)
	userInAllowList := false
	if orgFound && org.IsUserAllowed(username) {
		userInAllowList = true
	} else {
		_, userFound := q.accessControlList.AllowList.ServiceAccounts.GetByUsername(username)
		userInAllowList = userFound
	}

	// allow user defined in allow list to create standard instances
	if userInAllowList && instanceType == types.STANDARD {
		return true, nil
	} else if !userInAllowList && instanceType == types.EVAL { // allow user who are not in allow list to create eval instances
		return true, nil
	}

	return false, nil
}

func (q allowListQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
	if !q.accessControlList.EnableInstanceLimitControl {
		return "", nil
	}

	username := kafka.Owner
	orgId := kafka.OrganisationId
	var allowListItem acl.AllowedListItem
	message := fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, acl.GetDefaultMaxAllowedInstances())
	org, orgFound := q.accessControlList.AllowList.Organisations.GetById(orgId)
	filterByOrd := false
	if orgFound && org.IsUserAllowed(username) {
		allowListItem = org
		message = fmt.Sprintf("Organization '%s' has reached a maximum number of %d allowed instances.", orgId, org.GetMaxAllowedInstances())
		filterByOrd = true
	} else {
		user, userFound := q.accessControlList.AllowList.ServiceAccounts.GetByUsername(username)
		if userFound {
			allowListItem = user
			message = fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances())
		}
	}

	var count int64
	dbConn := q.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("instance_type = ?", instanceType.String())

	if instanceType == types.STANDARD && filterByOrd {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}

	if err := dbConn.Count(&count).Error; err != nil {
		return "", errors.GeneralError("count failed from database")
	}

	totalInstanceCount := int(count)
	if allowListItem != nil && instanceType == types.STANDARD {
		if allowListItem.IsInstanceCountWithinLimit(totalInstanceCount) {
			return "", nil
		} else {
			return "", errors.MaximumAllowedInstanceReached(message)
		}
	}

	if instanceType == types.EVAL && allowListItem == nil {
		if totalInstanceCount >= acl.GetDefaultMaxAllowedInstances() {
			return "", errors.MaximumAllowedInstanceReached(message)
		}
		return "", nil
	}

	return "", errors.InsufficientQuotaError("Insufficient Quota")
}

func (q allowListQuotaService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	return nil // NOOP
}
