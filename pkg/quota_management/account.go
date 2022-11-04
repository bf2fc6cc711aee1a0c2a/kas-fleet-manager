package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

type Account struct {
	Username            string    `yaml:"username"`
	MaxAllowedInstances int       `yaml:"max_allowed_instances"`
	GrantedQuota        QuotaList `yaml:"granted_quota,omitempty"`
}

var _ QuotaManagementListItem = &Account{}

func (account Account) IsInstanceCountWithinLimit(instanceTypeID string, billingModelID string, count int) bool {
	return count <= account.GetMaxAllowedInstances(instanceTypeID, billingModelID)
}

func (account Account) GetMaxAllowedInstances(instanceTypeID string, billingModelID string) int {
	bm, ok := account.getBillingModel(instanceTypeID, billingModelID)

	if !ok {
		return 0
	}

	if bm.Allowed <= 0 {
		if account.MaxAllowedInstances <= 0 {
			return MaxAllowedInstances
		}
		return account.MaxAllowedInstances
	}

	return bm.Allowed
}

func (account Account) GetGrantedQuota() QuotaList {
	if len(account.GrantedQuota) == 0 {
		return defaultQuotaList
	}
	return account.GrantedQuota
}

func (account Account) getBillingModel(instanceTypeId string, billingModelID string) (BillingModel, bool) {
	grantedQuota := account.GetGrantedQuota()

	idx, instanceType := arrays.FindFirst(grantedQuota, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) })
	if idx != -1 {
		idx, bm := arrays.FindFirst(instanceType.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.ID, billingModelID) })
		if idx != -1 {
			return bm, true
		}
	}
	return BillingModel{}, false
}

func (account Account) HasQuotaFor(instanceTypeId string, billingModelID string) bool {
	_, ok := account.getBillingModel(instanceTypeId, billingModelID)
	return ok
}

type AccountList []Account

func (allowedAccounts AccountList) GetByUsername(username string) (Account, bool) {
	for _, user := range allowedAccounts {
		if username == user.Username {
			return user, true
		}
	}

	return Account{}, false
}
