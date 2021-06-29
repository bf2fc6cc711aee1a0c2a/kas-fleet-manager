package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
)

func PresentServiceStatus(userInDenyList bool, kafkaMaximumCapacityReached bool) *public.ServiceStatus {
	return &public.ServiceStatus{
		Kafkas: public.ServiceStatusKafkas{
			MaxCapacityReached: userInDenyList || kafkaMaximumCapacityReached,
		},
	}
}
