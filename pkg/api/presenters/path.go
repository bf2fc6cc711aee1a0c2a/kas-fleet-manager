package presenters

import (
	"fmt"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	BasePath = "/api/managed-services-api/v1"
)

func ObjectPath(id string, obj interface{}) string {
	return fmt.Sprintf("%s/%s/%s", BasePath, path(obj), id)
}

func path(i interface{}) string {
	switch i.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return "kafkas"
	case errors.ServiceError, *errors.ServiceError:
		return "errors"
	default:
		return ""
	}
}
