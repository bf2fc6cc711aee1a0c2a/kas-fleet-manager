package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
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
	Bus          signalbus.SignalBus
	Service      services.ConnectorNamespaceService
	AuthZService authz.AuthZService
}

func NewConnectorNamespaceHandler(handler ConnectorNamespaceHandler) *ConnectorNamespaceHandler {
	return &handler
}

func (h *ConnectorNamespaceHandler) Create(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	user := h.AuthZService.GetValidationUser(ctx)

	var resource public.ConnectorNamespaceRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Name, handlers.MinLen(1)),
			handlers.Validation("cluster_id", &resource.ClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), user.AuthorizedClusterUser()),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			// validate tenant kind
			if _, ok := presenters.AllNamespaceTenantKinds[string(resource.Kind)]; !ok {
				return nil, coreservices.HandleCreateError("connector namespace",
					errors.MinimumFieldLengthNotReached("%s is not valid. Must be one of: [%s, %s]", "kind",
						public.CONNECTORNAMESPACETENANTKIND_USER, public.CONNECTORNAMESPACETENANTKIND_ORGANISATION))
			}

			userID := user.UserId()
			organisationId := user.OrgId()

			// validate that it's an org admin user for organization tenant namespace
			if resource.Kind == public.CONNECTORNAMESPACETENANTKIND_ORGANISATION {
				if !user.IsOrgAdmin() {
					return nil, errors.Unauthorized("user not authorized")
				}
			}

			convResource, serr := presenters.ConvertConnectorNamespaceRequest(&resource, userID, organisationId)
			if serr != nil {
				return nil, serr
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

	ctx := r.Context()
	user := h.AuthZService.GetValidationUser(ctx)

	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceUser()),
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

	ctx := r.Context()
	user := h.AuthZService.GetValidationUser(ctx)

	var resource public.ConnectorNamespacePatchRequest
	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceAdmin()),
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

	ctx := r.Context()
	user := h.AuthZService.GetValidationUser(ctx)

	connectorNamespaceId := mux.Vars(r)["connector_namespace_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_namespace_id", &connectorNamespaceId,
				handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceAdmin()),
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

			user, err := h.AuthZService.GetUser(ctx)
			if err != nil {
				return nil, err
			}
			userId := user.UserId()
			organisationId := user.OrgId()
			userQuery := fmt.Sprintf("tenant_user_id = '%s' OR tenant_organisation_id = '%s'", userId, organisationId)
			if len(listArgs.Search) == 0 {
				listArgs.Search = userQuery
			} else {
				listArgs.Search = fmt.Sprintf("%s AND (%s)", listArgs.Search, userQuery)
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
