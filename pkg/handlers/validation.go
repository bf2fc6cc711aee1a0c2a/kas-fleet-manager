package handlers

import (
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
