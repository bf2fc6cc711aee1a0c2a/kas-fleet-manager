package services

import (
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	truncatedNameLen = 10
)

// Field names suspected to contain personally identifiable information
var piiFields []string = []string{
	"username",
	"first_name",
	"last_name",
	"email",
	"address",
}

func handleGetError(resourceType, field string, value interface{}, err error) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	if gorm.IsRecordNotFoundError(err) {
		return errors.NotFound("%s with %s='%v' not found", resourceType, field, value)
	}
	return errors.GeneralError("Unable to find %s with %s='%v': %s", resourceType, field, value, err)
}

func handleCreateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("This %s already exists", resourceType)
	}
	return errors.GeneralError("Unable to create %s: %s", resourceType, err.Error())
}

func handleUpdateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("Changes to %s conflict with existing records", resourceType)
	}
	return errors.GeneralError("Unable to update %s: %s", resourceType, err.Error())
}

func handleDeleteError(resourceType string, err error) *errors.ServiceError {
	return errors.GeneralError("Unable to delete %s: %s", resourceType, err.Error())
}

func fields(obj interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	val := reflect.Indirect(reflect.ValueOf(obj))
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldByName := val.FieldByName(field.Name)
		if !fieldByName.IsNil() {
			m[field.Name] = fieldByName.Interface()
		}
	}

	return m
}

func truncateString(str string, num int) string {
	truncatedString := str
	if len(str) > num {
		truncatedString = str[0:num]
	}
	return truncatedString
}
