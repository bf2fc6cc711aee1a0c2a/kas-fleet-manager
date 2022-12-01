package handlers

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type kafkaPromoteValidatorRequest struct {
	DesiredKafkaBillingModel   string
	DesiredKafkaMarketplace    string
	DesiredKafkaCloudAccountID string
	KafkaInstanceType          string
}

type kafkaPromoteValidator interface {
	Validate(r kafkaPromoteValidatorRequest) error
}

type amsKafkaPromoteValidator struct {
	KafkaConfig *config.KafkaConfig
}

var _ kafkaPromoteValidator = &amsKafkaPromoteValidator{}

func (v *amsKafkaPromoteValidator) Validate(r kafkaPromoteValidatorRequest) error {
	desiredAMSBillingModel := r.DesiredKafkaBillingModel
	// TODO improve not referencing a hardcoded string
	if r.DesiredKafkaBillingModel == "marketplace" {
		if r.DesiredKafkaMarketplace == "" {
			return errors.BadRequest("marketplace attribute is mandatory when desired kafka billing model is 'marketplace'")
		}

		if r.DesiredKafkaCloudAccountID == "" {
			return errors.BadRequest("billing cloud account id attribute is mandatory when desired kafka billing model is 'marketplace'")
		}

		// we construct AMS billing model from the combination of desired
		// Kafka billing model and desired marketplace
		desiredAMSBillingModel = fmt.Sprintf("%s-%s", desiredAMSBillingModel, r.DesiredKafkaMarketplace)
	}

	kafkaInstanceTypeConfig, err := v.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(r.KafkaInstanceType)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error getting instance type: %s", err.Error())
	}

	kafkaBillingModelConfig, err := kafkaInstanceTypeConfig.GetKafkaSupportedBillingModelByID(r.DesiredKafkaBillingModel)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error getting kafka billing model: %s", err.Error())
	}

	bmExists := kafkaBillingModelConfig.HasSupportForAMSBillingModel(desiredAMSBillingModel)
	if !bmExists {
		return fmt.Errorf("AMS billing model %q is not supported for kafka billing model %q in instance type %q", desiredAMSBillingModel, r.DesiredKafkaBillingModel, r.KafkaInstanceType)
	}

	return nil
}

type quotaManagementListKafkaPromoteValidator struct {
	KafkaConfig *config.KafkaConfig
}

var _ kafkaPromoteValidator = &quotaManagementListKafkaPromoteValidator{}

func (v *quotaManagementListKafkaPromoteValidator) Validate(r kafkaPromoteValidatorRequest) error {
	kafkaInstanceTypeConfig, err := v.KafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(r.KafkaInstanceType)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error getting instance type: %s", err.Error())
	}

	_, err = kafkaInstanceTypeConfig.GetKafkaSupportedBillingModelByID(r.DesiredKafkaBillingModel)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error getting kafka billing model: %s", err.Error())
	}

	return nil
}
