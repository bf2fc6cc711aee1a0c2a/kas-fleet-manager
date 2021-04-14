package auth

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type OCMAuthorizationMiddleware interface {
	// RequireIssuer will check that the given issuer is set as part of the JWT
	// claims in the request and return code ServiceErrorCode in case it doesn't
	RequireIssuer(issuer string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
	// RequireTermsAcceptance will check that the user has accepted the required terms
	RequireTermsAcceptance(enabled bool, ocmClient ocm.Client, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
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

func (m *ocmAuthorizationMiddleware) RequireTermsAcceptance(enabled bool, ocmClient ocm.Client, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if enabled {
				ctx := request.Context()
				claims, err := GetClaimsFromContext(ctx)
				if err != nil {
					shared.HandleError(ctx, writer, code, err.Error())
					return
				}
				username := GetUsernameFromClaims(claims)
				if termsRequired, err := ocmClient.GetRequiresTermsAcceptance(username); err != nil {
					shared.HandleError(ctx, writer, code, err.Error())
				} else {
					if termsRequired {
						shared.HandleError(ctx, writer, code, "required terms have not been accepted")
					}
				}
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
