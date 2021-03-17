package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
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

func (h connectorsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var patch openapi.Connector

	connectorId := mux.Vars(r)["connector_id"]
	connectorTypeId := mux.Vars(r)["connector_type_id"]

	cfg := &handlerConfig{
		MarshalInto: &patch,
		Validate: []validate{
			validation("connector_id", &connectorId, minLen(1), maxLen(maxConnectorIdLength)),
			validation("connector_type_id", &connectorTypeId, maxLen(maxConnectorTypeIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			dbresource, err := h.connectorsService.Get(r.Context(), connectorId, connectorTypeId)
			if err != nil {
				return nil, err
			}

			resource, err := presenters.PresentConnector(dbresource)
			if err != nil {
				return nil, err
			}

			// Apply the patch...
			if patch.Metadata.Name != "" {
				resource.Metadata.Name = patch.Metadata.Name
			}
			if patch.ConnectorTypeId != "" {
				resource.ConnectorTypeId = patch.ConnectorTypeId
			}
			if patch.ConnectorSpec != nil {
				resource.ConnectorSpec = patch.ConnectorSpec
			}
			if patch.Metadata.ResourceVersion != 0 {
				resource.Metadata.ResourceVersion = patch.Metadata.ResourceVersion
			}

			// revalidate changed fields...
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

			p, svcErr := presenters.ConvertConnector(resource)
			if svcErr != nil {
				return nil, svcErr
			}

			err = h.connectorsService.Update(r.Context(), p)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnector(p)
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
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
			err := h.connectorsService.Delete(r.Context(), connectorId)
			return nil, err
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
