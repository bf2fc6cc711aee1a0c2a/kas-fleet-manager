package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
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
func validateCloudProvider(kafkaRequest *openapi.KafkaRequestPayload, configService services.ConfigService, action string) validate {
	return func() *errors.ServiceError {
		// Set Cloud Provider default if not received in the request
		if kafkaRequest.CloudProvider == "" {
			defaultProvider, _ := configService.GetDefaultProvider()
			kafkaRequest.CloudProvider = defaultProvider.Name
		}

		// Validation for Cloud Provider
		provider, providerSupported := configService.GetSupportedProviders().GetByName(kafkaRequest.CloudProvider)
		if !providerSupported {
			return errors.Validation("provider %s is not supported, supported providers are: %s", kafkaRequest.CloudProvider, configService.GetSupportedProviders())
		}

		// Set Cloud Region default if not received in the request
		if kafkaRequest.Region == "" {
			defaultRegion, _ := configService.GetDefaultRegionForProvider(provider)
			kafkaRequest.Region = defaultRegion.Name
		}

		// Validation for Cloud Region
		regionSupported := configService.IsRegionSupportedForProvider(kafkaRequest.CloudProvider, kafkaRequest.Region)
		if !regionSupported {
			provider, _ := configService.GetSupportedProviders().GetByName(kafkaRequest.CloudProvider)
			return errors.Validation("region %s is not supported for %s, supported regions are: %s", kafkaRequest.Region, kafkaRequest.CloudProvider, provider.Regions)
		}

		return nil
	}
}

// validateMaxAllowedInstances returns a validator that validate that the current connected user has not
// reached the max number of allowed instances that can be created for this user if any or the global max allowed defaults
func validateMaxAllowedInstances(kafkaService services.KafkaService, configService services.ConfigService, context context.Context) validate {
	return func() *errors.ServiceError {
		if !configService.IsAllowListEnabled() {
			return nil
		}

		username, orgId := auth.GetUsernameFromContext(context), auth.GetOrgIdFromContext(context)
		user, _ := configService.GetAllowedUserByUsernameAndOrgId(username, orgId)

		_, pageMeta, err := kafkaService.List(context, &services.ListArguments{Page: 1, Size: 1})
		if err != nil {
			return err
		}

		if !user.IsInstanceCountWithinLimit(pageMeta.Total) {
			return errors.Forbidden(fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances()))
		}

		return nil
	}
}
