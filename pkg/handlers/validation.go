package handlers

import (
	"fmt"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"net/http"
	"strings"

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
func validateCloudProvider(kr *openapi.KafkaRequest, configService services.ConfigService, action string) validate {
	return func() *errors.ServiceError {
		defaultProvider, err := configService.GetDefaultProvider()
		if err != nil {
			return errors.GeneralError("failed to get default provider configuration")
		}

		if kr.CloudProvider == "" {
			kr.CloudProvider = defaultProvider.Name
		}
		if kr.Region == "" {
			defaultRegion, err := configService.GetDefaultRegionForProvider(defaultProvider)
			if err != nil {
				return errors.GeneralError(fmt.Sprintf("failed to get default region for provider %s", defaultProvider.Name))
			}
			kr.Region = defaultRegion.Name
		}

		providerSupported := configService.IsProviderSupported(kr.CloudProvider)
		if !providerSupported {
			return errors.Validation("provider %s is not supported, supported providers are: %s", kr.CloudProvider, configService.GetSupportedProviders())
		}

		regionSupported := configService.IsRegionSupportedForProvider(kr.CloudProvider, kr.Region)
		if !regionSupported {
			provider, _ := configService.GetSupportedProviders().GetByName(kr.CloudProvider)
			return errors.Validation("region %s is not supported for %s, supported regions are: %s", kr.Region, kr.CloudProvider, provider.Regions)
		}

		return nil
	}
}
