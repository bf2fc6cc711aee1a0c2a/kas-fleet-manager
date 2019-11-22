package presenters

import (
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/errors"
)

func ObjectKind(i interface{}) string {
	switch i.(type) {
	case api.Dinosaur, *api.Dinosaur:
		return "Dinosaur"
	case errors.ServiceError, *errors.ServiceError:
		return "Error"
	default:
		return ""
	}
}
