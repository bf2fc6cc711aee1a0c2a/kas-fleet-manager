package utils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

/********************************************************************************************************************
 * This resolver is used when the user specifies a `DesiredBillingModel` in the request.
 * If the specified `DesiredBillingModel` is a `marketplace`, attempts are done to detect the marketplace type (if
 * not specified)
 ********************************************************************************************************************/

var _ BillingModelResolver = &kafkaBillingModelResolver{}

type kafkaBillingModelResolver struct {
	quotaConfigProvider AMSQuotaConfigProvider
	kafkaConfig         *config.KafkaConfig
}

func (k *kafkaBillingModelResolver) SupportRequest(kafka *dbapi.KafkaRequest) bool {
	return kafka.DesiredKafkaBillingModel != ""
}

// Resolve - Tries to resolve the ams billing model to be used.
// 1) If the specified `DesiredBillingModel` supports only one `ams billing model`, the supported `ams billing model` will be returned.
// 2) If a marketplace type has been specified and one of the supported `ams billing model` supports that marketplace, then that `ams billing model` will be returned.
// 3) In all the other cases, the detection is demanded to the `detectAmsBillingModel` method.
func (k *kafkaBillingModelResolver) Resolve(orgId string, kafka *dbapi.KafkaRequest) (BillingModelDetails, error) {
	kafkaBillingModel, err := k.kafkaConfig.GetBillingModelByID(kafka.InstanceType, kafka.DesiredKafkaBillingModel)
	if err != nil {
		return BillingModelDetails{}, err
	}

	if kafka.Marketplace != "" && !kafkaBillingModel.HasSupportForMarketplace() {
		return BillingModelDetails{}, errors.InsufficientQuotaError("marketplace value '%s' is not compatible with billing model '%s'", kafka.Marketplace, kafkaBillingModel.ID)
	}

	if len(kafkaBillingModel.AMSBillingModels) == 1 {
		// it supports only one amsbilling model: we don't need to detect anything
		return BillingModelDetails{
			KafkaBillingModel: kafkaBillingModel,
			AMSBillingModel:   kafkaBillingModel.AMSBillingModels[0],
		}, nil
	}

	if kafka.Marketplace != "" {
		amsMarketplace := "marketplace-" + kafka.Marketplace
		if arrays.AnyMatch(kafkaBillingModel.AMSBillingModels, arrays.StringEqualsIgnoreCasePredicate(amsMarketplace)) {
			return BillingModelDetails{
				KafkaBillingModel: kafkaBillingModel,
				AMSBillingModel:   amsMarketplace,
			}, nil
		}
		return BillingModelDetails{}, errors.InsufficientQuotaError("ams marketplace '%s' is not supported by billing mode '%s'. Supported marketplaces are %v", kafka.Marketplace, kafkaBillingModel.ID, kafkaBillingModel.AMSBillingModels)
	}

	// If we arrive here, it means that the kafka billing model support more than one ams billing model and no marketplace has been specified
	// We need to detect the ams billing model
	amsBillingModel, err := k.detectAmsBillingModel(orgId, kafka, kafkaBillingModel)
	if err != nil {
		return BillingModelDetails{}, err
	}
	return BillingModelDetails{
		KafkaBillingModel: kafkaBillingModel,
		AMSBillingModel:   amsBillingModel,
	}, nil
}

// detectAmsBillingModel - tries to detect the `ams billing model`
// 1) If the user has specified a cloud account id, it falls back to the `cloud account billing model resolver`
// 2) If no cloud account has been specified, it falls back to the `simple billing model resolver`
func (k *kafkaBillingModelResolver) detectAmsBillingModel(orgId string, kafka *dbapi.KafkaRequest, kafkaBillingModel config.KafkaBillingModel) (string, error) {
	validAmsBillingModels := kafkaBillingModel.AMSBillingModels
	if kafka.BillingCloudAccountId != "" {
		validAmsBillingModels = arrays.Filter(validAmsBillingModels, arrays.StringHasPrefixIgnoreCasePredicate("marketplace-"))
		if len(validAmsBillingModels) == 0 {
			return "", errors.InsufficientQuotaError("marketplace is not supported by billing mode '%s'. Supported ams billing models are %v", kafka.Marketplace, kafkaBillingModel.ID, kafkaBillingModel.AMSBillingModels)
		}

		accountBillingModelResolver := cloudAccountBillingModelResolver{
			quotaConfigProvider: k.quotaConfigProvider,
			kafkaConfig:         k.kafkaConfig,
		}
		billingModelDetails, err := accountBillingModelResolver.resolve(orgId, kafka, kafkaBillingModel)
		if err != nil {
			return "", err
		}
		return billingModelDetails.AMSBillingModel, nil
	}

	var billingModelsToCheck []config.KafkaBillingModel
	if kafkaBillingModel.HasSupportForStandard() {
		standardBillingModel := kafkaBillingModel
		standardBillingModel.ID = "standard"
		standardBillingModel.AMSBillingModels = []string{"standard"}
		billingModelsToCheck = append(billingModelsToCheck, standardBillingModel)
	}
	if kafkaBillingModel.HasSupportForMarketplace() {
		marketplaceBillingModel := kafkaBillingModel
		marketplaceBillingModel.ID = "marketplace"
		marketplaceBillingModel.AMSBillingModels = []string{"marketplace"}
		billingModelsToCheck = append(billingModelsToCheck, marketplaceBillingModel)
	}

	resolver := simpleBillingModelResolver{
		quotaConfigProvider: k.quotaConfigProvider,
		kafkaConfig:         k.kafkaConfig,
	}

	res, err := resolver.resolve(orgId, kafka, billingModelsToCheck)
	if err != nil {
		return "", err
	}
	return res.AMSBillingModel, nil
}
