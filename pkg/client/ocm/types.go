package ocm

type Parameter struct {
	Id    string
	Value string
}

type KafkaQuotaType string

const (
	DeveloperQuota KafkaQuotaType = "developer"
	StandardQuota  KafkaQuotaType = "standard"
)

type KafkaProduct string

const (
	RHOSAKProduct      KafkaProduct = "RHOSAK"
	RHOSAKTrialProduct KafkaProduct = "RHOSAKTrial"
	RHOSAKEvalProduct  KafkaProduct = "RHOSAKEval"
)

const (
	RHOSAKResourceName string = "rhosak"
)

type AMSRelatedResourceBillingModel string

const (
	AMSRelatedResourceBillingModelMarketplace AMSRelatedResourceBillingModel = "marketplace"
	AMSRelatedResourceBillingModelStandard    AMSRelatedResourceBillingModel = "standard"
)

func (abm AMSRelatedResourceBillingModel) String() string {
	return string(abm)
}
