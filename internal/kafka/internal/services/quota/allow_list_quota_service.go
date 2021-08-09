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

type ResKafkaInstanceCount struct {
	InstanceType string
	Count        int
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

	var res []ResKafkaInstanceCount
	dbConn := q.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Select("instance_type, count(1) as Count")

	if filterByOrd {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}
	dbConn = dbConn.Group("instance_type").Scan(&res)
	if err := dbConn.Error; err != nil {
		return false, errors.GeneralError("count failed from database")
	}
	instanceTypeCountMap := map[string]int{
		types.EVAL.String():     0,
		types.STANDARD.String(): 0,
	}

	for _, kafkaInstance := range res {
		instanceTypeCountMap[kafkaInstance.InstanceType] = kafkaInstance.Count
	}

	if allowListItem != nil {
		total := instanceTypeCountMap[types.STANDARD.String()]
		if allowListItem.IsInstanceCountWithinLimit(total) {
			return true, nil
		} else if evalInstancesReached(instanceTypeCountMap) {
			return false, errors.MaximumAllowedInstanceReached(message)
		}
	} else if evalInstancesReached(instanceTypeCountMap) { // user not in allow list can only create eval instances. TODO, they should be able to create standard instances too.
		return false, errors.MaximumAllowedInstanceReached(message)
	}

	return false, nil
}

func (q allowListQuotaService) ReserveQuota(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
	return "", nil // NOOP
}

func (q allowListQuotaService) DeleteQuota(SubscriptionId string) *errors.ServiceError {
	return nil // NOOP
}

func evalInstancesReached(instanceTypeCountMap map[string]int) bool {
	return instanceTypeCountMap[types.EVAL.String()] >= acl.GetDefaultMaxAllowedInstances()
}
