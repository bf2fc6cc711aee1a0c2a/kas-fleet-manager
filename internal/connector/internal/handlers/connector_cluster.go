package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/golang/glog"
	"net/http"
	"net/url"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/goava/di"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorClusterIdLength = 32
)

type ConnectorClusterHandler struct {
	di.Inject
	Bus                signalbus.SignalBus
	Service            services.ConnectorClusterService
	Keycloak           sso.KafkaKeycloakService
	ConnectorTypes     services.ConnectorTypesService
	ConnectorNamespace services.ConnectorNamespaceService
	Vault              vault.VaultService
	KeycloakConfig     *keycloak.KeycloakConfig
	ServerConfig       *server.ServerConfig
	AuthZ              authz.AuthZService
	QuotaConfig        *config.ConnectorsQuotaConfig
}

func NewConnectorClusterHandler(handler ConnectorClusterHandler) *ConnectorClusterHandler {
	return &handler
}

func (h *ConnectorClusterHandler) Create(w http.ResponseWriter, r *http.Request) {

	user := h.AuthZ.GetValidationUser(r.Context())

	var resource public.ConnectorClusterRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Name, handlers.WithDefault("New Cluster"),
				handlers.MinLen(1), handlers.MaxLen(100)),
			user.AuthorizedOrgAdmin(),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters.ConvertConnectorClusterRequest(resource)

			convResource.ID = api.NewID()
			convResource.Owner = user.UserId()
			convResource.OrganisationId = user.OrgId()
			convResource.Status.Phase = dbapi.ConnectorClusterPhaseDisconnected

			acc, err := h.Keycloak.RegisterConnectorFleetshardOperatorServiceAccount(convResource.ID)
			if err != nil {
				return nil, errors.GeneralError("failed to create service account for connector cluster %s due to error: %v", convResource.ID, err)
			}
			convResource.ClientId = acc.ClientID
			convResource.ClientSecret = acc.ClientSecret

			if err = h.Service.Create(r.Context(), &convResource); err != nil {
				// deregister service account on creation error
				if derr := h.Keycloak.DeRegisterConnectorFleetshardOperatorServiceAccount(convResource.ID); derr != nil {
					// just log the error
					glog.Errorf("Error de-registering unused service account %s: %v", convResource.ClientId, derr)
				}
				return nil, err
			}
			return presenters.PresentConnectorCluster(convResource), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h *ConnectorClusterHandler) Get(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	user := h.AuthZ.GetValidationUser(ctx)

	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), user.AuthorizedClusterUser()),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.Service.Get(ctx, connectorClusterId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(resource), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) Update(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	user := h.AuthZ.GetValidationUser(ctx)

	var resource public.ConnectorClusterRequest
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), user.AuthorizedClusterAdmin()),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			existing, err := h.Service.Get(ctx, connectorClusterId)
			if err != nil {
				return nil, err
			}

			// Copy over the fields that support being updated...
			existing.Name = resource.Name

			return nil, h.Service.Update(ctx, &existing)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) Delete(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	user := h.AuthZ.GetValidationUser(ctx)

	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), user.AuthorizedClusterAdmin()),
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
			cluster, err := h.Service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}

			u, eerr := h.buildTokenURL(cluster)
			if eerr != nil {
				return false, errors.GeneralError("failed creating auth token url")
			}
			params := h.buildAddonParams(cluster, u)
			result := make([]public.AddonParameter, len(params))
			for i, p := range params {
				result[i] = presenters.PresentAddonParameter(p)
			}
			return result, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (o *ConnectorClusterHandler) buildAddonParams(cluster dbapi.ConnectorCluster, authTokenURL string) []ocm.Parameter {
	p := []ocm.Parameter{
		{
			Id:    "control-plane-base-url",
			Value: o.ServerConfig.PublicHostURL,
		},
		{
			Id:    "cluster-id",
			Value: cluster.ID,
		},
		{
			Id:    "auth-token-url",
			Value: authTokenURL,
		},
		{
			Id:    "mas-sso-base-url",
			Value: o.KeycloakConfig.BaseURL,
		},
		{
			Id:    "mas-sso-realm",
			Value: o.KeycloakConfig.KafkaRealm.Realm,
		},
		{
			Id:    "client-id",
			Value: cluster.ClientId,
		},
		{
			Id:    "client-secret",
			Value: cluster.ClientSecret,
		},
	}
	return p
}

func (o *ConnectorClusterHandler) buildTokenURL(cluster dbapi.ConnectorCluster) (string, error) {
	u, err := url.Parse(o.KeycloakConfig.KafkaRealm.TokenEndpointURI)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(cluster.ClientId, cluster.ClientSecret)
	return u.String(), nil
}

func (h *ConnectorClusterHandler) GetNamespaces(writer http.ResponseWriter, request *http.Request) {

	ctx := request.Context()
	user := h.AuthZ.GetValidationUser(ctx)

	connectorClusterId := mux.Vars(request)["connector_cluster_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), user.AuthorizedClusterUser()),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			listArgs := coreservices.NewListArguments(request.URL.Query())
			resources, paging, err := h.ConnectorNamespace.List(ctx, []string{connectorClusterId}, listArgs, 0)
			if err != nil {
				return nil, err
			}

			resourceList := public.ConnectorNamespaceList{
				Kind:  "ConnectorNamespaceList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters.PresentConnectorNamespace(resource, h.QuotaConfig)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(writer, request, cfg)
}
