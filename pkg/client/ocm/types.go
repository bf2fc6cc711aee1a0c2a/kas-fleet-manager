package ocm

type Parameter struct {
	Id    string
	Value string
}

type DinosaurQuotaType string

const (
	EvalQuota     DinosaurQuotaType = "eval"
	StandardQuota DinosaurQuotaType = "standard"
)

func (t DinosaurQuotaType) GetProduct() string {
	if t == StandardQuota {
		return "RHOSAK"
	}

	return "RHOSAKTrial"
}

func (t DinosaurQuotaType) GetResourceName() string {
	return "rhosak"
}

func (t DinosaurQuotaType) Equals(t1 DinosaurQuotaType) bool {
	return t1.GetProduct() == t.GetProduct() && t1.GetResourceName() == t.GetResourceName()
}
