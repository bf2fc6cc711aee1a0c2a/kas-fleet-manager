package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

const (
	// KindKafka is a string identifier for the type api.KafkaRequest
	KindKafka = "Kafka"
	// CloudRegion is a string identifier for the type api.CloudRegion
	KindCloudRegion = "CloudRegion"
	// KindCloudProvider is a string identifier for the type api.CloudProvider
	KindCloudProvider = "CloudProvider"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"
	// KindServiceAccount is a string identifier for the type api.ServiceAccount
	KindServiceAccount = "ServiceAccount"

	KindCluster = "Cluster"

	// KindClusterAddonParameters is a string identifier for the
	// type public.EnterpriseClusterAddonParameters
	KindClusterAddonParameters = "ClusterAddonParameters"

	BasePath = "/api/kafkas_mgmt/v1"
)

func PresentReference(id, obj interface{}) compat.ObjectReference {
	return handlers.PresentReferenceWith(id, obj, objectKind, objectPath)
}

func objectKind(i interface{}) string {
	switch i.(type) {
	case dbapi.KafkaRequest, *dbapi.KafkaRequest:
		return KindKafka
	case api.CloudRegion, *api.CloudRegion:
		return KindCloudRegion
	case api.CloudProvider, *api.CloudProvider:
		return KindCloudProvider
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	case api.ServiceAccount, *api.ServiceAccount:
		return KindServiceAccount
	case api.Cluster, *api.Cluster:
		return KindCluster
	case public.EnterpriseClusterAddonParameters, *public.EnterpriseClusterAddonParameters:
		return KindClusterAddonParameters
	default:
		return ""
	}
}

func objectPath(id string, obj interface{}) string {
	switch obj.(type) {
	case dbapi.KafkaRequest, *dbapi.KafkaRequest:
		return fmt.Sprintf("%s/kafkas/%s", BasePath, id)
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	case api.Cluster, *api.Cluster:
		return fmt.Sprintf("%s/clusters/%s", BasePath, id)
	case api.ServiceAccount, *api.ServiceAccount:
		return fmt.Sprintf("%s/service_accounts/%s", BasePath, id)
	case public.EnterpriseClusterAddonParameters, *public.EnterpriseClusterAddonParameters:
		return fmt.Sprintf("%s/clusters/%s/addon_parameters", BasePath, id)
	default:
		return ""
	}
}
