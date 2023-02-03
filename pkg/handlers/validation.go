package handlers

import (
	"net/http"
	"regexp"

	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

var (
	// Kafka cluster names must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. For example, 'my-name', or 'abc-123'.

	ValidUuidRegexp               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ValidClientIdUuidRegexp       = regexp.MustCompile(`^srvc-acct-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ValidServiceAccountNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	ValidServiceAccountDescRegexp = regexp.MustCompile(`^[a-zA-Z0-9.,\-\s]*$`)
	ValidAlphaNumeric             = regexp.MustCompile(`^[a-zA-Z0-9]*$`)
	// taken from here: https://regex101.com/r/SEg6KL/1 - will likely be removed if we can use our permissions to get cluster dns from cluster id
	ValidDnsName           = regexp.MustCompile(`^(?:[_a-z0-9](?:[_a-z0-9-]{0,61}[a-z0-9])?\.)+(?:[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?)?$`)
	MinRequiredFieldLength = 1

	MaxServiceAccountNameLength = 50
	MaxServiceAccountDescLength = 255
	MaxServiceAccountId         = 36
	MaxServiceAccountClientId   = 47
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

func ValidateExternalClusterId(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidUuidRegexp.MatchString(*value) {
			return errors.InvalidExternalClusterId("%s does not match %s", field, ValidUuidRegexp.String())
		}
		return nil
	}
}

func ValidateNotEmptyClusterId(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if value != nil && len(*value) > 0 && !ValidAlphaNumeric.MatchString(*value) {
			return errors.InvalidClusterId("%s does not match %s", field, ValidUuidRegexp.String())
		}
		return nil
	}
}

func ValidateDnsName(value *string, field string) Validate {
	return func() *errors.ServiceError {
		if !ValidDnsName.MatchString(*value) {
			return errors.InvalidDnsName("%s does not match %s", field, ValidDnsName.String())
		}
		return nil
	}
}

func ValidateServiceAccountClientId(value *string, field string, ssoProvider string) Validate {
	return func() *errors.ServiceError {
		if ssoProvider == keycloak.REDHAT_SSO {
			// only service accounts from mas sso are prefixed with "srvc-acc-", always return nil for redhat_sso provider
			return nil
		}
		if !ValidClientIdUuidRegexp.MatchString(*value) {
			return errors.MalformedServiceAccountId("%s does not match %s", field, ValidClientIdUuidRegexp.String())
		}
		return nil
	}
}

func ValidateMaxLength(value *string, field string, maxVal *int) Validate {
	return func() *errors.ServiceError {
		if maxVal != nil && len(*value) > *maxVal {
			return errors.MaximumFieldLengthExceeded("%s is not valid, maximum length %d is required", field, *maxVal)
		}
		return nil
	}
}

func ValidateLength(value *string, field string, min int, maxVal *int) Validate {
	if min < 1 {
		min = 1
	}
	return func() *errors.ServiceError {
		err := ValidateMaxLength(value, field, maxVal)()
		if err != nil {
			return err
		}
		err = ValidateMinLength(value, field, min)()
		if err != nil {
			return err
		}
		return nil
	}
}

func ValidateMinLength(value *string, field string, min int) Validate {
	return func() *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid, minimum length %d is required", field, min)
		}
		return nil
	}
}

func ValidateQueryParam(queryParams url.Values, field string) Validate {

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
