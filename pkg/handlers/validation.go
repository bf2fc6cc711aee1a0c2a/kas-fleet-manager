package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

var (
	// Kafka cluster names must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. For example, 'my-name', or 'abc-123'.
	validKafkaClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	minRequiredFieldLength      = 1
	maxKafkaNameLength          = 32
)

// validateAsyncEnabled returns a validator that returns an error if the async query param is not true
func validateAsyncEnabled(r *http.Request, action string) validate {
	return func() *errors.ServiceError {
		asyncParam := r.URL.Query().Get("async")
		if asyncParam != "true" {
			return errors.SyncActionNotSupported()
		}
		return nil
	}
}

// validateMultiAZEnabled returns a validator that returns an error if the multiAZ is not true
func validateMultiAZEnabled(value *bool, action string) validate {
	return func() *errors.ServiceError {
		if !*value {
			return errors.NotMultiAzActionNotSupported()
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
			return errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", kafkaRequest.CloudProvider, configService.GetSupportedProviders())
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
			return errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", kafkaRequest.Region, kafkaRequest.CloudProvider, provider.Regions)
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

		var allowListItem config.AllowedListItem

		orgId := auth.GetOrgIdFromContext(context)
		username := auth.GetUsernameFromContext(context)

		org, orgFound := configService.GetOrganisationById(orgId)
		var message string
		if orgFound {
			allowListItem = org
			message = fmt.Sprintf("Organisation '%s' has reached a maximum number of %d allowed instances.", orgId, org.GetMaxAllowedInstances())
		} else {
			user, _ := configService.GetServiceAccountByUsername(username)
			allowListItem = user
			message = fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances())
		}

		_, pageMeta, err := kafkaService.List(context, &services.ListArguments{Page: 1, Size: 1})
		if err != nil {
			return err
		}

		if !allowListItem.IsInstanceCountWithinLimit(pageMeta.Total) {
			return errors.MaximumAllowedInstanceReached(message)
		}

		return nil
	}
}

func validKafkaClusterName(value *string, field string) validate {
	return func() *errors.ServiceError {
		if !validKafkaClusterNameRegexp.MatchString(*value) {
			return errors.MalformedKafkaClusterName("%s does not match %s", field, validKafkaClusterNameRegexp.String())
		}
		return nil
	}
}

// validateKafkaClusterNameIsUnique returns a validator that validates that the kafka cluster name is unique
func validateKafkaClusterNameIsUnique(name *string, kafkaService services.KafkaService, context context.Context) validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := kafkaService.List(context, &services.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateKafkaClusterName()
		}

		return nil
	}
}

func validateLength(value *string, field string, minVal *int, maxVal *int) validate {
	var min = 1
	if *minVal > 1 {
		min = *minVal
	}
	return func() *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		if maxVal != nil && len(*value) > *maxVal {
			return errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", field, maxVal)
		}
		return nil
	}
}
