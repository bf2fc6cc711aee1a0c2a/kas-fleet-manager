package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var defaultQuotaForStandard = Quota{
	InstanceTypeID: "standard",
	BillingModels:  nil,
}

var defaultQuotaList = []Quota{defaultQuotaForStandard}

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
	bm, ok := org.getBillingModel(instanceTypeID, billingModelID)

	if !ok {
		return 0
	}

	if bm.Allowed <= 0 {
		if org.MaxAllowedInstances <= 0 {
			return MaxAllowedInstances
		}
		return org.MaxAllowedInstances
	}

	return bm.Allowed
}

func (org Organisation) GetGrantedQuota() QuotaList {
	if len(org.GrantedQuota) == 0 {
		return defaultQuotaList
	}
	return org.GrantedQuota
}

func (org Organisation) getBillingModel(instanceTypeId string, billingModelID string) (BillingModel, bool) {
	grantedQuota := org.GetGrantedQuota()

	idx, instanceType := arrays.FindFirst(grantedQuota, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) })
	if idx != -1 {
		idx, bm := arrays.FindFirst(instanceType.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.ID, billingModelID) })
		if idx != -1 {
			return bm, true
		}
	}
	return BillingModel{}, false
}

func (org Organisation) HasQuotaFor(instanceTypeId string, billingModelID string) bool {
	instanceTypes := org.GetGrantedQuota()

	idx, instanceType := arrays.FindFirst(instanceTypes, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) })
	return idx != -1 && arrays.AnyMatch(instanceType.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.ID, billingModelID) })
}

type OrganisationList []Organisation

func (orgList OrganisationList) GetById(Id string) (Organisation, bool) {
	for _, organisation := range orgList {
		if Id == organisation.Id {
			return organisation, true
		}
	}

	return Organisation{}, false
}
