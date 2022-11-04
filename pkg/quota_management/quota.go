package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var defaultBillingModel = BillingModel{
	ID:          "STANDARD",
	Expiration:  0, // no expiration
	GracePeriod: 0, // no grace period
}

var defaultBillingModels = []BillingModel{defaultBillingModel}

type Quota struct {
	InstanceTypeID string           `yaml:"instance_type_id"`
	BillingModels  BillingModelList `yaml:"billing_models,omitempty"`
}

func (quota *Quota) GetBillingModels() BillingModelList {
	if len(quota.BillingModels) == 0 {
		ret := defaultBillingModel
		ret.Allowed = 0 // if 0, defaults to the MaxAllowedInstances
		return defaultBillingModels
	}

	return quota.BillingModels
}

func (quota *Quota) GetBillingModelByID(billingModelID string) (BillingModel, bool) {

	if idx, bm := arrays.FindFirst(quota.GetBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.ID, billingModelID) }); idx != -1 {
		return bm, true
	}

	return BillingModel{}, false
}

type QuotaList []Quota
