package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

type kafkaHandler struct {
	service services.KafkaService
}

func NewKafkaHandler(service services.KafkaService) *kafkaHandler {
	return &kafkaHandler{
		service: service,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafkaRequest openapi.KafkaRequest
	kafkaRequest.Owner = auth.GetUsernameFromContext(r.Context())
	cfg := &handlerConfig{
		MarshalInto: &kafkaRequest,
		Validate: []validate{
			validateNotEmpty(&kafkaRequest.Owner, "owner"),
			validateEmpty(&kafkaRequest.Id, "id"),
			validateNotEmpty(&kafkaRequest.Region, "region"),
			validateNotEmpty(&kafkaRequest.CloudProvider, "cloud_provider"),
			validateNotEmpty(&kafkaRequest.Name, "name"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			convKafka := presenters.ConvertKafkaRequest(kafkaRequest)

			err := h.service.RegisterKafkaJob(convKafka)
			if err != nil {
				return nil, err
			}
			err = h.service.Create(convKafka)
			if err != nil {
				return nil, err
			}
			return presenters.PresentKafkaRequest(convKafka), nil
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func (h kafkaHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			kafkaRequest, err := h.service.Get(id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentKafkaRequest(kafkaRequest), nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

// Delete is the handler for deleting a kafka request
func (h kafkaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			err := h.service.Delete(id)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (h kafkaHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			kafkaRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			kafkaRequestList := openapi.KafkaRequestList{
				Kind:  "KafkaRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.KafkaRequest{},
			}

			for _, kafkaRequest := range kafkaRequests {
				converted := presenters.PresentKafkaRequest(kafkaRequest)
				kafkaRequestList.Items = append(kafkaRequestList.Items, converted)
			}

			return kafkaRequestList, nil
		},
		ErrorHandler: handleError,
	}

	handleList(w, r, cfg)
}
