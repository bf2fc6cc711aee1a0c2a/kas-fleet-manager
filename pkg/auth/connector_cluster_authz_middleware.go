package auth

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

func UseConnectorClusterAuthorisationMiddleware(router *mux.Router) {
	rolesAuthzMiddleware := NewRolesAuhzMiddleware()
	check1 := rolesAuthzMiddleware.RequireRealmRole("connector_fleetshard_operator", errors.ErrorNotFound)
	router.Use(check1, checkClusterId)
}

func checkClusterId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		clusterId := mux.Vars(request)["id"]
		claims, err := GetClaimsFromContext(ctx)
		if err != nil {
			// deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are not authorised
			shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
			return
		}
		if clusterIdInClaim, ok := claims["connector-fleetshard-operator-cluster-id"].(string); ok {
			if clusterIdInClaim == clusterId {
				next.ServeHTTP(writer, request)
				return
			}
		}
		shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
	})
}
