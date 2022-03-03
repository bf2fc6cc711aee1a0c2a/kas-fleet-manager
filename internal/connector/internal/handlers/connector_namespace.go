package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/goava/di"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorNamespaceIdLength = 32
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
			handlers.Validation("name", &resource.Name, handlers.MinLen(1)),
			handlers.Validation("cluster_id", &resource.ClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			// validate tenant kind
			if _, ok := presenters.AllNamespaceTenantKinds[string(resource.Kind)]; !ok {
				return nil, coreservices.HandleCreateError("connector namespace",
					errors.MinimumFieldLengthNotReached("%s is not valid. Must be one of: [%s, %s]", "namespace_id",
						public.CONNECTORNAMESPACETENANTKIND_USER, public.CONNECTORNAMESPACETENANTKIND_ORGANISATION))
			}
			ctx := r.Context()
			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			userID := auth.GetUsernameFromClaims(claims)
			organisationId := auth.GetOrgIdFromClaims(claims)

			convResource, serr := presenters.ConvertConnectorNamespaceRequest(&resource, userID, organisationId)
			if serr != nil {
				return nil, serr
			}
			if err := h.checkAuthorizedClusterAccess(userID, convResource, organisationId); err != nil {
				return nil, err
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

func (h *ConnectorNamespaceHandler) checkAuthorizedClusterAccess(userID string, convResource *dbapi.ConnectorNamespace,
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
			handlers.Validation("name", &resource.Name, handlers.MinLen(1)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			userId := auth.GetUsernameFromClaims(claims)

			convResource := presenters.ConvertConnectorNamespaceEvalRequest(&resource, userId)
			if err := h.Service.SetEvalClusterId(convResource); err != nil {
				return nil, err
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
	var resource public.ConnectorNamespacePatchRequest

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
			if len(resource.Name) != 0 {
				existing.Name = resource.Name
			} else {
				// name is the only updatable field for now
				return nil, nil
			}

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
			err := h.Service.Delete(connectorNamespaceId)
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
			userId := auth.GetUsernameFromClaims(claims)
			organisationId := auth.GetOrgIdFromClaims(claims)

			userQuery := fmt.Sprintf("owner = '%s' or tenant_user_id = '%s' or tenant_organisation_id = '%s'",
				userId, userId, organisationId)
			if len(listArgs.Search) == 0 {
				listArgs.Search = userQuery
			} else {
				listArgs.Search = fmt.Sprintf("%s AND %s", listArgs.Search, userQuery)
			}

			resources, paging, serviceError := h.Service.List(ctx, nil, listArgs)
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
