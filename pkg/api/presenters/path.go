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
	switch obj := obj.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return fmt.Sprintf("%s/kafkas/%s", BasePath, id)
	case api.Connector, *api.Connector:
		return fmt.Sprintf("%s/kafka-connectors/%s", BasePath, id)
	case api.ConnectorType, *api.ConnectorType:
		return fmt.Sprintf("%s/kafka-connector-types/%s", BasePath, id)
	case api.ConnectorCluster, *api.ConnectorCluster:
		return fmt.Sprintf("%s/kafka-connector-clusters/%s", BasePath, id)
	case api.ConnectorDeployment:
		return fmt.Sprintf("%s/kafka-connector-clusters/%s/deployments/%s", BasePath, obj.ClusterID, id)
	case *api.ConnectorDeployment:
		return fmt.Sprintf("%s/kafka-connector-clusters/%s/deployments/%s", BasePath, obj.ClusterID, id)
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	case api.ServiceAccount, *api.ServiceAccount:
		return fmt.Sprintf("%s/serviceaccounts/%s", BasePath, id)
	default:
		return ""
	}
}
