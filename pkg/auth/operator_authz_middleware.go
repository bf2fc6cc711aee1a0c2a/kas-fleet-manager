package auth

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"
)

type Actor string

const (
	Kas       Actor = "kas"
	Connector Actor = "connector"
)

func UseOperatorAuthorisationMiddleware(router *mux.Router, actor Actor, jwkValidIssuerURI string, clusterIdVar string, clusterService AuthAgentService) {
	router.Use(
		checkClusterId(actor, clusterIdVar, clusterService),
		NewRequireIssuerMiddleware().RequireIssuer([]string{jwkValidIssuerURI}, errors.ErrorNotFound),
	)
}

func checkClusterId(actor Actor, clusterIdVar string, authAgentService AuthAgentService) mux.MiddlewareFunc {
	if actor == Kas {
		return checkKafkaClusterId(clusterIdVar, authAgentService)
	} else {
		return checkConnectorClusterId(clusterIdVar)
	}
}

func checkConnectorClusterId(clusterIdVar string) mux.MiddlewareFunc {
	var clusterIdClaimKey = "connector-fleetshard-operator-cluster-id"

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
			if clusterIdInClaim, ok := claims[clusterIdClaimKey].(string); ok {
				if clusterIdInClaim == clusterId {
					next.ServeHTTP(writer, request)
					return
				}
			}
			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}

func checkKafkaClusterId(clusterIdVar string, authAgentService AuthAgentService) mux.MiddlewareFunc {
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

			savedClientId, err := authAgentService.GetClientId(clusterId)
			if err != nil {
				glog.Errorf("unable to get clientID for cluster with ID '%s': %v", clusterId, err)
				shared.HandleError(request, writer, errors.GeneralError("unable to get clientID for cluster with ID '%s'", clusterId))
			}

			if clientId, ok := claims["clientId"].(string); ok {
				if clientId == savedClientId {
					next.ServeHTTP(writer, request)
					return
				}
			}

			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}
