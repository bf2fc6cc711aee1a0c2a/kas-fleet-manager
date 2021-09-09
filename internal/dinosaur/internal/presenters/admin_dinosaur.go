package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
)

func PresentDinosaurRequestAdminEndpoint(dinosaurRequest *dbapi.DinosaurRequest) private.Dinosaur {
	reference := PresentReference(dinosaurRequest.ID, dinosaurRequest)

	return private.Dinosaur{
		Id:                     reference.Id,
		Kind:                   reference.Kind,
		Href:                   reference.Href,
		Status:                 dinosaurRequest.Status,
		CloudProvider:          dinosaurRequest.CloudProvider,
		MultiAz:                dinosaurRequest.MultiAZ,
		Region:                 dinosaurRequest.Region,
		Owner:                  dinosaurRequest.Owner,
		Name:                   dinosaurRequest.Name,
		BootstrapServerHost:    setBootstrapServerHost(dinosaurRequest.BootstrapServerHost),
		CreatedAt:              dinosaurRequest.CreatedAt,
		UpdatedAt:              dinosaurRequest.UpdatedAt,
		FailedReason:           dinosaurRequest.FailedReason,
		DesiredDinosaurVersion: dinosaurRequest.DesiredDinosaurVersion,
		ActualDinosaurVersion:  dinosaurRequest.ActualDinosaurVersion,
		DesiredStrimziVersion:  dinosaurRequest.DesiredStrimziVersion,
		ActualStrimziVersion:   dinosaurRequest.ActualStrimziVersion,
		DinosaurUpgrading:      dinosaurRequest.DinosaurUpgrading,
		StrimziUpgrading:       dinosaurRequest.StrimziUpgrading,
		OrganisationId:         dinosaurRequest.OrganisationId,
		SubscriptionId:         dinosaurRequest.SubscriptionId,
		SsoClientId:            dinosaurRequest.SsoClientID,
		OwnerAccountId:         dinosaurRequest.OwnerAccountId,
		QuotaType:              dinosaurRequest.QuotaType,
		Routes:                 GetRoutesFromDinosaurRequest(dinosaurRequest),
		RoutesCreated:          dinosaurRequest.RoutesCreated,
		ClusterId:              dinosaurRequest.ClusterID,
		InstanceType:           dinosaurRequest.InstanceType,
		Namespace:              dinosaurRequest.Namespace,
	}
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
