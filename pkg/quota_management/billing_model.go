package quota_management

type BillingModel struct {
	Id                  string          `yaml:"id"`
	ExpirationDate      *ExpirationDate `yaml:"expiration_date,omitempty"`
	MaxAllowedInstances int             `yaml:"max_allowed_instances"`
}

func (bm *BillingModel) HasExpired() bool {
	return bm.ExpirationDate.HasExpired()
}

type BillingModelList []BillingModel
