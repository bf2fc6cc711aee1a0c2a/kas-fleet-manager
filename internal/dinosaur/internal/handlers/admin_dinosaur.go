package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

type adminDinosaurHandler struct {
	service        services.DinosaurService
	providerConfig *config.ProviderConfig
}

func NewAdminDinosaurHandler(service services.DinosaurService, providerConfig *config.ProviderConfig) *adminDinosaurHandler {
	return &adminDinosaurHandler{
		service:        service,
		providerConfig: providerConfig,
	}
}

func (h adminDinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h adminDinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list dinosaur requests: %s", err.Error())
			}

			dinosaurRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			dinosaurRequestList := private.DinosaurList{
				Kind:  "DinosaurList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []private.Dinosaur{},
			}

			for _, dinosaurRequest := range dinosaurRequests {
				converted := presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest)
				dinosaurRequestList.Items = append(dinosaurRequestList.Items, converted)
			}

			return dinosaurRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h adminDinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting dinosaur requests"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			err := h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h adminDinosaurHandler) Update(w http.ResponseWriter, r *http.Request) {

	var dinosaurUpdateReq private.DinosaurUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurUpdateReq,
		Validate: []handlers.Validate{
			handlers.ValidateDinosaurUpdateFields(&dinosaurUpdateReq.DinosaurVersion, &dinosaurUpdateReq.StrimziVersion),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err2 := h.service.Get(ctx, id)
			if err2 != nil {
				return nil, err2
			}
			dinosaurStatus := dinosaurRequest.Status
			if !shared.Contains(constants.GetUpdateableStatuses(), dinosaurStatus) {
				return nil, errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update dinosaur in %s status. Supported statuses for update are: %v", dinosaurStatus, constants.GetUpdateableStatuses()))
			}
			updateRequired := false
			if dinosaurRequest.DesiredDinosaurVersion != dinosaurUpdateReq.DinosaurVersion && dinosaurUpdateReq.DinosaurVersion != "" {
				dinosaurRequest.DesiredDinosaurVersion = dinosaurUpdateReq.DinosaurVersion
				updateRequired = true
			}
			if dinosaurRequest.DesiredStrimziVersion != dinosaurUpdateReq.StrimziVersion && dinosaurUpdateReq.StrimziVersion != "" {
				dinosaurRequest.DesiredStrimziVersion = dinosaurUpdateReq.StrimziVersion
				updateRequired = true
			}
			if updateRequired {
				err3 := h.service.VerifyAndUpdateDinosaurAdmin(ctx, dinosaurRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
