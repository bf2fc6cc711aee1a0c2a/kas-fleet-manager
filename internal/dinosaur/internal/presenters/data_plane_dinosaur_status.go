package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
)

func ConvertDataPlaneDinosaurStatus(status map[string]private.DataPlaneDinosaurStatus) []*dbapi.DataPlaneDinosaurStatus {
	var r []*dbapi.DataPlaneDinosaurStatus
	for k, v := range status {
		var c []dbapi.DataPlaneDinosaurStatusCondition
		var routes []dbapi.DataPlaneDinosaurRouteRequest
		for _, s := range v.Conditions {
			c = append(c, dbapi.DataPlaneDinosaurStatusCondition{
				Type:    s.Type,
				Reason:  s.Reason,
				Status:  s.Status,
				Message: s.Message,
			})
		}
		if v.Routes != nil {
			for _, ro := range *v.Routes {
				routes = append(routes, dbapi.DataPlaneDinosaurRouteRequest{
					Name:   ro.Name,
					Prefix: ro.Prefix,
					Router: ro.Router,
				})
			}
		}
		r = append(r, &dbapi.DataPlaneDinosaurStatus{
			DinosaurClusterId:  k,
			Conditions:      c,
			Routes:          routes,
			DinosaurVersion:    v.Versions.Dinosaur,
			StrimziVersion:  v.Versions.Strimzi,
			DinosaurIBPVersion: v.Versions.DinosaurIbp,
		})
	}

	return r
}
