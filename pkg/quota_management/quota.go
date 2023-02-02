package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

var defaultBillingModel = BillingModel{
	Id: "standard",
}

var defaultBillingModels = []BillingModel{defaultBillingModel}

type Quota struct {
	InstanceTypeID     string           `yaml:"instance_type_id"`
	KafkaBillingModels BillingModelList `yaml:"kafka_billing_models,omitempty"`
}

func (quota *Quota) GetKafkaBillingModels() BillingModelList {
	if len(quota.KafkaBillingModels) == 0 {
		return defaultBillingModels
	}

	return quota.KafkaBillingModels
}

func (quota *Quota) GetKafkaBillingModelByID(billingModelId string) (BillingModel, bool) {

	if idx, bm := arrays.FindFirst(quota.GetKafkaBillingModels(), func(bm BillingModel) bool { return shared.StringEqualsIgnoreCase(bm.Id, billingModelId) }); idx != -1 {
		return bm, true
	}

	return BillingModel{}, false
}

type QuotaList []Quota
