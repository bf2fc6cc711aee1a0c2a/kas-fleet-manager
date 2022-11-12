package types

type KafkaInstanceType string

const (
	DEVELOPER KafkaInstanceType = "developer"
	STANDARD  KafkaInstanceType = "standard"
)

var ValidKafkaInstanceTypes = []string{
	DEVELOPER.String(),
	STANDARD.String(),
}

func (t KafkaInstanceType) String() string {
	return string(t)
}
