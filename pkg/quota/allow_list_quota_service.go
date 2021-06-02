package quota

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type allowListQuotaService struct {
	connectionFactory *db.ConnectionFactory
	configService     services.ConfigService
}

func (q allowListQuotaService) CheckQuota(kafka *api.KafkaRequest) *errors.ServiceError {
	if !q.configService.GetConfig().AccessControlList.EnableInstanceLimitControl {
		return nil
	}

	username := kafka.Owner
	orgId := kafka.OrganisationId
	var allowListItem config.AllowedListItem
	message := fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, config.GetDefaultMaxAllowedInstances())
	org, orgFound := q.configService.GetOrganisationById(orgId)
	filterByOrd := false
	if orgFound && org.IsUserAllowed(username) {
		allowListItem = org
		message = fmt.Sprintf("Organization '%s' has reached a maximum number of %d allowed instances.", orgId, org.GetMaxAllowedInstances())
		filterByOrd = true
	} else {
		user, userFound := q.configService.GetServiceAccountByUsername(username)
		if userFound {
			allowListItem = user
			message = fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances())
		}
	}

	dbConn := q.connectionFactory.New().Model(&api.KafkaRequest{})
	if filterByOrd {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}

	var count int64
	if err := dbConn.Count(&count).Error; err != nil {
		return errors.GeneralError("count failed from database")
	}

	total := int(count)
	maxInstanceReached := total >= config.GetDefaultMaxAllowedInstances()
	// check instance limit for internal users (users and orgs listed in allow list config)
	if allowListItem != nil {
		maxInstanceReached = !allowListItem.IsInstanceCountWithinLimit(total)
	}

	if maxInstanceReached {
		return errors.MaximumAllowedInstanceReached(message)
	}

	return nil
}

func (q allowListQuotaService) ReserveQuota(kafka *api.KafkaRequest) (string, *errors.ServiceError) {
	return "", nil // NOOP
}

func (q allowListQuotaService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	return nil // NOOP
}
