package auth

import (
	"fmt"
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
	router.Use(
		checkClusterId(actor, clusterIdVar),
		NewRequireIssuerMiddleware().RequireIssuer([]string{jwkValidIssuerURI}, errors.ErrorNotFound),
	)
}

func checkClusterId(actor Actor, clusterIdVar string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			clusterId := mux.Vars(request)[clusterIdVar]
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				// deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are not authorised
				shared.HandleError(request, writer, errors.NotFound(""))
				return
			}
			if clientId, ok := claims["clientId"].(string); ok {
				if actor == Kas {
					if clientId == fmt.Sprintf("kas-fleetshard-agent-%s", clusterId) {
						next.ServeHTTP(writer, request)
						return
					}
				} else {
					if clientId == fmt.Sprintf("connector-fleetshard-agent-%s", clusterId) {
						next.ServeHTTP(writer, request)
						return
					}
				}
			}

			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}
