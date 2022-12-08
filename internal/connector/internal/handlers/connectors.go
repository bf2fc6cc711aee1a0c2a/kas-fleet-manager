package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/spyzhov/ajson"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	maxKafkaNameLength   = 32
	maxConnectorIdLength = 32
	APPLICATION_JSON     = "application/json"
	JSON_PATCH           = "application/json-patch+json"
	MERGE_PATCH          = "application/merge-patch+json"
)

type ConnectorsHandler struct {
	connectorsService     services.ConnectorsService
	connectorTypesService services.ConnectorTypesService
	namespaceService      services.ConnectorNamespaceService
	vaultService          vault.VaultService
	authZService          authz.AuthZService
	connectorsConfig      *config.ConnectorsConfig
}

// this is an initial guess at what operation is being performed in update
var stateToOperationsMap = map[public.ConnectorDesiredState]phase.ConnectorOperation{
	public.CONNECTORDESIREDSTATE_UNASSIGNED: phase.UnassignConnector,
	public.CONNECTORDESIREDSTATE_READY:      phase.AssignConnector,
	public.CONNECTORDESIREDSTATE_STOPPED:    phase.StopConnector,
}

func NewConnectorsHandler(connectorsService services.ConnectorsService, connectorTypesService services.ConnectorTypesService,
	namespaceService services.ConnectorNamespaceService, vaultService vault.VaultService, authZService authz.AuthZService,
	connectorsConfig *config.ConnectorsConfig) *ConnectorsHandler {
	return &ConnectorsHandler{
		connectorsService:     connectorsService,
		connectorTypesService: connectorTypesService,
		namespaceService:      namespaceService,
		vaultService:          vaultService,
		authZService:          authZService,
		connectorsConfig:      connectorsConfig,
	}
}

func (h ConnectorsHandler) Create(w http.ResponseWriter, r *http.Request) {

	user := h.authZService.GetValidationUser(r.Context())

	var resource public.ConnectorRequest
	cfg := &handlers.HandlerConfig{

		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating connector"),
			handlers.Validation("channel", (*string)(&resource.Channel), handlers.WithDefault("stable"), handlers.MaxLen(40)),
			handlers.Validation("name", &resource.Name, handlers.WithDefault("New Connector"), handlers.MinLen(1), handlers.MaxLen(100)),
			handlers.Validation("kafka.id", &resource.Kafka.Id, handlers.MinLen(1), handlers.MaxLen(maxKafkaNameLength)),
			handlers.Validation("kafka.url", &resource.Kafka.Url, handlers.MinLen(1)),
			handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
			handlers.Validation("service_account.client_secret", &resource.ServiceAccount.ClientSecret, handlers.MinLen(1)),
			handlers.Validation("connector_type_id", &resource.ConnectorTypeId, handlers.MinLen(1), handlers.MaxLen(maxConnectorTypeIdLength)),
			handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.WithDefault("ready"), handlers.IsOneOf(dbapi.ValidDesiredStates...)),
			validateConnectorRequest(h.connectorTypesService, &resource),
			handlers.Validation("namespace_id", &resource.NamespaceId,
				handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceUser(errors.ErrorBadRequest), user.ValidateNamespaceConnectorQuota()),
			validateCreateAnnotations(resource.Annotations),
		},

		Action: func() (interface{}, *errors.ServiceError) {

			// validate type id first
			ct, err := h.connectorTypesService.Get(resource.ConnectorTypeId)
			if err != nil {
				return nil, errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
			}

			newID := api.NewID()
			addSystemAnnotations(&resource.Annotations, user)
			// copy type annotations to connector, e.g. for pricing
			for _, a := range ct.Annotations {
				resource.Annotations[a.Key] = a.Value
			}

			convResource, err := presenters.ConvertConnectorRequest(newID, resource)
			if err != nil {
				return nil, err
			}

			convResource.Owner = user.UserId()
			convResource.OrganisationId = user.OrgId()

			// namespace id is a required field if unassigned connectors are not supported
			if !h.connectorsConfig.ConnectorEnableUnassignedConnectors && (convResource.NamespaceId == nil || *convResource.NamespaceId == "") {
				return nil, errors.MinimumFieldLengthNotReached("namespace_id is not valid. Minimum length 1 is required.")
			}
			if err := ValidateConnectorOperation(r.Context(), h.namespaceService, convResource, phase.CreateConnector); err != nil {
				return nil, err
			}

			err = moveSecretsToVault(convResource, ct, h.vaultService, true)
			if err != nil {
				return nil, err
			}

			if svcErr := h.connectorsService.Create(r.Context(), convResource); svcErr != nil {
				return nil, svcErr
			}

			if err := stripSecretReferences(convResource, ct); err != nil {
				return nil, err
			}

			return presenters.PresentConnector(convResource)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h ConnectorsHandler) Patch(w http.ResponseWriter, r *http.Request) {

	connectorId := mux.Vars(r)["connector_id"]
	contentType := r.Header.Get("Content-Type")

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
			handlers.Validation("Content-Type header", &contentType, handlers.IsOneOf(APPLICATION_JSON, JSON_PATCH, MERGE_PATCH)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			dbresource, serr := h.connectorsService.Get(r.Context(), connectorId)
			if serr != nil {
				return nil, serr
			}
			originalResource, _ := presenters.PresentConnector(&dbresource.Connector)

			resource, serr := presenters.PresentConnector(&dbresource.Connector)
			if serr != nil {
				return nil, serr
			}

			ct, serr := h.connectorTypesService.Get(dbresource.ConnectorTypeId)
			if serr != nil {
				return nil, errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
			}

			originalSecrets, err := getSecretRefs(&dbresource.Connector, ct)
			if err != nil {
				return nil, errors.GeneralError("could not get existing secrets: %v", err)
			}

			// Apply the patch..
			patchBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, errors.BadRequest("failed to get patch bytes")
			}

			// get json resource
			resourceJson, serr := getResourceJson(resource)
			if serr != nil {
				return nil, serr
			}

			// convert json-patch into merge-patch, since validateConnectorPatch can only validate merge-patch
			if contentType == JSON_PATCH {
				patchBytes, err = convertToMergePatch(patchBytes, resourceJson)
				if err != nil {
					return nil, errors.BadRequest("failed to convert to merge patch: %v", err)
				}
				// changed patch bytes to merge patch
				contentType = MERGE_PATCH
			}

			// Don't allow updating connector secrets with values like {"ref": "something"}
			serr = validateConnectorPatch(patchBytes, ct)
			if serr != nil {
				return nil, serr
			}

			patch := public.ConnectorRequest{}
			serr = PatchResource(resourceJson, contentType, patchBytes, &patch)
			if serr != nil {
				return nil, serr
			}

			// get and validate patch operation type
			var operation phase.ConnectorOperation
			if operation, serr = h.getOperation(resource, patch); err != nil {
				return nil, serr
			}
			if operation == phase.UnassignConnector && !h.connectorsConfig.ConnectorEnableUnassignedConnectors {
				return nil, errors.FieldValidationError("Unsupported connector state %s", patch.DesiredState)
			}
			if serr = ValidateConnectorOperation(r.Context(), h.namespaceService, &dbresource.Connector, operation,
				func(connector *dbapi.Connector) *errors.ServiceError {
					resource.DesiredState = public.ConnectorDesiredState(dbresource.DesiredState)
					return nil
				}); serr != nil {
				return nil, serr
			}

			// But we don't want to allow the user to update ALL fields.. so copy
			// over the fields that they are allowed to modify..
			resource.Name = patch.Name
			resource.Connector = patch.Connector
			resource.Annotations = patch.Annotations
			resource.Kafka = patch.Kafka
			resource.ServiceAccount = patch.ServiceAccount
			resource.SchemaRegistry = patch.SchemaRegistry

			if h.connectorsConfig.ConnectorEnableUnassignedConnectors {
				// check namespace id change, from unassigned to assigned and vice versa
				if operation == phase.AssignConnector && patch.NamespaceId != "" && resource.NamespaceId == "" {
					resource.NamespaceId = patch.NamespaceId
				}
				if operation == phase.UnassignConnector && patch.NamespaceId == "" && resource.NamespaceId != "" {
					resource.NamespaceId = patch.NamespaceId
				}
			}

			// If we didn't change anything, then just skip the update...
			if reflect.DeepEqual(originalResource, resource) {
				return originalResource, nil
			}

			// revalidate
			user := h.authZService.GetValidationUser(r.Context())
			validates := []handlers.Validate{
				handlers.Validation("name", &resource.Name, handlers.MinLen(1), handlers.MaxLen(100)),
				handlers.Validation("connector_type_id", &resource.ConnectorTypeId, handlers.MinLen(1), handlers.MaxLen(maxConnectorTypeIdLength)),
				handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
				handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.IsOneOf(dbapi.ValidDesiredStates...)),
				validatePatchAnnotations(resource.Annotations, originalResource.Annotations),
				validateConnector(h.connectorTypesService, &resource),
			}

			// Don't validate user's tenancy in admin api calls
			if strings.Compare(r.URL.Path, fmt.Sprintf("%s/%s", "/api/connector_mgmt/v1/admin/kafka_connectors", connectorId)) != 0 {
				validates = append(validates, handlers.Validation("namespace_id", &resource.NamespaceId, handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceUser(errors.ErrorBadRequest)))
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

			svcErr = moveSecretsToVault(p, ct, h.vaultService, false)
			if svcErr != nil {
				return nil, svcErr
			}

			// update connector phase before desired state
			if originalResource.Status.State != public.ConnectorState(dbapi.ConnectorStatusPhaseAssigning) {
				dbresource.Status.Phase = phase.ConnectorStartingPhase[operation]
				p.Status.Phase = dbresource.Status.Phase
				serr = h.connectorsService.SaveStatus(r.Context(), dbresource.Status)
				if serr != nil {
					return nil, serr
				}
			}
			// update modified connector including desired state
			serr = h.connectorsService.Update(r.Context(), p)
			if serr != nil {
				return nil, serr
			}

			newSecrets, err := getSecretRefs(p, ct)
			if err != nil {
				return nil, errors.GeneralError("could not get existing secrets: %v", err)
			}

			staleSecrets := StringListSubtract(originalSecrets, newSecrets...)
			if len(staleSecrets) > 0 {
				_ = db.AddPostCommitAction(r.Context(), func() {
					for _, s := range staleSecrets {
						err = h.vaultService.DeleteSecretString(s)
						if err != nil {
							logger.Logger.Errorf("failed to delete vault secret key '%s': %v", s, err)
						}
					}
				})
			}

			if err := stripSecretReferences(p, ct); err != nil {
				return nil, err
			}

			return presenters.PresentConnector(p)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h ConnectorsHandler) getOperation(resource public.Connector, patch public.ConnectorRequest) (phase.ConnectorOperation, *errors.ServiceError) {
	operation, ok := stateToOperationsMap[patch.DesiredState]
	if !ok {
		return operation, errors.BadRequest("Unsupported patch desired state %s", patch.DesiredState)
	}
	if patch.DesiredState == resource.DesiredState {
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

func ValidateConnectorOperation(ctx context.Context, namespaceService services.ConnectorNamespaceService, connector *dbapi.Connector,
	operation phase.ConnectorOperation, updatePhase ...func(*dbapi.Connector) *errors.ServiceError) (err *errors.ServiceError) {
	if connector.NamespaceId != nil {
		var namespace *dbapi.ConnectorNamespace
		if namespace, err = namespaceService.Get(ctx, *connector.NamespaceId); err == nil {
			_, err = phase.PerformConnectorOperation(namespace, connector, operation, updatePhase...)
		}
	} else {
		// create, update and delete are the only operations allowed for connectors without a namespace id
		switch operation {
		case phase.CreateConnector, phase.UpdateConnector, phase.DeleteConnector:
			// valid operations
		default:
			return errors.Validation("Invalid operation %s for connector with id %s in state %s",
				operation, connector.ID, connector.DesiredState)
		}
	}
	return err
}

func validateConnectorPatch(bytes []byte, ct *dbapi.ConnectorType) *errors.ServiceError {
	type Connector struct {
		ConnectorSpec api.JSON `json:"connector,omitempty"`
	}
	c := Connector{}
	err := json.Unmarshal(bytes, &c)
	if err != nil {
		return errors.BadRequest("invalid patch: %v", err)
	}

	if len(c.ConnectorSpec) > 0 {
		_, err = secrets.ModifySecrets(ct.JsonSchema, c.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() == ajson.Object {
				if len(node.Keys()) > 0 {
					return fmt.Errorf("attempting to change opaque connector secret")
				}
			}
			return nil
		})
		if err != nil {
			return errors.BadRequest("invalid patch: %v", err)
		}
	}
	return nil
}

func StringListSubtract(l []string, items ...string) (result []string) {
	m := make(map[string]struct{}, len(l))
	for _, x := range l {
		m[x] = struct{}{}
	}
	for _, item := range items {
		delete(m, item)
	}
	for item := range m {
		result = append(result, item)
	}
	return result
}

func convertToMergePatch(patchBytes []byte, resourceJson []byte) ([]byte, error) {
	patchObj, err := jsonpatch.DecodePatch(patchBytes)
	if err != nil {
		return nil, errors.BadRequest("invalid json patch: %v", err)
	}
	patched, err := patchObj.Apply(resourceJson)
	if err != nil {
		return nil, errors.GeneralError("failed to apply patch: %v", err)
	}
	// diff original and patched to create merge-patch for validation
	result, err := jsonpatch.CreateMergePatch(resourceJson, patched)
	if err != nil {
		return nil, errors.GeneralError("failed to apply patch: %v", err)
	}
	return result, nil
}

func PatchResource(resourceJson []byte, patchType string, patchBytes []byte, patched interface{}) *errors.ServiceError {
	// apply the patch...
	var err error
	switch patchType {
	case "application/merge-patch+json", "application/json":
		resourceJson, err = jsonpatch.MergePatch(resourceJson, patchBytes)
		if err != nil {
			return errors.BadRequest("invalid merge patch: %v", err)
		}
	default:
		return errors.BadRequest("bad Content-Type, must be one of: application/json-patch+json, application/merge-patch+json (or equivalently application/json).")
	}

	err = json.Unmarshal(resourceJson, patched)
	if err != nil {
		return errors.GeneralError("failed to decode patched resource: %v", err)
	}
	return nil
}

func getResourceJson(resource interface{}) ([]byte, *errors.ServiceError) {
	if reflect.ValueOf(resource).Kind() == reflect.Ptr {
		resource = reflect.ValueOf(resource).Elem() // deref pointer...
	}
	resourceJson, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.GeneralError("failed to encode to json")
	}
	return resourceJson, nil
}

func (h ConnectorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorId := mux.Vars(r)["connector_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.connectorsService.Get(r.Context(), connectorId)
			if err != nil {
				return nil, err
			}

			ct, serr := h.connectorTypesService.Get(resource.ConnectorTypeId)
			if serr != nil {
				// gracefully degrade by not showing the connector spec, and updating the status
				// to signal this error
				resource.ConnectorSpec = api.JSON("{}")
				resource.Status.Phase = "bad-connector-type"
			} else {
				if err := stripSecretReferences(&resource.Connector, ct); err != nil {
					return nil, err
				}
			}

			return presenters.PresentConnectorWithError(resource)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a connector
func (h ConnectorsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	connectorId := mux.Vars(r)["connector_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_id", &connectorId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			ctx := r.Context()
			return nil, HandleConnectorDelete(ctx, h.connectorsService, h.namespaceService, connectorId)
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func HandleConnectorDelete(ctx context.Context, connectorsService services.ConnectorsService,
	namespaceService services.ConnectorNamespaceService, connectorId string) *errors.ServiceError {

	c, err := connectorsService.Get(ctx, connectorId)
	if err != nil {
		return err
	}

	// validate delete operation if connector is assigned to a namespace
	if c.NamespaceId != nil {
		err = ValidateConnectorOperation(ctx, namespaceService, &c.Connector, phase.DeleteConnector,
			func(connector *dbapi.Connector) (err *errors.ServiceError) {
				err = connectorsService.SaveStatus(ctx, connector.Status)
				if err == nil {
					err = connectorsService.Update(ctx, connector)
				}
				return err
			})
	} else {
		// connector not in namespace
		c.Status.Phase = dbapi.ConnectorStatusPhaseDeleted
		c.DesiredState = dbapi.ConnectorDeleted
		err = connectorsService.SaveStatus(ctx, c.Status)
		if err == nil {
			err = connectorsService.Update(ctx, &c.Connector)
		}
	}
	return err
}

func (h ConnectorsHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := coreServices.NewListArguments(r.URL.Query())
			resources, paging, err := h.connectorsService.List(ctx, listArgs, "")
			if err != nil {
				return nil, err
			}

			resourceList := public.ConnectorList{
				Kind:  "ConnectorList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {

				ct, serr := h.connectorTypesService.Get(resource.ConnectorTypeId)
				if serr != nil {
					// gracefully degrade by not showing the connector spec, and updating the status
					// to signal this error
					resource.ConnectorSpec = api.JSON("{}")
					resource.Status.Phase = "bad-connector-type"
				} else {
					if err := stripSecretReferences(&resource.Connector, ct); err != nil {
						return nil, err
					}
				}

				converted, err := presenters.PresentConnectorWithError(resource)
				if err != nil {
					glog.Errorf("connector id='%s' presentation failed: %v", resource.ID, err)
					return nil, errors.GeneralError("internal error")
				}
				resourceList.Items = append(resourceList.Items, converted)

			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
