package handlers

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"reflect"
)

var (
	maxConnectorIdLength = 32
)

type connectorsHandler struct {
	connectorsService     services.ConnectorsService
	connectorTypesService services.ConnectorTypesService
	kafkaService          services.KafkaService
	vaultService          services.VaultService
}

func NewConnectorsHandler(kafkaService services.KafkaService, connectorsService services.ConnectorsService, connectorTypesService services.ConnectorTypesService, vaultService services.VaultService) *connectorsHandler {
	return &connectorsHandler{
		kafkaService:          kafkaService,
		connectorsService:     connectorsService,
		connectorTypesService: connectorTypesService,
		vaultService:          vaultService,
	}
}

func (h connectorsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource openapi.Connector
	tid := mux.Vars(r)["tid"]
	cfg := &handlerConfig{

		MarshalInto: &resource,
		Validate: []validate{
			validateAsyncEnabled(r, "creating connector"),
			validation("name", &resource.Metadata.Name,
				withDefault("New Connector"), minLen(1), maxLen(100)),
			validation("kafka_id", &resource.Metadata.KafkaId, minLen(1), maxLen(maxKafkaNameLength)),
			validation("connector_type_id", &resource.ConnectorTypeId, minLen(1), maxLen(maxConnectorTypeIdLength)),
			validation("desired_state", &resource.DesiredState, withDefault("ready"), isOneOf("ready", "stopped")),
			validateConnectorSpec(h.connectorTypesService, &resource, tid),
		},

		Action: func() (interface{}, *errors.ServiceError) {

			// Get the Kafka to assert the user can access that kafka instance.
			_, err := h.kafkaService.Get(r.Context(), resource.Metadata.KafkaId)
			if err != nil {
				return nil, err
			}

			convResource, err := presenters.ConvertConnector(resource)
			if err != nil {
				return nil, err
			}

			claims, goerr := auth.GetClaimsFromContext(r.Context())
			if goerr != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}

			convResource.ID = api.NewID()
			convResource.Owner = auth.GetUsernameFromClaims(claims)
			convResource.OrganisationId = auth.GetOrgIdFromClaims(claims)

			err = moveSecretsToVault(convResource, h.connectorTypesService, h.vaultService)
			if err != nil {
				return nil, err
			}

			if svcErr := h.connectorsService.Create(r.Context(), convResource); svcErr != nil {
				return nil, svcErr
			}

			if err := stripSecretReferences(convResource, h.connectorTypesService); err != nil {
				return nil, err
			}

			return presenters.PresentConnector(convResource)
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func (h connectorsHandler) Patch(w http.ResponseWriter, r *http.Request) {

	connectorId := mux.Vars(r)["connector_id"]
	connectorTypeId := mux.Vars(r)["connector_type_id"]
	contentType := r.Header.Get("Content-Type")

	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_id", &connectorId, minLen(1), maxLen(maxConnectorIdLength)),
			validation("connector_type_id", &connectorTypeId, maxLen(maxConnectorTypeIdLength)),
			validation("Content-Type header", &contentType, isOneOf("application/json-patch+json", "application/merge-patch+json")),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			dbresource, serr := h.connectorsService.Get(r.Context(), connectorId, connectorTypeId)
			if serr != nil {
				return nil, serr
			}

			resource, serr := presenters.PresentConnector(dbresource)
			if serr != nil {
				return nil, serr
			}

			// Apply the patch..
			patchBytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, errors.BadRequest("failed to get patch bytes")
			}

			patch := openapi.Connector{}
			serr = PatchResource(resource, contentType, patchBytes, &patch)
			if serr != nil {
				return nil, serr
			}

			// But we don't want to allow the user to update ALL fields.. so copy
			// over the fields that they are allowed to modify..
			resource.Metadata.Name = patch.Metadata.Name
			resource.ConnectorTypeId = patch.ConnectorTypeId
			resource.ConnectorSpec = patch.ConnectorSpec
			resource.DesiredState = patch.DesiredState
			resource.Metadata.ResourceVersion = patch.Metadata.ResourceVersion

			// revalidate
			validates := []validate{
				validation("name", &resource.Metadata.Name, minLen(1), maxLen(100)),
				validation("connector_type_id", &resource.ConnectorTypeId, minLen(1), maxLen(maxKafkaNameLength)),
				validateConnectorSpec(h.connectorTypesService, &resource, connectorTypeId),
			}

			for _, v := range validates {
				err := v()
				if err != nil {
					return nil, err
				}
			}

			// If we didn't change anything, then just skip the update...
			originalResource, _ := presenters.PresentConnector(dbresource)
			if reflect.DeepEqual(originalResource, resource) {
				return originalResource, nil
			}

			p, svcErr := presenters.ConvertConnector(resource)
			if svcErr != nil {
				return nil, svcErr
			}

			serr = h.connectorsService.Update(r.Context(), p)
			if serr != nil {
				return nil, serr
			}

			dbresource.Status.Phase = api.ConnectorStatusPhaseProvisioning
			p.Status.Phase = api.ConnectorStatusPhaseProvisioning
			serr = h.connectorsService.SaveStatus(r.Context(), dbresource.Status)
			if serr != nil {
				return nil, serr
			}

			if err := stripSecretReferences(p, h.connectorTypesService); err != nil {
				return nil, err
			}

			return presenters.PresentConnector(p)
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func PatchResource(resource interface{}, patchType string, patchBytes []byte, patched interface{}) *errors.ServiceError {
	if reflect.ValueOf(resource).Kind() == reflect.Ptr {
		resource = reflect.ValueOf(resource).Elem() // deref pointer...
	}
	resourceJson, err := json.Marshal(resource)
	if err != nil {
		return errors.GeneralError("failed to encode to json")
	}

	// apply the patch...
	switch patchType {
	case "application/json-patch+json":
		var patchObj jsonpatch.Patch
		patchObj, err := jsonpatch.DecodePatch(patchBytes)
		if err != nil {
			return errors.BadRequest("invalid json patch: %v", err)
		}
		resourceJson, err = patchObj.Apply(resourceJson)
		if err != nil {
			return errors.GeneralError("failed to apply patch patch: %v", err)
		}
	case "application/merge-patch+json":
		resourceJson, err = jsonpatch.MergePatch(resourceJson, patchBytes)
		if err != nil {
			return errors.BadRequest("invalid json patch: %v", err)
		}
	default:
		return errors.BadRequest("bad Content-Type, must be one of: application/json-patch+json or application/merge-patch+json")
	}

	err = json.Unmarshal(resourceJson, patched)
	if err != nil {
		return errors.GeneralError("failed to decode patched resource: %v", err)
	}
	return nil
}

func (h connectorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorId := mux.Vars(r)["connector_id"]
	connectorTypeId := mux.Vars(r)["connector_type_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_id", &connectorId, minLen(1), maxLen(maxConnectorIdLength)),
			validation("connector_type_id", &connectorTypeId, maxLen(maxConnectorTypeIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.connectorsService.Get(r.Context(), connectorId, connectorTypeId)
			if err != nil {
				return nil, err
			}

			if err := stripSecretReferences(resource, h.connectorTypesService); err != nil {
				return nil, err
			}

			return presenters.PresentConnector(resource)
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

// Delete is the handler for deleting a kafka request
func (h connectorsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	connectorId := mux.Vars(r)["connector_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_id", &connectorId, minLen(1), maxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			c, err := h.connectorsService.Get(r.Context(), connectorId, "")
			if err != nil {
				return nil, err
			}
			if c.DesiredState != "deleted" {
				c.DesiredState = "deleted"
				err := h.connectorsService.Update(r.Context(), c)
				if err != nil {
					return nil, err
				}
			}

			return nil, nil
		},
		ErrorHandler: handleError,
	}
	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (h connectorsHandler) List(w http.ResponseWriter, r *http.Request) {
	kafkaId := r.URL.Query().Get("kafka_id")
	connectorTypeId := mux.Vars(r)["connector_type_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("kafka_id", &kafkaId, minLen(1), maxLen(maxKafkaNameLength)),
			validation("connector_type_id", &connectorTypeId, maxLen(maxConnectorTypeIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())
			resources, paging, err := h.connectorsService.List(ctx, kafkaId, listArgs, connectorTypeId)
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
				if err := stripSecretReferences(resource, h.connectorTypesService); err != nil {
					return nil, err
				}
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
