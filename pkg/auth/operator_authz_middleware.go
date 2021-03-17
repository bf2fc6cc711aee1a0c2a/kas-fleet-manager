package auth

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

type Actor string

const (
	Kas       Actor = "kas"
	Connector Actor = "connector"
)

func UseOperatorAuthorisationMiddleware(router *mux.Router, actor Actor, jwkValidIssuerURI string, clusterIdVar string) {
	var requiredRole string

	if actor == Kas {
		requiredRole = "kas_fleetshard_operator"
	} else {
		requiredRole = "connector_fleetshard_operator"
	}

	router.Use(
		NewRolesAuhzMiddleware().RequireRealmRole(requiredRole, errors.ErrorNotFound),
		checkClusterId(actor, clusterIdVar),
		NewOCMAuthorizationMiddleware().RequireIssuer(jwkValidIssuerURI, errors.ErrorNotFound),
	)
}

func checkClusterId(actor Actor, clusterIdVar string) mux.MiddlewareFunc {
	var clusterIdClaimKey string

	if actor == Kas {
		clusterIdClaimKey = "kas-fleetshard-operator-cluster-id"
	} else {
		clusterIdClaimKey = "connector-fleetshard-operator-cluster-id"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			clusterId := mux.Vars(request)[clusterIdVar]
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				// deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are not authorised
				shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
				return
			}
			if clusterIdInClaim, ok := claims[clusterIdClaimKey].(string); ok {
				if clusterIdInClaim == clusterId {
					next.ServeHTTP(writer, request)
					return
				}
			}
			shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
		})
	}
}
