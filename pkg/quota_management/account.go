package quota_management

import (
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
	bm, ok := getBillingModel(account.GetGrantedQuota(), instanceTypeID, billingModelID)

	if !ok || bm.HasExpired() {
		return 0
	}

	if bm.MaxAllowedInstances > 0 {
		return bm.MaxAllowedInstances
	}

	if account.MaxAllowedInstances > 0 {
		return account.MaxAllowedInstances
	}

	return MaxAllowedInstances
}

func (account Account) GetGrantedQuota() QuotaList {
	if len(account.GrantedQuota) == 0 {
		return defaultGrantedQuota
	}
	return account.GrantedQuota
}

func (account Account) HasQuotaConfigurationFor(instanceTypeId string, billingModelID string) bool {
	return hasQuotaConfigurationFor(account.GetGrantedQuota(), instanceTypeId, billingModelID)
}

type AccountList []Account

func (allowedAccounts AccountList) GetByUsername(username string) (Account, bool) {
	idx, account := arrays.FindFirst(allowedAccounts, func(a Account) bool { return username == a.Username })
	return account, idx != -1
}
