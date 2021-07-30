package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
)

func ConvertDataPlaneKafkaStatus(status map[string]private.DataPlaneKafkaStatus) []*dbapi.DataPlaneKafkaStatus {
	var r []*dbapi.DataPlaneKafkaStatus
	for k, v := range status {
		var c []dbapi.DataPlaneKafkaStatusCondition
		var routes []dbapi.DataPlaneKafkaRouteRequest
		for _, s := range v.Conditions {
			c = append(c, dbapi.DataPlaneKafkaStatusCondition{
				Type:    s.Type,
				Reason:  s.Reason,
				Status:  s.Status,
				Message: s.Message,
			})
		}
		if v.Routes != nil {
			for _, ro := range *v.Routes {
				routes = append(routes, dbapi.DataPlaneKafkaRouteRequest{
					Name:   ro.Name,
					Prefix: ro.Prefix,
					Router: ro.Router,
				})
			}
		}
		r = append(r, &dbapi.DataPlaneKafkaStatus{
			KafkaClusterId: k,
			Conditions:     c,
			Routes:         routes,
			KafkaVersion:   v.Versions.Kafka,
			StrimziVersion: v.Versions.Strimzi,
		})
	}

	return r
}
