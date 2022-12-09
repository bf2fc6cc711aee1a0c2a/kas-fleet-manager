package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var defaultQuotaForStandard = Quota{
	InstanceTypeID:     "standard",
	KafkaBillingModels: nil,
}

var defaultGrantedQuota = []Quota{defaultQuotaForStandard}

var _ QuotaManagementListItem = &Organisation{}

type Organisation struct {
	Id                  string      `yaml:"id"`
	AnyUser             bool        `yaml:"any_user"`
	MaxAllowedInstances int         `yaml:"max_allowed_instances"`
	RegisteredUsers     AccountList `yaml:"registered_users"`
	GrantedQuota        QuotaList   `yaml:"granted_quota,omitempty"`
}

func (org Organisation) IsUserRegistered(username string) bool {
	if !org.HasUsersRegistered() {
		return org.AnyUser
	}
	_, found := org.RegisteredUsers.GetByUsername(username)
	return found
}

func (org Organisation) HasUsersRegistered() bool {
	return len(org.RegisteredUsers) > 0
}

func (org Organisation) IsInstanceCountWithinLimit(instanceTypeID string, billingModelID string, count int) bool {
	return count <= org.GetMaxAllowedInstances(instanceTypeID, billingModelID)
}

func (org Organisation) GetMaxAllowedInstances(instanceTypeID string, billingModelID string) int {
	bm, ok := getBillingModel(org.GetGrantedQuota(), instanceTypeID, billingModelID)

	if !ok || bm.HasExpired() {
		return 0
	}

	if bm.MaxAllowedInstances > 0 {
		return bm.MaxAllowedInstances
	}

	if org.MaxAllowedInstances > 0 {
		return org.MaxAllowedInstances
	}

	return MaxAllowedInstances
}

func (org Organisation) GetGrantedQuota() QuotaList {
	if len(org.GrantedQuota) == 0 {
		return defaultGrantedQuota
	}
	return org.GrantedQuota
}

func (org Organisation) HasQuotaConfigurationFor(instanceTypeId string, billingModelID string) bool {
	return hasQuotaConfigurationFor(org.GetGrantedQuota(), instanceTypeId, billingModelID)
}

type OrganisationList []Organisation

func (orgList OrganisationList) GetById(Id string) (Organisation, bool) {
	idx, org := arrays.FindFirst(orgList, func(o Organisation) bool { return Id == o.Id })
	return org, idx != -1
}
