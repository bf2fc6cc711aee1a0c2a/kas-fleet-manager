package acl

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type AccessControlListMiddleware struct {
	accessControlListConfig *config.AccessControlListConfig
}

func NewAccessControlListMiddleware(accessControlListConfig *config.AccessControlListConfig) *AccessControlListMiddleware {
	middleware := AccessControlListMiddleware{
		accessControlListConfig: accessControlListConfig,
	}
	return &middleware
}

// Middleware handler to authorize users based on the provided ACL configuration
func (middleware *AccessControlListMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := r.Context()
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			shared.HandleError(r, w, errors.NewWithCause(errors.ErrorForbidden, err, ""))
			return
		}

		username := auth.GetUsernameFromClaims(claims)

		if middleware.accessControlListConfig.EnableDenyList {
			userIsDenied := middleware.accessControlListConfig.DenyList.IsUserDenied(username)
			if userIsDenied {
				shared.HandleError(r, w, errors.New(errors.ErrorForbidden, "User '%s' is not authorized to access the service.", username))
				return
			}
		}

		orgId := auth.GetOrgIdFromClaims(claims)

		if !middleware.accessControlListConfig.AllowList.AllowAnyRegisteredUsers {
			// check if user is in the allow list
			org, _ := middleware.accessControlListConfig.AllowList.Organisations.GetById(orgId)
			userIsAnAllowListOrgMember := org.IsUserAllowed(username)
			_, userIsAnAllowListServiceAccount := middleware.accessControlListConfig.AllowList.ServiceAccounts.GetByUsername(username)

			// If the user is not in the allow list as an org member or service account, they are not authorised
			if !userIsAnAllowListServiceAccount && !userIsAnAllowListOrgMember {
				shared.HandleError(r, w, errors.New(errors.ErrorForbidden, "User '%s' is not authorized to access the service.", username))
				return
			}
		}

		// If the users claim has an orgId, resources should be filtered by their organisation. Otherwise, filter them by owner.
		context = auth.SetFilterByOrganisationContext(context, orgId != "")
		*r = *r.WithContext(context)

		next.ServeHTTP(w, r)
	})
}
