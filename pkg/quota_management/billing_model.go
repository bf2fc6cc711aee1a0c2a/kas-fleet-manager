package quota_management

type BillingModel struct {
	Name        string `yaml:"name"`
	Expiration  int    `yaml:"expiration"`
	GracePeriod int    `yaml:"grace_period"`
	Allowed     int    `yaml:"allowed"`
}

type BillingModelList []BillingModel
