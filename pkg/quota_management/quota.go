package quota_management

var defaultBillingModel = BillingModel{
	Name:        "STANDARD",
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
		ret.Allowed = 1 // TODO: change to 'default max allowed instances'
		return defaultBillingModels
	}

	return quota.BillingModels
}

type QuotaList []Quota
