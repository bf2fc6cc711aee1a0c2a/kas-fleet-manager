package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
)

func PresentServiceStatus(userInDenyList bool, dinosaurMaximumCapacityReached bool) *public.ServiceStatus {
	return &public.ServiceStatus{
		Dinosaurs: public.ServiceStatusDinosaurs{
			MaxCapacityReached: userInDenyList || dinosaurMaximumCapacityReached,
		},
	}
}
