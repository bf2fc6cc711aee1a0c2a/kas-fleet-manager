package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	"io/ioutil"
	"reflect"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/workers"
	"gorm.io/gorm"

	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/goava/di"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreservices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type ConnectorAdminHandler struct {
	di.Inject
	ConnectorsConfig      *config.ConnectorsConfig
	AuthZService          authz.AuthZService
	Service               services.ConnectorClusterService
	ConnectorsService     services.ConnectorsService
	NamespaceService      services.ConnectorNamespaceService
	QuotaConfig           *config.ConnectorsQuotaConfig
	ConnectorCluster      *ConnectorClusterHandler //TODO: eventually move deployment handling into a deployment service
	ConnectorTypesService services.ConnectorTypesService
}

func NewConnectorAdminHandler(handler ConnectorAdminHandler) *ConnectorAdminHandler {
	return &handler
}

func (h *ConnectorAdminHandler) GetConnectorCluster(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["connector_cluster_id"]

	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			cluster, serviceError := h.Service.Get(r.Context(), id)
			if serviceError != nil {
				return nil, serviceError
			}
			return presenters.PresentPrivateConnectorCluster(cluster), nil
		},
	}

	handlers.HandleGet(w, r, &cfg)
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
			for i := range upgrades {
				result[i] = *presenters.PresentConnectorAvailableOperatorUpgrade(&upgrades[i])
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
			for i2 := range resource {
				upgrades[i2] = *presenters.ConvertConnectorAvailableOperatorUpgrade(&resource[i2])
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

			namespaces, paging, err := h.NamespaceService.List(request.Context(), []string{id}, listArgs, 0)
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
				result.Items[i] = presenters.PresentPrivateConnectorNamespace(namespace, h.QuotaConfig)
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

			namespaces, paging, err := h.NamespaceService.List(request.Context(), []string{}, listArgs, 0)
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
				result.Items[i] = presenters.PresentPrivateConnectorNamespace(namespace, h.QuotaConfig)
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
		Validate: []handlers.Validate{
			handlers.Validation("name", &resource.Name, handlers.WithDefault(generateNamespaceName()), handlers.MaxLen(maxConnectorNamespaceNameLength), handlers.Matches(namespaceNamePattern)),
			handlers.Validation("cluster_id", &resource.ClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			ctx := request.Context()
			connectorNamespace, serviceError := presenters.ConvertConnectorNamespaceWithTenantRequest(&resource)
			if serviceError != nil {
				return nil, serviceError
			}

			if connectorNamespace.TenantUser != nil {
				connectorNamespace.Owner = connectorNamespace.TenantUser.ID

				// is eval namespace??
				clusterOrgId, err := h.Service.GetClusterOrg(connectorNamespace.ClusterId)
				if err != nil {
					return nil, err
				}
				if connectorNamespace.Expiration != nil && h.isEvalOrg(clusterOrgId) {
					// check for single evaluation namespace
					if err := h.NamespaceService.CanCreateEvalNamespace(connectorNamespace.Owner); err != nil {
						return nil, err
					}
					if connectorNamespace.ClusterId == "" {
						if err := h.NamespaceService.SetEvalClusterId(connectorNamespace); err != nil {
							return nil, err
						}
					}
				}
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
			i = presenters.PresentPrivateConnectorNamespace(connectorNamespace, h.QuotaConfig)
			return
		},
	}

	handlers.Handle(writer, request, &cfg, http.StatusCreated)
}

func (h *ConnectorAdminHandler) GetConnectorNamespace(writer http.ResponseWriter, request *http.Request) {
	namespaceId := mux.Vars(request)["namespace_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			namespace, serviceError := h.NamespaceService.Get(request.Context(), namespaceId)
			if serviceError != nil {
				return nil, serviceError
			}
			return presenters.PresentPrivateConnectorNamespace(namespace, h.QuotaConfig), nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) DeleteConnectorNamespace(writer http.ResponseWriter, request *http.Request) {
	namespaceId := mux.Vars(request)["namespace_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			ctx := request.Context()
			if parseBoolParam(request.URL.Query().Get("force")) {
				// set namespace status to deleted
				namespace, err := h.NamespaceService.Get(ctx, namespaceId)
				if err != nil {
					return nil, err
				}
				namespace.Status.Phase = dbapi.ConnectorNamespacePhaseDeleted
				if err := h.NamespaceService.UpdateConnectorNamespaceStatus(ctx, namespaceId, &namespace.Status); err != nil {
					return nil, err
				}
			} else {
				serviceError = h.NamespaceService.Delete(ctx, namespaceId)
			}
			return nil, serviceError
		},
	}

	handlers.HandleDelete(writer, request, &cfg, http.StatusNoContent)
}

func (h *ConnectorAdminHandler) GetClusterConnectors(writer http.ResponseWriter, request *http.Request) {
	id := mux.Vars(request)["connector_cluster_id"]
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			connectors, paging, err := h.ConnectorsService.List(request.Context(), listArgs, id)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorAdminViewList{
				Kind:  "ConnectorAdminViewList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorAdminView, len(connectors))
			for i, namespace := range connectors {
				result.Items[i], err = presenters.PresentConnectorAdminView(namespace)
				if err != nil {
					return nil, err
				}
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetNamespaceConnectors(writer http.ResponseWriter, request *http.Request) {
	id := mux.Vars(request)["namespace_id"]
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("namespace_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			if len(listArgs.Search) == 0 {
				listArgs.Search = fmt.Sprintf("namespace_id = %s", id)
			} else {
				listArgs.Search = fmt.Sprintf("namespace_id = %s AND (%s)", id, listArgs.Search)
			}
			connectors, paging, err := h.ConnectorsService.List(request.Context(), listArgs, "")
			if err != nil {
				return nil, err
			}

			result := private.ConnectorAdminViewList{
				Kind:  "ConnectorAdminViewList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorAdminView, len(connectors))
			for i, namespace := range connectors {
				result.Items[i], err = presenters.PresentConnectorAdminView(namespace)
				if err != nil {
					return nil, err
				}
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetConnector(writer http.ResponseWriter, request *http.Request) {
	connectorId := mux.Vars(request)["connector_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			connector, serviceError := h.ConnectorsService.Get(request.Context(), connectorId, "")
			if serviceError != nil {
				return nil, serviceError
			}
			return presenters.PresentConnectorAdminView(connector)
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetOperation(resource public.Connector, patch private.ConnectorRequest) (phase.ConnectorOperation, *errors.ServiceError) {
	operation, ok := stateToOperationsMap[public.ConnectorDesiredState(patch.DesiredState)]
	if !ok {
		return operation, errors.BadRequest("Unsupported patch desired state %s", patch.DesiredState)
	}
	if string(patch.DesiredState) == string(resource.DesiredState) {
		// desired state not changing, it's an update
		operation = phase.UpdateConnector
	} else {
		// assigning a stopped connector is a restart
		if operation == phase.AssignConnector && resource.DesiredState == public.CONNECTORDESIREDSTATE_STOPPED {
			operation = phase.RestartConnector
		}
	}
	return operation, nil
}

func (h *ConnectorAdminHandler) PatchConnector(writer http.ResponseWriter, request *http.Request) {
	connectorId := mux.Vars(request)["connector_id"]
	contentType := request.Header.Get("Content-Type")

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
			handlers.Validation("Content-Type header", &contentType, handlers.IsOneOf("application/json", "application/json-patch+json", "application/merge-patch+json")),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			connector, serviceError := h.ConnectorsService.Get(request.Context(), connectorId, "")
			if serviceError != nil {
				return nil, serviceError
			}

			resource, serviceError := presenters.PresentConnector(&connector.Connector)
			if serviceError != nil {
				return nil, serviceError
			}

			// Apply the patch..
			patchBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return nil, errors.BadRequest("failed to get patch bytes")
			}

			patch := private.ConnectorRequest{}
			serviceError = PatchResource(resource, contentType, patchBytes, &patch)
			if serviceError != nil {
				return nil, serviceError
			}

			// get and validate patch operation type
			var operation phase.ConnectorOperation
			if operation, serviceError = h.GetOperation(resource, patch); err != nil {
				return nil, serviceError
			}
			if operation == phase.UnassignConnector && !h.ConnectorsConfig.ConnectorEnableUnassignedConnectors {
				return nil, errors.FieldValidationError("Unsupported connector state %s", patch.DesiredState)
			}
			if serviceError = ValidateConnectorOperation(request.Context(), h.NamespaceService, &connector.Connector, operation,
				func(connector *dbapi.Connector) *errors.ServiceError {
					resource.DesiredState = public.ConnectorDesiredState(connector.DesiredState)
					return nil
				}); serviceError != nil {
				return nil, serviceError
			}

			// But we don't want to allow the user to update ALL fields. Copy
			// over the fields that they are allowed to modify, in this case only DesiredState
			resource.DesiredState = public.ConnectorDesiredState(patch.DesiredState)

			// If we didn't change anything, then just skip the update...
			originalResource, _ := presenters.PresentConnector(&connector.Connector)
			if reflect.DeepEqual(originalResource, resource) {
				return originalResource, nil
			}

			// revalidate
			validates := []handlers.Validate{
				handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.IsOneOf(dbapi.ValidDesiredStates...)),
			}

			for _, v := range validates {
				err := v()
				if err != nil {
					return nil, err
				}
			}

			p, svcErr := presenters.ConvertConnector(resource)
			if svcErr != nil {
				return nil, svcErr
			}

			// update connector phase before desired state
			if originalResource.Status.State != public.ConnectorState(dbapi.ConnectorStatusPhaseAssigning) {
				connector.Status.Phase = phase.ConnectorStartingPhase[operation]
				p.Status.Phase = connector.Status.Phase
				serviceError = h.ConnectorsService.SaveStatus(request.Context(), connector.Status)
				if serviceError != nil {
					return nil, serviceError
				}
			}
			// update modified connector including desired state
			serviceError = h.ConnectorsService.Update(request.Context(), p)
			if serviceError != nil {
				return nil, serviceError
			}

			return presenters.PresentConnector(p)
		},
	}

	// return 202 status accepted
	handlers.Handle(writer, request, cfg, http.StatusAccepted)
}

func (h *ConnectorAdminHandler) DeleteConnector(writer http.ResponseWriter, request *http.Request) {
	connectorId := mux.Vars(request)["connector_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			// check force flag to force deletion of connector and deployments
			if parseBoolParam(request.URL.Query().Get("force")) {
				serviceError = h.ConnectorsService.ForceDelete(request.Context(), connectorId)
			} else {
				ctx := request.Context()
				return nil, HandleConnectorDelete(ctx, h.ConnectorsService, h.NamespaceService, connectorId)
			}
			return nil, serviceError
		},
	}

	handlers.HandleDelete(writer, request, &cfg, http.StatusNoContent)
}

func (h *ConnectorAdminHandler) GetClusterDeployments(writer http.ResponseWriter, request *http.Request) {
	clusterId := mux.Vars(request)["connector_cluster_id"]
	channelUpdates := request.URL.Query().Get("channel_updates")
	danglingDeployments := request.URL.Query().Get("dangling_deployments")
	listArgs := coreservices.NewListArguments(request.URL.Query())
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &clusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			deployments, paging, err := h.Service.ListConnectorDeployments(request.Context(), clusterId, parseBoolParam(channelUpdates), parseBoolParam(danglingDeployments), listArgs, 0)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorDeploymentAdminViewList{
				Kind:  "ConnectorDeploymentAdminViewList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorDeploymentAdminView, len(deployments))
			for i, deployment := range deployments {
				deploymentAdminView, err := presenters.PresentConnectorDeploymentAdminView(deployment, clusterId)
				if err != nil {
					return nil, err
				}
				result.Items[i] = deploymentAdminView
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func parseBoolParam(channelUpdates string) bool {
	result, _ := strconv.ParseBool(channelUpdates)
	return result
}

func (h *ConnectorAdminHandler) GetConnectorDeployment(writer http.ResponseWriter, request *http.Request) {
	clusterId := mux.Vars(request)["connector_cluster_id"]
	deploymentId := mux.Vars(request)["deployment_id"]
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &clusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
			handlers.Validation("deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			deployment, serviceError := h.Service.GetDeployment(request.Context(), deploymentId)
			if serviceError != nil {
				return nil, serviceError
			}
			// check if the cluster ids match
			if deployment.ClusterID != clusterId {
				return nil, coreservices.HandleGetError(`Connector deployment`, `id`, deploymentId, gorm.ErrRecordNotFound)
			}
			return presenters.PresentConnectorDeploymentAdminView(deployment, clusterId)
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetNamespaceDeployments(writer http.ResponseWriter, request *http.Request) {
	namespaceId := mux.Vars(request)["namespace_id"]
	channelUpdates := request.URL.Query().Get("channel_updates")
	listArgs := coreservices.NewListArguments(request.URL.Query())
	danglingDeployments := request.URL.Query().Get("dangling_deployments")
	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			if len(listArgs.Search) == 0 {
				listArgs.Search = fmt.Sprintf("namespace_id = %s", namespaceId)
			} else {
				listArgs.Search = fmt.Sprintf("namespace_id = %s AND (%s)", namespaceId, listArgs.Search)
			}
			deployments, paging, err := h.Service.ListConnectorDeployments(request.Context(), "", parseBoolParam(channelUpdates), parseBoolParam(danglingDeployments), listArgs, 0)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorDeploymentAdminViewList{
				Kind:  "ConnectorDeploymentAdminViewList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorDeploymentAdminView, len(deployments))
			for i, deployment := range deployments {
				deploymentAdminView, err := presenters.PresentConnectorDeploymentAdminView(deployment, deployment.ClusterID)
				if err != nil {
					return nil, err
				}
				result.Items[i] = deploymentAdminView
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) ListConnectorTypes(writer http.ResponseWriter, request *http.Request) {
	listArgs := coreservices.NewListArguments(request.URL.Query())

	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			entries, paging, err := h.ConnectorTypesService.ListCatalogEntries(listArgs)
			if err != nil {
				return nil, err
			}

			result := private.ConnectorTypeAdminViewList{
				Kind:  "ConnectorTypeAdminViewList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			result.Items = make([]private.ConnectorTypeAdminView, len(entries))
			for i, deployment := range entries {
				item, err := presenters.PresentConnectorTypeAdminView(deployment)
				if err != nil {
					return nil, err
				}
				if item != nil {
					result.Items[i] = *item
				}
			}

			return result, nil
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) GetConnectorType(writer http.ResponseWriter, request *http.Request) {
	id := mux.Vars(request)["connector_type_id"]

	cfg := handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_type_id", &id, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			entry, err := h.ConnectorTypesService.GetCatalogEntry(id)
			if err != nil {
				return nil, err
			}
			if entry == nil {
				return nil, nil
			}

			return presenters.PresentConnectorTypeAdminView(*entry)
		},
	}

	handlers.HandleGet(writer, request, &cfg)
}

func (h *ConnectorAdminHandler) PatchConnectorDeployment(writer http.ResponseWriter, request *http.Request) {
	clusterId := mux.Vars(request)["connector_cluster_id"]
	deploymentId := mux.Vars(request)["deployment_id"]
	var resource private.ConnectorDeploymentAdminView
	cfg := handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &clusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
			handlers.Validation("deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			// Get existing deployment
			existingDeployment, serviceError := h.Service.GetDeployment(request.Context(), deploymentId)
			if serviceError != nil {
				return nil, serviceError
			}
			// check if the cluster ids match
			if existingDeployment.ClusterID != clusterId {
				return nil, coreservices.HandleGetError(`Connector existingDeployment`, `id`, deploymentId, gorm.ErrRecordNotFound)
			}

			// Handle the fields that support being updated...
			var updatedDeployment dbapi.ConnectorDeployment
			updatedDeployment.ID = existingDeployment.ID
			if len(resource.Spec.ShardMetadata) != 0 {
				// channel update
				updateRevision, err := workers.GetShardMetadataRevision(resource.Spec.ShardMetadata)
				if err != nil {
					return nil, errors.GeneralError("Error in patching deployment, updateRevision not found in shardMetadata %+v: %v", resource.Spec.ShardMetadata, err.Error())
				}
				updateShardMetadata, serr := h.ConnectorTypesService.GetConnectorShardMetadata(existingDeployment.ConnectorShardMetadata.ConnectorTypeId, existingDeployment.ConnectorShardMetadata.Channel, updateRevision)
				if serr != nil {
					return nil, errors.GeneralError("Error in patching deployment, shardMetadata %+v not found: %v", resource.Spec.ShardMetadata, err.Error())
				}
				updatedDeployment.ConnectorShardMetadataID = updateShardMetadata.ID
				updatedDeployment.ConnectorShardMetadata = *updateShardMetadata
			} else {
				// Spec.ShardMetadata is the only updatable field for now
				return presenters.PresentConnectorDeploymentAdminView(existingDeployment, clusterId)
			}

			// update
			uerr := h.Service.UpdateDeployment(&updatedDeployment)
			if uerr != nil {
				return nil, uerr
			}

			// read updated deployment back
			existingDeployment, serviceError = h.Service.GetDeployment(request.Context(), deploymentId)
			if serviceError != nil {
				return nil, serviceError
			}
			return presenters.PresentConnectorDeploymentAdminView(existingDeployment, clusterId)
		},
	}

	handlers.Handle(writer, request, &cfg, http.StatusAccepted)
}

func (h *ConnectorAdminHandler) isEvalOrg(id string) bool {
	for _, eid := range h.ConnectorsConfig.ConnectorEvalOrganizations {
		if id == eid {
			return true
		}
	}
	return false
}
