package auth

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type RequireIssuerMiddleware interface {
	// RequireIssuer will check that the given issuer is set as part of the JWT
	// claims in the request and return code ServiceErrorCode in case it doesn't
	RequireIssuer(issuer string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type requireIssuerMiddleware struct {
}

var _ RequireIssuerMiddleware = &requireIssuerMiddleware{}

func NewRequireIssuerMiddleware() RequireIssuerMiddleware {
	return &requireIssuerMiddleware{}
}

func (m *requireIssuerMiddleware) RequireIssuer(issuer string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}
			issuerMatches := claims.VerifyIssuer(issuer, true)
			if !issuerMatches {
				shared.HandleError(request, writer, serviceErr)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
