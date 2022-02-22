package presenters

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

const (
	// KindConnector is a string identifier for the type dbapi.Connector
	KindConnector = "Connector"
	// KindConnectorCluster is a string identifier for the type dbapi.ConnectorCluster
	KindConnectorCluster = "ConnectorCluster"
	// KindConnectorDeployment is a string identifier for the type dbapi.ConnectorDeployment
	KindConnectorDeployment = "ConnectorDeployment"
	// KindConnectorNamespace is a string identifier for the type dbapi.ConnectorNamespace
	KindConnectorNamespace = "ConnectorNamespace"
	// KindConnectorType is a string identifier for the type dbapi.ConnectorType
	KindConnectorType = "ConnectorType"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"
)

func PresentReference(id, obj interface{}) compat.ObjectReference {
	return handlers.PresentReferenceWith(id, obj, objectKind, objectPath)
}

func objectKind(i interface{}) string {
	switch i.(type) {
	case dbapi.Connector, *dbapi.Connector:
		return KindConnector
	case dbapi.ConnectorCluster, *dbapi.ConnectorCluster:
		return KindConnectorCluster
	case dbapi.ConnectorDeployment, *dbapi.ConnectorDeployment:
		return KindConnectorDeployment
	case dbapi.ConnectorNamespace, *dbapi.ConnectorNamespace:
		return KindConnectorNamespace
	case dbapi.ConnectorType, *dbapi.ConnectorType:
		return KindConnectorType
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	default:
		return ""
	}
}

func objectPath(id string, obj interface{}) string {
	switch obj := obj.(type) {
	case dbapi.Connector, *dbapi.Connector:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connectors/%s", id)
	case dbapi.ConnectorType, *dbapi.ConnectorType:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_types/%s", id)
	case dbapi.ConnectorCluster, *dbapi.ConnectorCluster:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s", id)
	case dbapi.ConnectorDeployment:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s/deployments/%s", obj.ClusterID, id)
	case *dbapi.ConnectorDeployment:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_clusters/%s/deployments/%s", obj.ClusterID, id)
	case dbapi.ConnectorNamespace, *dbapi.ConnectorNamespace:
		return fmt.Sprintf("/api/connector_mgmt/v1/kafka_connector_namespaces/%s", id)
	default:
		return ""
	}
}
