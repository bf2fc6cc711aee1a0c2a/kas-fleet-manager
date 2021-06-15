package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/goava/di"
	"net/http"
	"net/url"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorClusterIdLength = 32
)

type ConnectorClusterHandler struct {
	di.Inject
	Bus            signalbus.SignalBus
	Service        services.ConnectorClusterService
	Keycloak       coreservices.KafkaKeycloakService
	ConnectorTypes services.ConnectorTypesService
	Vault          vault.VaultService
	KeycloakConfig *config.KeycloakConfig
	ServerConfig   *config.ServerConfig
}

func NewConnectorClusterHandler(handler ConnectorClusterHandler) *ConnectorClusterHandler {
	return &handler
}

func (h *ConnectorClusterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource public.ConnectorCluster
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Metadata.Name,
				handlers.WithDefault("New Cluster"), handlers.MinLen(1), handlers.MaxLen(100)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters.ConvertConnectorCluster(resource)

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convResource.Owner = auth.GetUsernameFromClaims(claims)
			convResource.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convResource.Status.Phase = dbapi.ConnectorClusterPhaseUnconnected

			if err := h.Service.Create(r.Context(), &convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(convResource), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h *ConnectorClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.Service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(resource), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			err := h.Service.Delete(r.Context(), connectorClusterId)
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := coreservices.NewListArguments(r.URL.Query())
			resources, paging, err := h.Service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := public.ConnectorClusterList{
				Kind:  "ConnectorClusterList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters.PresentConnectorCluster(resource)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h *ConnectorClusterHandler) GetAddonParameters(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			// To make sure the user can access the cluster....
			_, err := h.Service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}

			acc, err := h.Keycloak.RegisterConnectorFleetshardOperatorServiceAccount(connectorClusterId, connectorFleetshardOperatorRoleName)
			if err != nil {
				return false, errors.GeneralError("failed to create service account for connector cluster %s due to error: %v", connectorClusterId, err)
			}
			params := h.buildAddonParams(acc, connectorClusterId)
			result := make([]public.AddonParameter, len(params))
			for i, p := range params {
				result[i] = presenters.PresentAddonParameter(p)
			}
			u, eerr := h.buildTokenURL(acc)
			if eerr == nil {
				w.Header().Set("Token-Help", fmt.Sprintf(`curl --data "grant_type=client_credentials" "%s" | jq -r .access_token`, u))
			}
			return result, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

const (
	connectorFleetshardOperatorRoleName                  = "connector_fleetshard_operator"
	connectorFleetshardOperatorParamMasSSOBaseUrl        = "mas-sso-base-url"
	connectorFleetshardOperatorParamMasSSORealm          = "mas-sso-realm"
	connectorFleetshardOperatorParamServiceAccountId     = "client-id"
	connectorFleetshardOperatorParamServiceAccountSecret = "client-secret"
	connectorFleetshardOperatorParamClusterId            = "cluster-id"
	connectorFleetshardOperatorParamControlPlaneBaseURL  = "control-plane-base-url"
)

func (o *ConnectorClusterHandler) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{
		{
			Id:    connectorFleetshardOperatorParamMasSSOBaseUrl,
			Value: o.KeycloakConfig.BaseURL,
		},
		{
			Id:    connectorFleetshardOperatorParamMasSSORealm,
			Value: o.KeycloakConfig.KafkaRealm.Realm,
		},
		{
			Id:    connectorFleetshardOperatorParamServiceAccountId,
			Value: serviceAccount.ClientID,
		},
		{
			Id:    connectorFleetshardOperatorParamServiceAccountSecret,
			Value: serviceAccount.ClientSecret,
		},
		{
			Id:    connectorFleetshardOperatorParamControlPlaneBaseURL,
			Value: o.ServerConfig.PublicHostURL,
		},
		{
			Id:    connectorFleetshardOperatorParamClusterId,
			Value: clusterId,
		},
	}
	return p
}

func (o *ConnectorClusterHandler) buildTokenURL(serviceAccount *api.ServiceAccount) (string, error) {
	u, err := url.Parse(o.KeycloakConfig.KafkaRealm.TokenEndpointURI)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(serviceAccount.ClientID, serviceAccount.ClientSecret)
	return u.String(), nil
}
