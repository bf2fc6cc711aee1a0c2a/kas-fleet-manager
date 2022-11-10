package utils

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

type BillingModelDetails struct {
	KafkaBillingModel config.KafkaBillingModel
	AMSBillingModel   string
}
