package handlers

import (
	"github.com/golang/glog"
	"net/http"

	"github.com/gorilla/mux"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/private/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

type connectorsHandler struct {
	connectorsService     services.ConnectorsService
	connectorTypesService services.ConnectorTypesService
	kafkaService          services.KafkaService
}

func NewConnectorsHandler(kafkaService services.KafkaService, connectorsService services.ConnectorsService, connectorTypesService services.ConnectorTypesService) *connectorsHandler {
	return &connectorsHandler{
		kafkaService:          kafkaService,
		connectorsService:     connectorsService,
		connectorTypesService: connectorTypesService,
	}
}

func (h connectorsHandler) Create(w http.ResponseWriter, r *http.Request) {

	var resource openapi.Connector
	resource.Metadata.Owner = auth.GetUsernameFromContext(r.Context())
	resource.Metadata.KafkaId = mux.Vars(r)["id"]
	tid := mux.Vars(r)["tid"]

	cfg := &handlerConfig{

		MarshalInto: &resource,
		Validate: []validate{
			validateAsyncEnabled(r, "creating connector"),
			validateLength(&resource.Metadata.Owner, "owner", &minRequiredFieldLength, nil),
			validateLength(&resource.Metadata.Name, "name", &minRequiredFieldLength, &maxKafkaNameLength),
			validateLength(&resource.Metadata.KafkaId, "kafka id", &minRequiredFieldLength, &maxKafkaNameLength),
			validateLength(&resource.ConnectorTypeId, "connector type id", &minRequiredFieldLength, &maxKafkaNameLength),

			//validateCloudProvider(&resource, h.config, "creating connector"),
			validateMultiAZEnabled(&resource.DeploymentLocation.MultiAz, "creating connector"),
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
			if err := h.connectorsService.Create(r.Context(), convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnector(convResource)
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func (h connectorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			kid := mux.Vars(r)["id"]
			cid := mux.Vars(r)["cid"]
			tid := mux.Vars(r)["tid"]
			resource, err := h.connectorsService.Get(r.Context(), kid, cid, tid)
			if err != nil {
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
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			kid := mux.Vars(r)["id"]
			cid := mux.Vars(r)["cid"]
			err := h.connectorsService.Delete(r.Context(), kid, cid)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (h connectorsHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			kid := mux.Vars(r)["id"]
			tid := mux.Vars(r)["tid"]
			listArgs := services.NewListArguments(r.URL.Query())
			resources, paging, err := h.connectorsService.List(ctx, kid, listArgs, tid)
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
