package auth

/*
   The goal of this simple authz middlewre is to provide a way for access review
   parameters to be declared for each route in a microservice. This is not meant
   to handle more complex access review calls in particular scopes, but rather
   just authz calls at the application scope

  This is a big TODO, not ready for consumption
*/

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type AuthorizationMiddleware interface {
	AuthorizeApi(next http.Handler) http.Handler
}

type authzMiddleware struct {
	ocmClient *ocm.Client
}

var _ AuthorizationMiddleware = &authzMiddleware{}

func NewAuthzMiddleware(ocmClient *ocm.Client) AuthorizationMiddleware {
	return &authzMiddleware{
		ocmClient:    ocmClient,
	}
}

func (a authzMiddleware) AuthorizeApi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, err := GetClaimsFromContext(ctx)
		if err != nil {
			shared.HandleError(r.Context(), w, errors.ErrorForbidden, err.Error())
			return
		}
		username := GetUsernameFromClaims(claims)
		allowed, err := a.ocmClient.Authorization.IsUserHasValidSubs(ctx)

		if err != nil {
			shared.HandleError(r.Context(), w, errors.ErrorForbidden, err.Error())
			return
		}
		if allowed {
			next.ServeHTTP(w, r)
		}

		if !allowed {
			shared.HandleError(r.Context(), w, errors.ErrorForbidden, fmt.Sprintf("User '%s' is not authorized to access the service.", username))
			return
		}
	})
}
