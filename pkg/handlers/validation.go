package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

func validateNotEmpty(value *string, field string) validate {
	return func() *errors.ServiceError {
		if value == nil || len(*value) == 0 {
			return errors.Validation("%s is required", field)
		}
		return nil
	}
}

func validateEmpty(value *string, field string) validate {
	return func() *errors.ServiceError {
		if len(*value) > 0 {
			return errors.Validation("%s must be empty", field)
		}
		return nil
	}
}

func validateInclusionIn(value *string, list []string) validate {
	return func() *errors.ServiceError {
		for _, item := range list {
			if strings.EqualFold(*value, item) {
				return nil
			}
		}
		return errors.Validation("%s is not a valid value", *value)
	}
}

func validateNonNegative(value *int32, field string) validate {
	return func() *errors.ServiceError {
		if (*value) < 0 {
			return errors.Validation("%s must be non-negative", field)
		}
		return nil
	}
}

// validateAsyncEnabled returns a validator that returns an error if the async query param is not true
func validateAsyncEnabled(r *http.Request, action string) validate {
	return func() *errors.ServiceError {
		asyncParam := r.URL.Query().Get("async")
		if asyncParam != "true" {
			return errors.SyncActionNotSupported(action)
		}
		return nil
	}
}

// validateMultiAZEnabled returns a validator that returns an error if the multiAZ is not true
func validateMultiAZEnabled(value *bool, action string) validate {
	return func() *errors.ServiceError {
		if !*value {
			return errors.NotMultiAzActionNotSupported(action)
		}
		return nil
	}
}

// validateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func validateCloudProvider(kafkaRequest *openapi.KafkaRequest, configService services.ConfigService, action string) validate {
	return func() *errors.ServiceError {
		var provider config.Provider
		if kafkaRequest.CloudProvider == "" {
			var err error
			provider, err = configService.GetDefaultProvider()
			if err != nil {
				return errors.GeneralError("failed to get default provider configuration")
			}
			kafkaRequest.CloudProvider = provider.Name
		} else {
			var providerSupported bool
			provider, providerSupported = configService.GetSupportedProviders().GetByName(kafkaRequest.CloudProvider)
			if !providerSupported {
				return errors.Validation("provider %s is not supported, supported providers are: %s", kafkaRequest.CloudProvider, configService.GetSupportedProviders())
			}
		}

		if kafkaRequest.Region == "" {
			defaultRegionForProvider, err := configService.GetDefaultRegionForProvider(provider)
			if err != nil {
				return errors.GeneralError(fmt.Sprintf("failed to get default region for provider %s", provider.Name))
			}
			kafkaRequest.Region = defaultRegionForProvider.Name
		}

		regionSupported := configService.IsRegionSupportedForProvider(kafkaRequest.CloudProvider, kafkaRequest.Region)
		if !regionSupported {
			return errors.Validation("region %s is not supported for %s, supported regions are: %s", kafkaRequest.Region, kafkaRequest.CloudProvider, provider.Regions)
		}

		return nil
	}
}
