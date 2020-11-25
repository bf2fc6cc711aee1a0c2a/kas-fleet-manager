package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
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
)

func ObjectKind(i interface{}) string {
	switch i.(type) {
	case api.KafkaRequest, *api.KafkaRequest:
		return KindKafka
	case api.CloudRegion, *api.CloudRegion:
		return KindCloudRegion
	case api.CloudProvider, *api.CloudProvider:
		return KindCloudProvider
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	default:
		return ""
	}
}
