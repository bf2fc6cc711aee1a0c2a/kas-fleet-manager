package ocm

type Parameter struct {
	Id    string
	Value string
}

type KafkaQuotaType string

const (
	EvalQuota     KafkaQuotaType = "eval"
	StandardQuota KafkaQuotaType = "standard"
)

func (t KafkaQuotaType) GetProduct() string {
	if t == StandardQuota {
		return "RHOSAK"
	}

	return "RHOSAKTrial"
}

func (t KafkaQuotaType) GetResourceName() string {
	return "rhosak"
}

func (t KafkaQuotaType) Equals(t1 KafkaQuotaType) bool {
	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
}
