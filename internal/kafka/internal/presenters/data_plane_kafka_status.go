package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

func ConvertDataPlaneKafkaStatus(status map[string]private.DataPlaneKafkaStatus) []*api.DataPlaneKafkaStatus {
	var r []*api.DataPlaneKafkaStatus
	for k, v := range status {
		var c []api.DataPlaneKafkaStatusCondition
		for _, s := range v.Conditions {
			c = append(c, api.DataPlaneKafkaStatusCondition{
				Type:    s.Type,
				Reason:  s.Reason,
				Status:  s.Status,
				Message: s.Message,
			})
		}
		r = append(r, &api.DataPlaneKafkaStatus{
			KafkaClusterId: k,
			Conditions:     c,
		})
	}

	return r
}
