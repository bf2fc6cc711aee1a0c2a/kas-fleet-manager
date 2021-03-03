package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"strings"
)

type ValidateOption func(field string, value *string) *errors.ServiceError

func validation(field string, value *string, options ...ValidateOption) validate {
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

func withDefault(d string) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if *value == "" {
			*value = d
		}
		return nil
	}
}

func minLen(min int) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value == nil || len(*value) < min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Minimum length %d is required.", field, min)
		}
		return nil
	}
}
func maxLen(min int) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value != nil && len(*value) > min {
			return errors.MinimumFieldLengthNotReached("%s is not valid. Maximum length %d is required.", field, min)
		}
		return nil
	}
}

func isOneOf(options ...string) ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if value != nil {
			for _, option := range options {
				if *value == option {
					return nil
				}
			}
		}
		return errors.MinimumFieldLengthNotReached("%s is not valid. Must be one of: %s", field, strings.Join(options, ","))
	}
}
