package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
)

func PresentKafkaRequestAdminEndpoint(kafkaRequest *dbapi.KafkaRequest) private.Kafka {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	return private.Kafka{
		Id:                  reference.Id,
		Kind:                reference.Kind,
		Href:                reference.Href,
		Status:              kafkaRequest.Status,
		CloudProvider:       kafkaRequest.CloudProvider,
		MultiAz:             kafkaRequest.MultiAZ,
		Region:              kafkaRequest.Region,
		Owner:               kafkaRequest.Owner,
		Name:                kafkaRequest.Name,
		BootstrapServerHost: setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		CreatedAt:           kafkaRequest.CreatedAt,
		UpdatedAt:           kafkaRequest.UpdatedAt,
		FailedReason:        kafkaRequest.FailedReason,
		KafkaVersion:        kafkaRequest.Version,
		// StrimziVersion
		OrganisationId: kafkaRequest.OrganisationId,
		SubscriptionId: kafkaRequest.SubscriptionId,
		SsoClientId:    kafkaRequest.SsoClientID,
		OwnerAccountId: kafkaRequest.OwnerAccountId,
		QuotaType:      kafkaRequest.QuotaType,
		// Routes:         kafkaRequest.Routes.MarshalJSON(),
		// RoutesCreated
		// ProductType
	}
}
