package quota_management

type BillingModel struct {
	ID          string `yaml:"id"`
	Expiration  int    `yaml:"expiration"`
	GracePeriod int    `yaml:"grace_period"`
	Allowed     int    `yaml:"allowed"`
}

type BillingModelList []BillingModel
