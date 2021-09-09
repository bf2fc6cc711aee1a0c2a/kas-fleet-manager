package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/account"
)

func PresentDinosaurRequestAdminEndpoint(dinosaurRequest *dbapi.DinosaurRequest, accountService account.AccountService) (*private.Dinosaur, *errors.ServiceError) {
	reference := PresentReference(dinosaurRequest.ID, dinosaurRequest)

	org, err := accountService.GetOrganization(fmt.Sprintf("external_id='%s'", dinosaurRequest.OrganisationId))

	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error presenting the request")
	}

	if org == nil {
		return nil, errors.New(errors.ErrorGeneral, "unable to find an organisation for external_id '%s'", dinosaurRequest.OrganisationId)
	}

	return &private.Dinosaur{
		Id:                        reference.Id,
		Kind:                      reference.Kind,
		Href:                      reference.Href,
		Status:                    dinosaurRequest.Status,
		CloudProvider:             dinosaurRequest.CloudProvider,
		MultiAz:                   dinosaurRequest.MultiAZ,
		Region:                    dinosaurRequest.Region,
		Owner:                     dinosaurRequest.Owner,
		Name:                      dinosaurRequest.Name,
		BootstrapServerHost:       setBootstrapServerHost(dinosaurRequest.BootstrapServerHost),
		CreatedAt:                 dinosaurRequest.CreatedAt,
		UpdatedAt:                 dinosaurRequest.UpdatedAt,
		FailedReason:              dinosaurRequest.FailedReason,
		DesiredDinosaurVersion:    dinosaurRequest.DesiredDinosaurVersion,
		ActualDinosaurVersion:     dinosaurRequest.ActualDinosaurVersion,
		DesiredStrimziVersion:     dinosaurRequest.DesiredStrimziVersion,
		ActualStrimziVersion:      dinosaurRequest.ActualStrimziVersion,
		DesiredDinosaurIbpVersion: dinosaurRequest.DesiredDinosaurIBPVersion,
		ActualDinosaurIbpVersion:  dinosaurRequest.ActualDinosaurIBPVersion,
		DinosaurUpgrading:         dinosaurRequest.DinosaurUpgrading,
		StrimziUpgrading:          dinosaurRequest.StrimziUpgrading,
		DinosaurIbpUpgrading:      dinosaurRequest.DinosaurIBPUpgrading,
		OrganisationId:            dinosaurRequest.OrganisationId,
		SubscriptionId:            dinosaurRequest.SubscriptionId,
		SsoClientId:               dinosaurRequest.SsoClientID,
		OwnerAccountId:            dinosaurRequest.OwnerAccountId,
		AccountNumber:             org.AccountNumber,
		QuotaType:                 dinosaurRequest.QuotaType,
		Routes:                    GetRoutesFromDinosaurRequest(dinosaurRequest),
		RoutesCreated:             dinosaurRequest.RoutesCreated,
		ClusterId:                 dinosaurRequest.ClusterID,
		InstanceType:              dinosaurRequest.InstanceType,
		Namespace:                 dinosaurRequest.Namespace,
	}, nil
}

func GetRoutesFromDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) []private.DinosaurAllOfRoutes {
	var routes []private.DinosaurAllOfRoutes
	routesArray, err := dinosaurRequest.GetRoutes()
	if err != nil {
		return routes
	} else {
		for _, r := range routesArray {
			routes = append(routes, private.DinosaurAllOfRoutes{Domain: r.Domain, Router: r.Router})
		}
		return routes
	}
}
