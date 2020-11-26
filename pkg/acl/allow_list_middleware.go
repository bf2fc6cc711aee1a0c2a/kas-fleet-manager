package acl

import (
	"fmt"
	"net/http"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/shared"
)

type AllowListMiddleware struct {
	configService services.ConfigService
}

func NewAllowListMiddleware(configService services.ConfigService) *AllowListMiddleware {
	middleware := AllowListMiddleware{
		configService: configService,
	}

	return &middleware
}

// Middleware handler to authorize users based on the provided ACL configuration
func (middleware *AllowListMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.configService.IsAllowListEnabled() {
			payload, _ := auth.GetAuthPayload(r)
			org, _ := middleware.configService.GetOrganisationById(payload.OrganisationId)
			userIsAllowed := middleware.configService.IsUserAllowed(payload.Username, org)
			if !userIsAllowed {
				shared.HandleError(r.Context(), w, errors.ErrorForbidden, fmt.Sprintf("User %s is not authorized to access the service.", payload.Username))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
