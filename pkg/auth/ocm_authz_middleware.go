package auth

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type OCMAuthorizationMiddleware interface {
	// RequireIssuer will check that the given issuer is set as part of the JWT
	// claims in the request and return code ServiceErrorCode in case it doesn't
	RequireIssuer(issuer string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type ocmAuthorizationMiddleware struct{}

var _ OCMAuthorizationMiddleware = &ocmAuthorizationMiddleware{}

func NewOCMAuthorizationMiddleware() OCMAuthorizationMiddleware {
	return &ocmAuthorizationMiddleware{}
}

func (m *ocmAuthorizationMiddleware) RequireIssuer(issuer string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				shared.HandleError(ctx, writer, code, "")
				return
			}
			issuerMatches := claims.VerifyIssuer(issuer, true)
			if !issuerMatches {
				shared.HandleError(ctx, writer, code, "")
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
