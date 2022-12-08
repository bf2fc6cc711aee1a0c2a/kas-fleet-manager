package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

type clusterHandler struct {
	kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	clusterService             services.ClusterService
}

func NewClusterHandler(kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon, clusterService services.ClusterService) *clusterHandler {
	return &clusterHandler{
		kasFleetshardOperatorAddon: kasFleetshardOperatorAddon,
		clusterService:             clusterService,
	}
}

func (h clusterHandler) RegisterEnterpriseCluster(w http.ResponseWriter, r *http.Request) {
	var clusterPayload public.EnterpriseOsdClusterPayload

	ctx := r.Context()

	cfg := &handlers.HandlerConfig{
		MarshalInto: &clusterPayload,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterPayload.ClusterId, "cluster id", ClusterIdLength, &ClusterIdLength),

			handlers.ValidateExternalClusterId(&clusterPayload.ClusterExternalId, "external cluster id"),

			handlers.ValidateClusterId(&clusterPayload.ClusterId, "cluster id"),

			ValidateClusterIdIsUnique(&clusterPayload.ClusterId, h.clusterService),

			handlers.ValidateDnsName(&clusterPayload.ClusterIngressDnsName, "cluster dns name"),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			claims, claimsErr := getClaims(ctx)

			if claimsErr != nil {
				return nil, claimsErr
			}

			orgId, getOrgIdErr := claims.GetOrgId()

			if getOrgIdErr != nil {
				return nil, errors.GeneralError(getOrgIdErr.Error())
			}

			clusterRequest := &api.Cluster{
				ClusterType:           api.Enterprise.String(),
				Status:                api.ClusterAccepted,
				ClusterID:             clusterPayload.ClusterId,
				OrganizationID:        orgId,
				ClusterDNS:            clusterPayload.ClusterIngressDnsName,
				ExternalID:            clusterPayload.ClusterExternalId,
				MultiAZ:               true,
				SupportedInstanceType: api.StandardTypeSupport.String(),
			}

			fsoParams, svcErr := h.kasFleetshardOperatorAddon.GetAddonParams(clusterRequest)

			if svcErr != nil {
				return nil, svcErr
			}

			clusterRequest.ClientID = fsoParams.GetParam(services.KasFleetshardOperatorParamServiceAccountId)

			clusterRequest.ClientSecret = fsoParams.GetParam(services.KasFleetshardOperatorParamServiceAccountSecret)

			svcErr = h.clusterService.RegisterClusterJob(clusterRequest)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentEnterpriseCluster(*clusterRequest, fsoParams)
		},
	}

	// return 200 status ok
	handlers.Handle(w, r, cfg, http.StatusOK)
}
