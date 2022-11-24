package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

func getBillingModel(grantedQuota QuotaList, instanceTypeId string, billingModelID string) (BillingModel, bool) {
	if idx, instanceType := arrays.FindFirst(grantedQuota, func(x Quota) bool { return shared.StringEqualsIgnoreCase(x.InstanceTypeID, instanceTypeId) }); idx != -1 {
		if idx, bm := arrays.FindFirst(instanceType.GetKafkaBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.Id, billingModelID) }); idx != -1 {
			return bm, true
		}
	}

	return BillingModel{}, false
}

func hasQuotaConfigurationFor(grantedQuota QuotaList, instanceTypeId string, billingModelID string) bool {
	_, ok := getBillingModel(grantedQuota, instanceTypeId, billingModelID)
	return ok
}
