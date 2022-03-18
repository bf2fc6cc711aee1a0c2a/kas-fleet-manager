package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/goava/di"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type ConnectorAdminHandler struct {
	di.Inject
	Bus              signalbus.SignalBus
	AuthZService     authz.AuthZService
	Service          services.ConnectorClusterService
	NamespaceService services.ConnectorNamespaceService
	Keycloak         sso.KafkaKeycloakService
	ConnectorTypes   services.ConnectorTypesService
	Vault            vault.VaultService
	KeycloakConfig   *keycloak.KeycloakConfig
	ServerConfig     *server.ServerConfig
}

func NewConnectorAdminHandler(handler ConnectorAdminHandler) *ConnectorAdminHandler {
	return &handler
}

func (h *ConnectorAdminHandler) ListConnectorClusters(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {

			listArgs := coreservices.NewListArguments(r.URL.Query())
			resources, paging, err := h.Service.List(r.Context(), listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := private.ConnectorClusterList{
				Kind:  "ConnectorClusterList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			resourceList.Items = make([]private.ConnectorCluster, len(resources))
			for i, resource := range resources {
				resourceList.Items[i] = presenters.PresentPrivateConnectorCluster(resource)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h *ConnectorAdminHandler) GetConnectorUpgradesByType(writer http.ResponseWriter, request *http.Request) {

	id := mux.Vars(request)["connector_cluster_id"]
	listArgs := coreservices.NewListArguments(request.URL.Query())

	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			upgrades, paging, serviceError := h.Service.GetAvailableDeploymentTypeUpgrades(listArgs)
			if serviceError != nil {
				return nil, serviceError
			}
			result := make([]private.ConnectorAvailableTypeUpgrade, len(upgrades))
			for j, upgrade := range upgrades {
				result[j] = *presenters.PresentConnectorAvailableTypeUpgrade(&upgrade)
			}

			i = private.ConnectorAvailableTypeUpgradeList{
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: result,
			}
			return
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) UpgradeConnectorsByType(writer http.ResponseWriter, request *http.Request) {
	resource := make([]private.ConnectorAvailableTypeUpgrade, 0)
	id := mux.Vars(request)["connector_cluster_id"]
	cfg := handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			upgrades := make([]dbapi.ConnectorDeploymentTypeUpgrade, len(resource))
			for i2, upgrade := range resource {
				upgrades[i2] = *presenters.ConvertConnectorAvailableTypeUpgrade(&upgrade)
			}
			return nil, h.Service.UpgradeConnectorsByType(request.Context(), id, upgrades)
		},
	}
	handlers.Handle(writer, request, &cfg, http.StatusNoContent)
}

func (h *ConnectorAdminHandler) GetConnectorUpgradesByOperator(writer http.ResponseWriter, request *http.Request) {
	id := mux.Vars(request)["connector_cluster_id"]
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			upgrades, paging, serviceError := h.Service.GetAvailableDeploymentOperatorUpgrades(listArgs)
			if serviceError != nil {
				return nil, serviceError
			}
			result := make([]private.ConnectorAvailableOperatorUpgrade, len(upgrades))
			for i, upgrade := range upgrades {
				result[i] = *presenters.PresentConnectorAvailableOperatorUpgrade(&upgrade)
			}

			i = private.ConnectorAvailableOperatorUpgradeList{
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: result,
			}
			return
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) UpgradeConnectorsByOperator(writer http.ResponseWriter, request *http.Request) {
	var resource []private.ConnectorAvailableOperatorUpgrade
	id := mux.Vars(request)["connector_cluster_id"]
	cfg := handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			upgrades := make(dbapi.ConnectorDeploymentOperatorUpgradeList, len(resource))
			for i2, upgrade := range resource {
				upgrades[i2] = *presenters.ConvertConnectorAvailableOperatorUpgrade(&upgrade)
			}
			return nil, h.Service.UpgradeConnectorsByOperator(request.Context(), id, upgrades)
		},
	}

	handlers.Handle(writer, request, &cfg, http.StatusNoContent)
}

func (h *ConnectorAdminHandler) GetClusterNamespaces(writer http.ResponseWriter, request *http.Request) {
	id := mux.Vars(request)["connector_cluster_id"]
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			namespaces, paging, err := h.NamespaceService.List(request.Context(), []string{id}, listArgs)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorNamespaceList{
				Kind:  "ConnectorNamespaceList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorNamespace, len(namespaces))
			for i, namespace := range namespaces {
				result.Items[i] = presenters.PresentPrivateConnectorNamespace(namespace)
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetConnectorNamespaces(writer http.ResponseWriter, request *http.Request) {
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {

			namespaces, paging, err := h.NamespaceService.List(request.Context(), []string{}, listArgs)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorNamespaceList{
				Kind:  "ConnectorNamespaceList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorNamespace, len(namespaces))
			for i, namespace := range namespaces {
				result.Items[i] = presenters.PresentPrivateConnectorNamespace(namespace)
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) CreateConnectorNamespace(writer http.ResponseWriter, request *http.Request) {
	var resource private.ConnectorNamespaceWithTenantRequest
	cfg := handlers.HandlerConfig{
		MarshalInto: &resource,
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			ctx := request.Context()
			connectorNamespace, serviceError := presenters.ConvertConnectorNamespaceWithTenantRequest(&resource)
			if serviceError != nil {
				return nil, serviceError
			}
			if connectorNamespace.TenantUser != nil {
				connectorNamespace.Owner = connectorNamespace.TenantUser.ID
			} else {
				// NOTE: admin user is owner
				user, err := h.AuthZService.GetUser(ctx)
				if err != nil {
					return nil, err
				}
				connectorNamespace.Owner = user.UserId()
			}
			if err := h.NamespaceService.Create(ctx, connectorNamespace); err != nil {
				return nil, err
			}
			i = presenters.PresentPrivateConnectorNamespace(connectorNamespace)
			return
		},
	}

	handlers.Handle(writer, request, &cfg, http.StatusCreated)
}

func (h *ConnectorAdminHandler) DeleteConnectorNamespace(writer http.ResponseWriter, request *http.Request) {
	namespaceId := mux.Vars(request)["namespace_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			serviceError = h.NamespaceService.Delete(request.Context(), namespaceId)
			return nil, serviceError
		},
	}

	handlers.HandleDelete(writer, request, &cfg, http.StatusNoContent)
}
