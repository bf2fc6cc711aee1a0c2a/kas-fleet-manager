package services

import (
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"gorm.io/gorm"
)

// Field names suspected to contain personally identifiable information
var piiFields []string = []string{
	"username",
	"first_name",
	"last_name",
	"email",
	"address",
}

func HandleGetError(resourceType, field string, value interface{}, err error) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	if IsRecordNotFoundError(err) {
		return errors.NotFound("%s with %s='%v' not found", resourceType, field, value)
	}
	return errors.NewWithCause(errors.ErrorGeneral, err, "unable to find %s with %s='%v'", resourceType, field, value)
}

func HandleGoneError(resourceType, field string, value interface{}) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	return errors.New(errors.ErrorGone, "%s with %s='%v' has been deleted", resourceType, field, value)
}

func HandleDeleteError(resourceType string, field string, value interface{}, err error) *errors.ServiceError {
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete %s with %s='%v'", resourceType, field, value)
}

func IsRecordNotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func HandleCreateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("this %s already exists", resourceType)
	}
	return errors.GeneralError("unable to create %s: %s", resourceType, err.Error())
}

func HandleUpdateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("changes to %s conflict with existing records", resourceType)
	}
	return errors.GeneralError("unable to update %s: %s", resourceType, err.Error())
}
