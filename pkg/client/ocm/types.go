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

//func (t KafkaQuotaType) GetProduct() string {
//	if t == StandardQuota {
//		return string(RHOSAKProduct)
//	}
//
//	return string(RHOSAKTrialProduct)
//}
//
//func (t KafkaQuotaType) GetResourceName() string {
//	return RHOSAKResourceName
//}

//func (t KafkaQuotaType) Equals(t1 KafkaQuotaType) bool {
//	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
//}
