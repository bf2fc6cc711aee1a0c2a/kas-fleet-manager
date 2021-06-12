package handlers

import (
	"fmt"
	presenters2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	services2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"net/http"
	"net/url"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorClusterIdLength = 32
)

type ConnectorClusterHandler struct {
	bus            signalbus.SignalBus
	service        services2.ConnectorClusterService
	config         services.ConfigService
	keycloak       services.KeycloakService
	connectorTypes services2.ConnectorTypesService
	vault          services.VaultService
}

func NewConnectorClusterHandler(bus signalbus.SignalBus, service services2.ConnectorClusterService, config services.ConfigService, keycloak services.KafkaKeycloakService, connectorTypes services2.ConnectorTypesService, vault services.VaultService) *ConnectorClusterHandler {
	return &ConnectorClusterHandler{
		bus:            bus,
		service:        service,
		config:         config,
		keycloak:       keycloak,
		connectorTypes: connectorTypes,
		vault:          vault,
	}
}

func (h *ConnectorClusterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource openapi.ConnectorCluster
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Metadata.Name,
				handlers.WithDefault("New Cluster"), handlers.MinLen(1), handlers.MaxLen(100)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters2.ConvertConnectorCluster(resource)

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convResource.Owner = auth.GetUsernameFromClaims(claims)
			convResource.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convResource.Status.Phase = api.ConnectorClusterPhaseUnconnected

			if err := h.service.Create(r.Context(), &convResource); err != nil {
				return nil, err
			}
			return presenters2.PresentConnectorCluster(convResource), nil
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
			resource, err := h.service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}
			return presenters2.PresentConnectorCluster(resource), nil
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
			err := h.service.Delete(r.Context(), connectorClusterId)
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())
			resources, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := openapi.ConnectorClusterList{
				Kind:  "ConnectorClusterList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters2.PresentConnectorCluster(resource)
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
			_, err := h.service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}

			acc, err := h.keycloak.RegisterConnectorFleetshardOperatorServiceAccount(connectorClusterId, connectorFleetshardOperatorRoleName)
			if err != nil {
				return false, errors.GeneralError("failed to create service account for connector cluster %s due to error: %v", connectorClusterId, err)
			}
			params := h.buildAddonParams(acc, connectorClusterId)
			result := make([]openapi.AddonParameter, len(params))
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
			Value: o.config.GetConfig().Keycloak.BaseURL,
		},
		{
			Id:    connectorFleetshardOperatorParamMasSSORealm,
			Value: o.config.GetConfig().Keycloak.KafkaRealm.Realm,
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
			Value: o.config.GetConfig().Server.PublicHostURL,
		},
		{
			Id:    connectorFleetshardOperatorParamClusterId,
			Value: clusterId,
		},
	}
	return p
}

func (o *ConnectorClusterHandler) buildTokenURL(serviceAccount *api.ServiceAccount) (string, error) {
	u, err := url.Parse(o.config.GetConfig().Keycloak.KafkaRealm.TokenEndpointURI)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(serviceAccount.ClientID, serviceAccount.ClientSecret)
	return u.String(), nil
}
