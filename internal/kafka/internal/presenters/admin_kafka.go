package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
)

func PresentKafkaRequestAdminEndpoint(kafkaRequest *dbapi.KafkaRequest) private.Kafka {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	return private.Kafka{
		Id:                    reference.Id,
		Kind:                  reference.Kind,
		Href:                  reference.Href,
		Status:                kafkaRequest.Status,
		CloudProvider:         kafkaRequest.CloudProvider,
		MultiAz:               kafkaRequest.MultiAZ,
		Region:                kafkaRequest.Region,
		Owner:                 kafkaRequest.Owner,
		Name:                  kafkaRequest.Name,
		BootstrapServerHost:   setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		CreatedAt:             kafkaRequest.CreatedAt,
		UpdatedAt:             kafkaRequest.UpdatedAt,
		FailedReason:          kafkaRequest.FailedReason,
		DesiredKafkaVersion:   kafkaRequest.DesiredKafkaVersion,
		ActualKafkaVersion:    kafkaRequest.ActualKafkaVersion,
		DesiredStrimziVersion: kafkaRequest.DesiredStrimziVersion,
		ActualStrimziVersion:  kafkaRequest.ActualStrimziVersion,
		KafkaUpgrading:        kafkaRequest.KafkaUpgrading,
		StrimziUpgrading:      kafkaRequest.StrimziUpgrading,
		OrganisationId:        kafkaRequest.OrganisationId,
		SubscriptionId:        kafkaRequest.SubscriptionId,
		SsoClientId:           kafkaRequest.SsoClientID,
		OwnerAccountId:        kafkaRequest.OwnerAccountId,
		QuotaType:             kafkaRequest.QuotaType,
		Routes:                GetRoutesFromKafkaRequest(kafkaRequest),
		RoutesCreated:         kafkaRequest.RoutesCreated,
		ClusterId:             kafkaRequest.ClusterID,
		// ProductType - TODO once this column is available in the database
	}
}

func GetRoutesFromKafkaRequest(kafkaRequest *dbapi.KafkaRequest) []private.KafkaAllOfRoutes {
	var routes []private.KafkaAllOfRoutes
	routesArray, err := kafkaRequest.GetRoutes()
	if err != nil {
		return routes
	} else {
		for _, r := range routesArray {
			routes = append(routes, private.KafkaAllOfRoutes{Domain: r.Domain, Router: r.Router})
		}
		return routes
	}
}
