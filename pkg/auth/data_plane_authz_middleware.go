package auth

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

// DataPlaneAuthorisationMiddleware is mainly used to perform authorisation checks for endpoints that are called by the kas-fleetshard-operator
type DataPlaneAuthorisationMiddleware interface {
	// CheckClusterId will check if the cluster id value in the request URL is matching the value in the request token
	CheckClusterId(next http.Handler) http.Handler
}

type dataPlaneAuthzMiddleware struct {
}

var _ DataPlaneAuthorisationMiddleware = &dataPlaneAuthzMiddleware{}

func NewDataPlaneAuthzMiddleware() DataPlaneAuthorisationMiddleware {
	return &dataPlaneAuthzMiddleware{}
}

func (m dataPlaneAuthzMiddleware) CheckClusterId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		clusterId := mux.Vars(request)["id"]
		claims, err := GetClaimsFromContext(ctx)
		if err != nil {
			// deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are not authorised
			shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
			return
		}
		if clusterIdInClaim, ok := claims["kas-fleetshard-operator-cluster-id"].(string); ok {
			if clusterIdInClaim == clusterId {
				next.ServeHTTP(writer, request)
				return
			}
		}
		shared.HandleError(ctx, writer, errors.ErrorNotFound, "")
	})
}
