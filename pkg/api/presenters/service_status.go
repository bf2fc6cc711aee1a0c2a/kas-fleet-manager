package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
)

func PresentServiceStatus(userInDenyList bool, kafkaMaximumCapacityReached bool) *openapi.ServiceStatus {
	return &openapi.ServiceStatus{
		Kafkas: openapi.ServiceStatusKafkas{
			MaxCapacityReached: userInDenyList || kafkaMaximumCapacityReached,
		},
	}
}
