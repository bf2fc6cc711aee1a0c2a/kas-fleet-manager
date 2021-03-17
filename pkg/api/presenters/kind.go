package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

const (
	// KindKafka is a string identifier for the type api.KafkaRequest
	KindKafka = "Kafka"
	// CloudRegion is a string identifier for the type api.CloudRegion
	KindCloudRegion = "CloudRegion"
	// KindCloudProvider is a string identifier for the type api.CloudProvider
	KindCloudProvider = "CloudProvider"
	// KindConnector is a string identifier for the type api.Connector
	KindConnector = "Connector"
	// KindConnectorCluster is a string identifier for the type api.ConnectorCluster
	KindConnectorCluster = "ConnectorCluster"
	// KindConnectorDeployment is a string identifier for the type api.ConnectorDeployment
	KindConnectorDeployment = "ConnectorDeployment"
	// KindConnectorType is a string identifier for the type api.ConnectorType
	KindConnectorType = "ConnectorType"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"
)

func ObjectKind(i interface{}) string {
	switch i.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return KindKafka
	case api.CloudRegion, *api.CloudRegion:
		return KindCloudRegion
	case api.CloudProvider, *api.CloudProvider:
		return KindCloudProvider
	case api.Connector, *api.Connector:
		return KindConnector
	case api.ConnectorCluster, *api.ConnectorCluster:
		return KindConnectorCluster
	case api.ConnectorDeployment, *api.ConnectorDeployment:
		return KindConnectorDeployment
	case api.ConnectorType, *api.ConnectorType:
		return KindConnectorType
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	default:
		return ""
	}
}
