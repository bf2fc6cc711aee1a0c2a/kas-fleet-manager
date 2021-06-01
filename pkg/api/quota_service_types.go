package api

const (
	AMSQuotaType       QuotaType = "ams"
	AllowListQuotaType QuotaType = "allow-list"
	UndefinedQuotaType QuotaType = ""
)

type QuotaType string

func (quotaType QuotaType) String() string {
	return string(quotaType)
}
