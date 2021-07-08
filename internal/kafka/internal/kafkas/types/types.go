package types

type KafkaInstanceType string

const (
	EVAL     KafkaInstanceType = "eval"
	STANDARD KafkaInstanceType = "standard"
)

func (t KafkaInstanceType) String() string {
	return string(t)
}

func (t KafkaInstanceType) ToProductType() string {
	if t == STANDARD {
		return "RHOSAK"
	}

	return "RHOSAKTrial"
}
