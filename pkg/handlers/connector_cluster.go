package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorClusterIdLength = 32
)

type connectorClusterHandler struct {
	service         services.ConnectorClusterService
	configService   services.ConfigService
	keycloakService services.KeycloakService
}

func NewConnectorClusterHandler(service services.ConnectorClusterService, configService services.ConfigService, keycloakService services.KeycloakService) *connectorClusterHandler {
	return &connectorClusterHandler{
		service:         service,
		configService:   configService,
		keycloakService: keycloakService,
	}
}

func (h *connectorClusterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource openapi.ConnectorCluster
	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("name", &resource.Metadata.Name,
				withDefault("New Cluster"), minLen(1), maxLen(100)),
			validation("group", &resource.Metadata.Group,
				withDefault("default"), minLen(1), maxLen(100)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource, serr := presenters.ConvertConnectorCluster(resource)
			if serr != nil {
				return nil, serr
			}

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convResource.Owner = auth.GetUsernameFromClaims(claims)
			convResource.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convResource.Status = api.ConnectorClusterStatusUnconnected

			if err := h.service.Create(r.Context(), convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(convResource)
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func (h *connectorClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.Get(r.Context(), id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(resource)
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func (h *connectorClusterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			err := h.service.Delete(r.Context(), id)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (h *connectorClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
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
				converted, err := presenters.PresentConnectorCluster(resource)
				if err != nil {
					glog.Errorf("connector cluster id='%s' presentation failed: %v", resource.ID, err)
					return nil, errors.GeneralError("internal error")
				}
				resourceList.Items = append(resourceList.Items, converted)

			}

			return resourceList, nil
		},
		ErrorHandler: handleError,
	}

	handleList(w, r, cfg)
}

func (h *connectorClusterHandler) GetAddonParameters(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			// To make sure the user can access the cluster....
			_, err := h.service.Get(r.Context(), id)
			if err != nil {
				return nil, err
			}

			acc, err := h.keycloakService.RegisterConnectorFleetshardOperatorServiceAccount(id, connectorFleetshardOperatorRoleName)
			if err != nil {
				return false, errors.GeneralError("failed to create service account for connector cluster %s due to error: %v", id, err)
			}
			params := h.buildAddonParams(acc, id)
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
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
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

func (o *connectorClusterHandler) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{
		{
			Id:    connectorFleetshardOperatorParamMasSSOBaseUrl,
			Value: o.configService.GetConfig().Keycloak.BaseURL,
		},
		{
			Id:    connectorFleetshardOperatorParamMasSSORealm,
			Value: o.configService.GetConfig().Keycloak.KafkaRealm.Realm,
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
			Value: o.configService.GetConfig().Server.PublicHostURL,
		},
		{
			Id:    connectorFleetshardOperatorParamClusterId,
			Value: clusterId,
		},
	}
	return p
}

func (o *connectorClusterHandler) buildTokenURL(serviceAccount *api.ServiceAccount) (string, error) {
	u, err := url.Parse(o.configService.GetConfig().Keycloak.KafkaRealm.TokenEndpointURI)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(serviceAccount.ClientID, serviceAccount.ClientSecret)
	return u.String(), nil
}

func (h *connectorClusterHandler) UpdateConnectorClusterStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var resource openapi.ConnectorClusterUpdateStatus

	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("status", &resource.Status, isOneOf(api.AllConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := h.service.UpdateConnectorClusterStatus(ctx, id, resource.Status)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handle(w, r, cfg, http.StatusNoContent)
}

func (h *connectorClusterHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
	// h.service.ListConnectors()
	id := mux.Vars(r)["id"]

	cfg := &handlerConfig{
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			ctx := r.Context()
			query := r.URL.Query()

			gtVersion := int64(0)
			if v := query.Get("gt_version"); v != "" {
				gtVersion, _ = strconv.ParseInt(v, 10, 0)
			}

			listArgs := services.NewListArguments(query)
			resources, paging, err := h.service.ListConnectors(ctx, id, listArgs, gtVersion)
			if err != nil {
				return nil, err
			}

			resourceList := openapi.ConnectorList{
				Kind:  "ConnectorList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted, err := presenters.PresentConnector(resource)
				if err != nil {
					glog.Errorf("connector id='%s' presentation failed: %v", resource.ID, err)
					return nil, errors.GeneralError("internal error")
				}
				resourceList.Items = append(resourceList.Items, converted)

			}

			return resourceList, nil
		},
		ErrorHandler: handleError,
	}
	handleList(w, r, cfg)
}

func (h *connectorClusterHandler) UpdateConnectorStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cid := mux.Vars(r)["cid"]
	var resource openapi.ConnectorUpdateStatus

	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("id", &id, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("cid", &id, minLen(1), maxLen(maxConnectorIdLength)),
			validation("status", &resource.Status, isOneOf(api.AllConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := h.service.UpdateConnectorStatus(ctx, id, cid, resource.Status)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handle(w, r, cfg, http.StatusNoContent)
}
