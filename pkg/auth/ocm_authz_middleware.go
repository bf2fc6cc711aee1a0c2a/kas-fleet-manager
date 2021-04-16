package auth

import (
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/patrickmn/go-cache"

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

type ocmAuthorizationMiddleware struct {
	cache *cache.Cache
}

var _ OCMAuthorizationMiddleware = &ocmAuthorizationMiddleware{}

func NewOCMAuthorizationMiddleware() OCMAuthorizationMiddleware {
	return &ocmAuthorizationMiddleware{
		// entries will expire in 5 minutes and will be evicted every 10 minutes
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
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
				termsRequired, cached := m.cache.Get(username)
				if !cached {
					termsRequired, _, err = ocmClient.GetRequiresTermsAcceptance(username)
					if err != nil {
						shared.HandleError(ctx, writer, code, err.Error())
						return
					}

					m.cache.Set(username, termsRequired, cache.DefaultExpiration)
				}

				if termsRequired.(bool) {
					shared.HandleError(ctx, writer, code, "required terms have not been accepted")
					return
				}
			}

			next.ServeHTTP(writer, request)
		})
	}
}
