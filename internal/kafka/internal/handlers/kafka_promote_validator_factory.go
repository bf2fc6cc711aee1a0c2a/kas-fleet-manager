package handlers

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

type KafkaPromoteValidatorFactory interface {
	GetValidator(quotaType api.QuotaType) (kafkaPromoteValidator, error)
}

type DefaultKafkaPromoteValidatorFactory struct {
	kafkaPromoteValidators map[api.QuotaType]kafkaPromoteValidator
}

var _ KafkaPromoteValidatorFactory = &DefaultKafkaPromoteValidatorFactory{}

func NewDefaultKafkaPromoteValidatorFactory(
	kafkaConfig *config.KafkaConfig,
) *DefaultKafkaPromoteValidatorFactory {

	return &DefaultKafkaPromoteValidatorFactory{
		kafkaPromoteValidators: map[api.QuotaType]kafkaPromoteValidator{
			api.QuotaManagementListQuotaType: &quotaManagementListKafkaPromoteValidator{
				KafkaConfig: kafkaConfig,
			},
			api.AMSQuotaType: &amsKafkaPromoteValidator{
				KafkaConfig: kafkaConfig,
			},
		},
	}
}

func (v *DefaultKafkaPromoteValidatorFactory) GetValidator(quotaType api.QuotaType) (kafkaPromoteValidator, error) {
	kafkaPromoteValidator, ok := v.kafkaPromoteValidators[quotaType]
	if !ok {
		return nil, fmt.Errorf("invalid quota type: %v", quotaType)
	}

	return kafkaPromoteValidator, nil
}
