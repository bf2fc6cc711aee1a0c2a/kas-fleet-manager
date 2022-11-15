package quota_management

type BillingModel struct {
	Id                  string `yaml:"id"`
	ExpirationDays      int    `yaml:"expiration_days"`
	GracePeriodDays     int    `yaml:"grace_period_days"`
	MaxAllowedInstances int    `yaml:"max_allowed_instances"`
}

type BillingModelList []BillingModel
