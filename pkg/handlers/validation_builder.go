package handlers

import (
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
)

type ValidateOption func(field string, value *string) *errors.ServiceError

func Validation(field string, value *string, options ...ValidateOption) Validate {
	return func() *errors.ServiceError {
		for _, option := range options {
			err := option(field, value)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func WithDefault(d string) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if *value == "" {
			*value = d
		}
		return nil
	}
}

func MinLen(min int) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		return nil
	}
}

func MaxLen(max int) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value != nil && len(*value) > max {
			return errors.MaximumFieldLengthExceeded("%s is not valid. Maximum length %d is required.", field, max)
		}
		return nil
	}
}

func IsOneOf(options ...string) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value != nil &&
			arrays.Contains(options, *value) {
			return nil
		}
		return errors.BadRequest("%s is not valid. Must be one of: %s", field, strings.Join(options, ", "))
	}
}

func Matches(regex *regexp.Regexp) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value == nil || len(*value) == 0 {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, 1)
		}
		if value != nil && !regex.MatchString(*value) {
			return errors.BadRequest("%s is not valid. Must match regex: %s", field, regex.String())
		}
		return nil
	}
}
