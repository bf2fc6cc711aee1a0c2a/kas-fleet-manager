package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/gorilla/mux"
)

type observatoriumProxyHandler struct {
	clusterService services.ClusterService
}

func NewObservatoriumProxyHandler(clusterService services.ClusterService) *observatoriumProxyHandler {
	return &observatoriumProxyHandler{
		clusterService: clusterService,
	}
}

// ValidateTokenAndExternalClusterID validates combination of external cluster ID parameter against client ID from the claims
func (h observatoriumProxyHandler) ValidateTokenAndExternalClusterID(w http.ResponseWriter, r *http.Request) {
	external_id := mux.Vars(r)["cluster_external_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateExternalClusterId(&external_id, "external cluster id"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()

			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				return nil, errors.Unauthenticated("unable to authenticate token provided in the request")
			}

			clientID, err := claims.GetClientID()
			if err != nil {
				return nil, errors.Unauthenticated("unable to get client ID from the token")
			}

			cluster, err := h.clusterService.FindCluster(services.FindClusterCriteria{ExternalID: external_id, MultiAZ: true})
			if err != nil {
				return nil, errors.GeneralError("failed to validate the request: %v" + err.Error())
			}

			if cluster == nil {
				return nil, errors.NotFound("unable to find cluster with external cluster ID: %s", external_id)
			}
			if clientID != cluster.ClientID {
				return nil, errors.Forbidden("failed to match the client ID in the token against provided external cluster ID: %s" + external_id)
			}
			return nil, nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
