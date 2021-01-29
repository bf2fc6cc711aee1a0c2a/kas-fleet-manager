package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

const (
	BasePath = "/api/managed-services-api/v1"
)

func ObjectPath(id string, obj interface{}) string {
	return fmt.Sprintf("%s/%s/%s", BasePath, path(obj), id)
}

func path(i interface{}) string {
	switch i := i.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return "kafkas"
	case api.Connector:
		return fmt.Sprintf("kafkas/%s/connector-deployments", i.KafkaID)
	case *api.Connector:
		return fmt.Sprintf("kafkas/%s/connector-deployments", i.KafkaID)
	case api.ConnectorType, *api.ConnectorType:
		return "connector-types"
	case errors.ServiceError, *errors.ServiceError:
		return "errors"
	case api.ServiceAccount, *api.ServiceAccount:
		return "serviceaccounts"
	default:
		return ""
	}
}
