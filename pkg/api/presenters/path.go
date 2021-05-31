package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

const (
	BasePath = "/api/kafkas_mgmt/v1"
)

func ObjectPath(id string, obj interface{}) string {
	switch obj := obj.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return fmt.Sprintf("%s/kafkas/%s", BasePath, id)
	case api.Connector, *api.Connector:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connectors/%s", id)
	case api.ConnectorType, *api.ConnectorType:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_types/%s", id)
	case api.ConnectorCluster, *api.ConnectorCluster:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s", id)
	case api.ConnectorDeployment:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s/deployments/%s", obj.ClusterID, id)
	case *api.ConnectorDeployment:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s/deployments/%s", obj.ClusterID, id)
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	case api.ServiceAccount, *api.ServiceAccount:
		return fmt.Sprintf("%s/service_accounts/%s", BasePath, id)
	default:
		return ""
	}
}
