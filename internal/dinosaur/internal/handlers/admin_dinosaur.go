package handlers

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/account"

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
	accountService account.AccountService
	providerConfig *config.ProviderConfig
}

func NewAdminDinosaurHandler(service services.DinosaurService, accountService account.AccountService, providerConfig *config.ProviderConfig) *adminDinosaurHandler {
	return &adminDinosaurHandler{
		service:        service,
		accountService: accountService,
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
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
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
				converted, err := presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
				if err != nil {
					return nil, err
				}
				dinosaurRequestList.Items = append(dinosaurRequestList.Items, *converted)
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
			ValidateDinosaurUpdateFields(&dinosaurUpdateReq.DinosaurVersion, &dinosaurUpdateReq.StrimziVersion, &dinosaurUpdateReq.DinosaurIbpVersion),
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
				if dinosaurRequest.DinosaurUpgrading {
					return nil, errors.New(errors.ErrorValidation, "Unable to update dinosaur version. Another upgrade is already in progress.")
				}
				dinosaurRequest.DesiredDinosaurVersion = dinosaurUpdateReq.DinosaurVersion
				updateRequired = true
			}
			if dinosaurRequest.DesiredStrimziVersion != dinosaurUpdateReq.StrimziVersion && dinosaurUpdateReq.StrimziVersion != "" {
				if dinosaurRequest.StrimziUpgrading {
					return nil, errors.New(errors.ErrorValidation, "Unable to update strimzi version. Another upgrade is already in progress.")
				}
				dinosaurRequest.DesiredStrimziVersion = dinosaurUpdateReq.StrimziVersion
				updateRequired = true
			}
			if dinosaurRequest.DesiredDinosaurIBPVersion != dinosaurUpdateReq.DinosaurIbpVersion && dinosaurUpdateReq.DinosaurIbpVersion != "" {
				if dinosaurRequest.DinosaurIBPUpgrading {
					return nil, errors.New(errors.ErrorValidation, "Unable to update ibp version. Another upgrade is already in progress.")
				}
				dinosaurRequest.DesiredDinosaurIBPVersion = dinosaurUpdateReq.DinosaurIbpVersion
				updateRequired = true
			}
			if updateRequired {
				err3 := h.service.VerifyAndUpdateDinosaurAdmin(ctx, dinosaurRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
