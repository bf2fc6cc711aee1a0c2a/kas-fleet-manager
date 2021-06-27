package handlers

import (
	"net/http"
	"regexp"

	"github.com/xeipuuv/gojsonschema"

	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

var (
	// Kafka cluster names must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. For example, 'my-name', or 'abc-123'.

	ValidUuidRegexp               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ValidServiceAccountNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	ValidServiceAccountDescRegexp = regexp.MustCompile(`^[a-zA-Z0-9.,\-\s]*$`)
	MinRequiredFieldLength        = 1

	MaxServiceAccountNameLength = 50
	MaxServiceAccountDescLength = 255
	MaxServiceAccountId         = 36
)

// ValidateAsyncEnabled returns a validator that returns an error if the async query param is not true
func ValidateAsyncEnabled(r *http.Request, action string) Validate {
	return func() *errors.ServiceError {
		asyncParam := r.URL.Query().Get("async")
		if asyncParam != "true" {
			return errors.SyncActionNotSupported()
		}
		return nil
	}
}

func ValidateServiceAccountName(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidServiceAccountNameRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountName("%s does not match %s", field, ValidServiceAccountNameRegexp.String())
		}
		return nil
	}
}

func ValidateServiceAccountDesc(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidServiceAccountDescRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountDesc("%s does not match %s", field, ValidServiceAccountDescRegexp.String())
		}
		return nil
	}
}

func ValidateServiceAccountId(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidUuidRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountId("%s does not match %s", field, ValidUuidRegexp.String())
		}
		return nil
	}
}

// ValidateMultiAZEnabled returns a validator that returns an error if the multiAZ is not true
func ValidateMultiAZEnabled(value *bool, action string) Validate {
	return func() *errors.ServiceError {
		if !*value {
			return errors.NotMultiAzActionNotSupported()
		}
		return nil
	}
}

// ValidateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func ValidateCloudProvider(kafkaRequest *openapi.KafkaRequestPayload, configService services.ConfigService, action string) Validate {
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

func ValidateMaxLength(value *string, field string, maxVal *int) Validate {
	return func() *errors.ServiceError {
		if maxVal != nil && len(*value) > *maxVal {
			return errors.MaximumFieldLengthMissing("%s is not valid. Maximum length %d is required", field, *maxVal)
		}
		return nil
	}
}

func ValidateLength(value *string, field string, minVal *int, maxVal *int) Validate {
	var min = 1
	if *minVal > 1 {
		min = *minVal
	}
	resp := ValidateMaxLength(value, field, maxVal)
	if resp != nil {
		return resp
	}
	return ValidateMinLength(value, field, min)
}

func ValidateMinLength(value *string, field string, min int) Validate {
	return func() *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		return nil
	}
}

func ValidateJsonSchema(schemaName string, schemaLoader gojsonschema.JSONLoader, documentName string, documentLoader gojsonschema.JSONLoader) *errors.ServiceError {
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

func ValidatQueryParam(queryParams url.Values, field string) Validate {

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
