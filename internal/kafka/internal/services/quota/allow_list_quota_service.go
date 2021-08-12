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

	dbConn := q.connectionFactory.New().Model(&dbapi.KafkaRequest{})
	if filterByOrd {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}

	var count int64
	if err := dbConn.Count(&count).Error; err != nil {
		return false, errors.GeneralError("count failed from database")
	}

	total := int(count)
	maxInstanceReached := total >= acl.GetDefaultMaxAllowedInstances()
	// check instance limit for internal users (users and orgs listed in allow list config)
	if allowListItem != nil {
		maxInstanceReached = !allowListItem.IsInstanceCountWithinLimit(total)
	}

	if maxInstanceReached {
		return false, errors.MaximumAllowedInstanceReached(message)
	}

	return true, nil
}

func (q allowListQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
	return "", nil // NOOP
}

func (q allowListQuotaService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	return nil // NOOP
}
