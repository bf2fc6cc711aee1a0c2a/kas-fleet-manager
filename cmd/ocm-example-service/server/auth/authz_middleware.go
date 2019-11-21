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

	"github.com/openshift-online/ocm-sdk-go"

	"gitlab.cee.redhat.com/service/uhc-example-service/pkg/api"
)

type AuthorizationMiddleware interface {
	AuthorizeApi(next http.Handler) http.Handler
}

type authzMiddleware struct {
	action       string
	resourceType string

	ocmClient *sdk.Connection
}

var _ AuthorizationMiddleware = &authzMiddleware{}

func NewAuthzMiddleware(ocmClient *sdk.Connection, action, resourceType string) AuthorizationMiddleware {
	return &authzMiddleware{
		ocmClient:    ocmClient,
		action:       action,
		resourcetype: resourceType,
	}
}

func (a authzMiddleware) AuthorizeApi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get username from context
		username, err := PreferredUsernameFromContext(ctx)
		if err != nil {
			fmt.Errorf("Authenticated username not present in request context")
			body := api.E500.Format(r, "Authentication details not present in context")
			api.SendError(w, r, &body)
			return
		}

		allowed, err := a.ocmClient.Authorization.AccessReview(
			ctx, username, a.action, a.resourceType, "", "", "")
		if err != nil {
			fmt.Errorf("Unable to make authorization request: %s", err)
			body := api.E500.Format(r, "Unable to make authorization request")
			api.SendError(w, r, &body)
			return
		}

		if allowed {
			next.ServeHTTP(w, r)
		}

		body := api.E403.Format(r, "")
		api.SendError(w, r, &body)
	})
}
