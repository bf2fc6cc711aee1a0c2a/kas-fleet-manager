package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
)

func PresentKafkaRequestAdminEndpoint(kafkaRequest *dbapi.KafkaRequest, accountService account.AccountService) (*private.Kafka, *errors.ServiceError) {
	reference := PresentReference(kafkaRequest.ID, kafkaRequest)

	org, err := accountService.GetOrganization(fmt.Sprintf("external_id='%s'", kafkaRequest.OrganisationId))

	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error presenting the request")
	}

	if org == nil {
		return nil, errors.New(errors.ErrorGeneral, "unable to find an organisation for external_id '%s'", kafkaRequest.OrganisationId)
	}

	return &private.Kafka{
		Id:                     reference.Id,
		Kind:                   reference.Kind,
		Href:                   reference.Href,
		Status:                 kafkaRequest.Status,
		CloudProvider:          kafkaRequest.CloudProvider,
		MultiAz:                kafkaRequest.MultiAZ,
		Region:                 kafkaRequest.Region,
		Owner:                  kafkaRequest.Owner,
		Name:                   kafkaRequest.Name,
		BootstrapServerHost:    setBootstrapServerHost(kafkaRequest.BootstrapServerHost),
		CreatedAt:              kafkaRequest.CreatedAt,
		UpdatedAt:              kafkaRequest.UpdatedAt,
		FailedReason:           kafkaRequest.FailedReason,
		DesiredKafkaVersion:    kafkaRequest.DesiredKafkaVersion,
		ActualKafkaVersion:     kafkaRequest.ActualKafkaVersion,
		DesiredStrimziVersion:  kafkaRequest.DesiredStrimziVersion,
		ActualStrimziVersion:   kafkaRequest.ActualStrimziVersion,
		DesiredKafkaIbpVersion: kafkaRequest.DesiredKafkaIBPVersion,
		ActualKafkaIbpVersion:  kafkaRequest.ActualKafkaIBPVersion,
		KafkaUpgrading:         kafkaRequest.KafkaUpgrading,
		StrimziUpgrading:       kafkaRequest.StrimziUpgrading,
		KafkaIbpUpgrading:      kafkaRequest.KafkaIBPUpgrading,
		KafkaStorageSize:       kafkaRequest.KafkaStorageSize,
		OrganisationId:         kafkaRequest.OrganisationId,
		SubscriptionId:         kafkaRequest.SubscriptionId,
		SsoClientId:            kafkaRequest.SsoClientID,
		OwnerAccountId:         kafkaRequest.OwnerAccountId,
		AccountNumber:          org.AccountNumber,
		QuotaType:              kafkaRequest.QuotaType,
		Routes:                 GetRoutesFromKafkaRequest(kafkaRequest),
		RoutesCreated:          kafkaRequest.RoutesCreated,
		ClusterId:              kafkaRequest.ClusterID,
		InstanceType:           kafkaRequest.InstanceType,
		Namespace:              kafkaRequest.Namespace,
		SizeId:                 kafkaRequest.SizeId,
	}, nil
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
