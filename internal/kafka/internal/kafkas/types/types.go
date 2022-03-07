package types

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

type KafkaInstanceType string

const (
	EVAL     KafkaInstanceType = "eval"
	STANDARD KafkaInstanceType = "standard"
)

var ValidKafkaInstanceTypes = []string{
	EVAL.String(),
	STANDARD.String(),
}

func (t KafkaInstanceType) String() string {
	return string(t)
}

func (t KafkaInstanceType) GetQuotaType() ocm.KafkaQuotaType {
	if t == STANDARD {
		return ocm.StandardQuota
	}
	return ocm.EvalQuota
}
