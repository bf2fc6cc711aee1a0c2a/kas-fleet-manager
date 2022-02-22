package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/goava/di"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorNamespaceIdLength   = 32
	maxConnectorNamespaceNameLength = 63
	defaultNamespaceName            = "default_namespace"
)

type ConnectorNamespaceHandler struct {
	di.Inject
	Bus     signalbus.SignalBus
	Service services.ConnectorNamespaceService
}

func NewConnectorNamespaceHandler(handler ConnectorNamespaceHandler) *ConnectorNamespaceHandler {
	return &handler
}

func (h *ConnectorNamespaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource public.ConnectorNamespaceRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Name, handlers.WithDefault(defaultNamespaceName),
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceNameLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters.ConvertConnectorNamespaceRequest(&resource)

			ctx := r.Context()
			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			userID := auth.GetUsernameFromClaims(claims)
			convResource.Owner = userID
			organisationId := auth.GetOrgIdFromClaims(claims)
			if convResource.ClusterId == "" {
				if err := h.Service.SetTenantClusterId(convResource, organisationId); err != nil {
					return nil, err
				}
			} else {
				// TODO move auth checks in an orthogonal feature
				if err := h.checkAuthorizedAccess(userID, convResource, organisationId); err != nil {
					return nil, err
				}
			}

			// set tenant to org if there is one, or set it to user
			if auth.GetFilterByOrganisationFromContext(ctx) {
				convResource.TenantOrganisationId = &organisationId
				convResource.TenantOrganisation = &dbapi.ConnectorTenantOrganisation{
					Model: db.Model{
						ID: organisationId,
					},
				}
			} else {
				convResource.TenantUserId = &userID
				convResource.TenantUser = &dbapi.ConnectorTenantUser{
					Model: db.Model{
						ID: userID,
					},
				}
			}

			if err := h.Service.Create(ctx, convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnectorNamespace(convResource), nil
		},
	}

	// return 201 status created
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h *ConnectorNamespaceHandler) checkAuthorizedAccess(userID string, convResource *dbapi.ConnectorNamespace,
	organisationId string) *errors.ServiceError {

	var ownerClusterIds []string
	if err := h.Service.GetOwnerClusterIds(userID, &ownerClusterIds); err != nil {
		return err
	}
	authorized := false
	for _, id := range ownerClusterIds {
		if id == convResource.ClusterId {
			authorized = true
			break
		}
	}
	if !authorized {
		var orgClusterIds []string
		if err := h.Service.GetOrgClusterIds(userID, organisationId, &orgClusterIds); err != nil {
			return err
		}
		for _, id := range orgClusterIds {
			if id == convResource.ClusterId {
				authorized = true
				break
			}
		}
		if !authorized {
			return errors.Unauthorized(
				"user %v is not authorized to create namespace in cluster %v",
				userID, convResource.ClusterId)
		}
	}

	return nil
}

func (h *ConnectorNamespaceHandler) CreateEvaluation(w http.ResponseWriter, r *http.Request) {
	var resource public.ConnectorNamespaceEvalRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Name, handlers.WithDefault(defaultNamespaceName),
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceNameLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters.ConvertConnectorNamespaceEvalRequest(&resource)

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			userId := auth.GetUsernameFromClaims(claims)
			convResource.Owner = userId
			if err := h.Service.SetEvalClusterId(convResource); err != nil {
				return nil, err
			}

			// set tenant user for eval namespace
			convResource.TenantUserId = &convResource.Owner
			convResource.TenantUser = &dbapi.ConnectorTenantUser{
				Model: db.Model{
					ID: userId,
				},
			}
			if err := h.Service.Create(r.Context(), convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnectorNamespace(convResource), nil
		},
	}

	// return 201 status created
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h *ConnectorNamespaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.Service.Get(r.Context(), connectorNamespaceId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorNamespace(resource), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorNamespaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	var resource public.ConnectorNamespaceRequest

	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			existing, err := h.Service.Get(r.Context(), connectorNamespaceId)
			if err != nil {
				return nil, err
			}

			// Copy over the fields that support being updated...
			existing.Name = resource.Name

			return nil, h.Service.Update(r.Context(), existing)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorNamespaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			err := h.Service.Delete(r.Context(), connectorNamespaceId)
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorNamespaceHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			ctx := r.Context()
			listArgs := coreservices.NewListArguments(r.URL.Query())

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			// TODO handle creating default namespaces in List()
			userId := auth.GetUsernameFromClaims(claims)
			var ownerClusterIds []string
			if err := h.Service.GetOwnerClusterIds(userId, &ownerClusterIds); err != nil {
				return nil, err
			}
			organisationId := auth.GetOrgIdFromClaims(claims)
			var orgClusterIds []string
			if err := h.Service.GetOrgClusterIds(userId, organisationId, &orgClusterIds); err != nil {
				return nil, err
			}

			clusterIds := append(ownerClusterIds, orgClusterIds...)
			if len(clusterIds) == 0 {
				// user doesn't have access to any clusters
				return public.ConnectorNamespaceList{
					Kind:  "ConnectorNamespaceList",
					Page:  1,
					Size:  0,
					Total: 0,
					Items: make([]public.ConnectorNamespace, 0),
				}, nil
			}

			resources, paging, serviceError := h.Service.List(ctx, clusterIds, listArgs)
			if serviceError != nil {
				return nil, serviceError
			}

			items := make([]public.ConnectorNamespace, len(resources))
			for j, resource := range resources {
				items[j] = presenters.PresentConnectorNamespace(resource)
			}
			resourceList := public.ConnectorNamespaceList{
				Kind:  "ConnectorNamespaceList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: items,
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
