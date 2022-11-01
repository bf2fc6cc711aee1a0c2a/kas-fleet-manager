package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var defaultInstanceType = Quota{
	InstanceTypeID: "STANDARD",
	BillingModels:  nil,
}

var defaultInstanceTypes = []Quota{defaultInstanceType}

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

func (org Organisation) IsInstanceCountWithinLimit(instanceTypeID string, billingModelName string, count int) bool {
	return count <= org.GetMaxAllowedInstances(instanceTypeID, billingModelName)
}

func (org Organisation) GetMaxAllowedInstances(instanceTypeID string, billingModelName string) int {
	bm, ok := org.getBillingModel(instanceTypeID, billingModelName)

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
		return defaultInstanceTypes
	}
	return org.GrantedQuota
}

func (org Organisation) getBillingModel(instanceTypeId string, billingModelName string) (BillingModel, bool) {
	grantedQuota := org.GetGrantedQuota()

	idx, instanceType := arrays.FindFirst(grantedQuota, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) })
	if idx != -1 {
		idx, bm := arrays.FindFirst(instanceType.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.Name, billingModelName) })
		if idx != -1 {
			return bm, true
		}
	}
	return BillingModel{}, false
}

func (org Organisation) HasQuotaFor(instanceTypeId string, billingModelName string) bool {
	instanceTypes := org.GetGrantedQuota()

	idx, instanceType := arrays.FindFirst(instanceTypes, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) })
	return idx != -1 && arrays.AnyMatch(instanceType.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.Name, billingModelName) })
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
