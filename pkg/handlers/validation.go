package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/xeipuuv/gojsonschema"

	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

var (
	// Kafka cluster names must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. For example, 'my-name', or 'abc-123'.
	validKafkaClusterNameRegexp   = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	validUuidRegexp               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	validServiceAccountNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	validServiceAccountDescRegexp = regexp.MustCompile(`^[a-zA-Z0-9\s]*$`)
	minRequiredFieldLength        = 1
	maxKafkaNameLength            = 32
	maxServiceAccountNameLength   = 50
	maxServiceAccountDescLength   = 255
	maxServiceAccountId           = 36
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

func validateServiceAccountName(value *string, field string) validate {
	return func() *errors.ServiceError {
		if !validServiceAccountNameRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountName("%s does not match %s", field, validServiceAccountNameRegexp.String())
		}
		return nil
	}
}

func validateServiceAccountDesc(value *string, field string) validate {
	return func() *errors.ServiceError {
		if !validServiceAccountDescRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountDesc("%s does not match %s", field, validServiceAccountDescRegexp.String())
		}
		return nil
	}
}

func validateServiceAccountId(value *string, field string) validate {
	return func() *errors.ServiceError {
		if !validUuidRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountId("%s does not match %s", field, validUuidRegexp.String())
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
		if !configService.GetConfig().AccessControlList.EnableInstanceLimitControl {
			return nil
		}

		var allowListItem config.AllowedListItem
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			return errors.Unauthenticated("user not authenticated")
		}

		orgId := auth.GetOrgIdFromClaims(claims)
		username := auth.GetUsernameFromClaims(claims)

		message := fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, config.GetDefaultMaxAllowedInstances())
		org, orgFound := configService.GetOrganisationById(orgId)
		if orgFound && org.IsUserAllowed(username) {
			allowListItem = org
			message = fmt.Sprintf("Organisation '%s' has reached a maximum number of %d allowed instances.", orgId, org.GetMaxAllowedInstances())
		} else {
			user, userFound := configService.GetServiceAccountByUsername(username)
			if userFound {
				allowListItem = user
				message = fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances())
			}
		}

		_, pageMeta, svcErr := kafkaService.List(context, &services.ListArguments{Page: 1, Size: 1})
		if svcErr != nil {
			return svcErr
		}

		maxInstanceReached := pageMeta.Total >= config.GetDefaultMaxAllowedInstances()
		// check instance limit for internal users (users and orgs listed in allow list config)
		if allowListItem != nil {
			maxInstanceReached = !allowListItem.IsInstanceCountWithinLimit(pageMeta.Total)
		}

		if maxInstanceReached {
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

func validateMaxLength(value *string, field string, maxVal *int) validate {
	return func() *errors.ServiceError {
		if maxVal != nil && len(*value) > *maxVal {
			return errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", field, *maxVal)
		}
		return nil
	}
}

func validateLength(value *string, field string, minVal *int, maxVal *int) validate {
	var min = 1
	if *minVal > 1 {
		min = *minVal
	}
	resp := validateMaxLength(value, field, maxVal)
	if resp != nil {
		return resp
	}
	return validateMinLength(value, field, min)
}

func validateMinLength(value *string, field string, min int) validate {
	return func() *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		return nil
	}
}

func validateJsonSchema(schemaName string, schemaLoader gojsonschema.JSONLoader, documentName string, documentLoader gojsonschema.JSONLoader) *errors.ServiceError {
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", schemaName, err)
	}

	r, err := schema.Validate(documentLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", documentName, err)
	}
	if !r.Valid() {
		return errors.BadRequest("%s not conform to the %s. %d errors encountered.  1st error: %s",
			documentName, schemaName, len(r.Errors()), r.Errors()[0].String())
	}
	return nil
}

func validatQueryParam(queryParams url.Values, field string) validate {

	return func() *errors.ServiceError {
		fieldValue := queryParams.Get(field)
		if fieldValue == "" {
			return errors.FailedToParseQueryParms("bad request, cannot parse query parameter '%s' '%s'", field, fieldValue)
		}
		if _, err := strconv.ParseInt(fieldValue, 10, 64); err != nil {
			return errors.FailedToParseQueryParms("bad request, cannot parse query parameter '%s' '%s'", field, fieldValue)
		}

		return nil
	}

}
