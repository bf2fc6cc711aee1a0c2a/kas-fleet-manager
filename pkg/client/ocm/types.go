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
