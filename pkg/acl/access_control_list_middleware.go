package acl

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type AccessControlListMiddleware struct {
	configService services.ConfigService
}

func NewAccessControlListMiddleware(configService services.ConfigService) *AccessControlListMiddleware {
	middleware := AccessControlListMiddleware{
		configService: configService,
	}

	return &middleware
}

// Middleware handler to authorize users based on the provided ACL configuration
func (middleware *AccessControlListMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := r.Context()
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			shared.HandleError(r.Context(), w, errors.ErrorForbidden, err.Error())
			return
		}

		username := auth.GetUsernameFromClaims(claims)

		accessControlListConfig := middleware.configService.GetConfig().AccessControlList
		if accessControlListConfig.EnableDenyList {
			userIsDenied := accessControlListConfig.DenyList.IsUserDenied(username)
			if userIsDenied {
				shared.HandleError(r.Context(), w, errors.ErrorForbidden, fmt.Sprintf("User '%s' is not authorized to access the service.", username))
				return
			}
		}

		orgId := auth.GetOrgIdFromClaims(claims)
		org, _ := middleware.configService.GetOrganisationById(orgId)

		if !accessControlListConfig.AllowList.AllowAnyRegisteredUsers {
			var userIsAllowed bool

			if org.IsUserAllowed(username) {
				userIsAllowed = true
			} else {
				// check if user is allowed as a service account if they do not belong to an org
				_, userIsAllowed = middleware.configService.GetServiceAccountByUsername(username)
			}

			if !userIsAllowed {
				shared.HandleError(r.Context(), w, errors.ErrorForbidden, fmt.Sprintf("User '%s' is not authorized to access the service.", username))
				return
			}
		}

		// user is allowed as a service account if they do not belong to an org listed in the allow list
		userIsAllowedAsServiceAccount := !org.IsUserAllowed(username)
		context = auth.SetUserIsAllowedAsServiceAccountContext(context, userIsAllowedAsServiceAccount)
		*r = *r.WithContext(context)

		next.ServeHTTP(w, r)
	})
}
