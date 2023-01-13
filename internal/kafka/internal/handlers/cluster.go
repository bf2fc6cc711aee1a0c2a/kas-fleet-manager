package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/gorilla/mux"
	"net/http"
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

			validateKafkaMachinePoolNodeCount(&clusterPayload),
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

			if !claims.IsOrgAdmin() {
				return nil, errors.New(errors.ErrorUnauthorized, "non admin user not authorized to perform this action")
			}

			supportedKafkaInstanceType := api.StandardTypeSupport.String()
			clusterRequest := &api.Cluster{
				ClusterType:                   api.EnterpriseDataPlaneClusterType.String(),
				ProviderType:                  api.ClusterProviderOCM,
				Status:                        api.ClusterAccepted,
				ClusterID:                     clusterPayload.ClusterId,
				OrganizationID:                orgId,
				ClusterDNS:                    clusterPayload.ClusterIngressDnsName,
				ExternalID:                    clusterPayload.ClusterExternalId,
				MultiAZ:                       true,
				AccessKafkasViaPrivateNetwork: clusterPayload.AccessKafkasViaPrivateNetwork,
				SupportedInstanceType:         supportedKafkaInstanceType,
			}

			capacityInfo := map[string]api.DynamicCapacityInfo{
				supportedKafkaInstanceType: {
					MaxNodes: clusterPayload.KafkaMachinePoolNodeCount,
				},
			}

			err := clusterRequest.SetDynamicCapacityInfo(capacityInfo)
			if err != nil { // this should never occur
				return nil, errors.GeneralError("invalid node count info")
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
			return presenters.PresentEnterpriseClusterWithAddonParams(*clusterRequest, fsoParams)
		},
	}

	// return 200 status ok
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h clusterHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			clusters, err := h.clusterService.ListEnterpriseClustersOfAnOrganization(ctx)
			if err != nil {
				return nil, err
			}

			clusterList := public.EnterpriseClusterList{
				Kind:  "ClusterList",
				Page:  1,
				Size:  int32(len(clusters)),
				Total: int32(len(clusters)),
				Items: []public.EnterpriseCluster{},
			}

			for _, cluster := range clusters {
				converted := presenters.PresentEnterpriseCluster(*cluster)
				clusterList.Items = append(clusterList.Items, converted)
			}

			return clusterList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h clusterHandler) DeregisterEnterpriseCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clusterID := mux.Vars(r)["id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting enterprise cluster"),
			ValidateKafkaClaims(ctx, ValidateOrganisationId()),
			validateEnterpriseClusterEligibleForDeregistration(ctx, clusterID, h.clusterService),
			validateEnterpriseClusterHasNoKafkas(ctx, clusterID, h.clusterService),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			return nil, h.clusterService.DeregisterClusterJob(clusterID)
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h clusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateClusterId(&clusterID, "cluster id"),
			ValidateKafkaClaims(ctx, ValidateOrganisationId()),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			// error checked in the validate, no need to check again
			claims, _ := getClaims(ctx)
			orgID, _ := claims.GetOrgId()

			cluster, err := h.clusterService.FindClusterByID(clusterID)
			if err != nil {
				return nil, err
			}

			if cluster == nil || cluster.OrganizationID != orgID || cluster.ClusterType != api.EnterpriseDataPlaneClusterType.String() {
				return nil, errors.NotFound("enterprise data plane cluster with id='%v' not found within organization: %s", clusterID, orgID)
			}

			return presenters.PresentEnterpriseCluster(*cluster), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h clusterHandler) GetEnterpriseClusterWithAddonParams(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateClusterId(&clusterID, "cluster id"),
			ValidateKafkaClaims(ctx, ValidateOrganisationId()),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			// error checked in the validate, no need to check again
			claims, _ := getClaims(ctx)
			orgID, _ := claims.GetOrgId()

			if !claims.IsOrgAdmin() {
				return nil, errors.New(errors.ErrorUnauthorized, "non admin user not authorized to perform this action")
			}

			cluster, err := h.clusterService.FindClusterByID(clusterID)
			if err != nil {
				return nil, err
			}

			if cluster == nil || cluster.OrganizationID != orgID || cluster.ClusterType != api.EnterpriseDataPlaneClusterType.String() {
				return nil, errors.NotFound("enterprise data plane cluster with id='%v' not found within organization: %s", clusterID, orgID)
			}

			fsoParams, svcErr := h.kasFleetshardOperatorAddon.GetAddonParams(cluster)

			if svcErr != nil {
				return nil, svcErr
			}

			return presenters.PresentEnterpriseClusterWithAddonParams(*cluster, fsoParams)
		},
	}
	handlers.HandleGet(w, r, cfg)
}
