package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

type dinosaurHandler struct {
	service        services.DinosaurService
	providerConfig *config.ProviderConfig
}

func NewDinosaurHandler(service services.DinosaurService, providerConfig *config.ProviderConfig) *dinosaurHandler {
	return &dinosaurHandler{
		service:        service,
		providerConfig: providerConfig,
	}
}

func (h dinosaurHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dinosaurRequest public.DinosaurRequestPayload
	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating dinosaur requests"),
			handlers.ValidateLength(&dinosaurRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxDinosaurNameLength),
			ValidDinosaurClusterName(&dinosaurRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(&dinosaurRequest.Name, h.service, r.Context()),
			ValidateCloudProvider(&dinosaurRequest, h.providerConfig, "creating dinosaur requests"),
			handlers.ValidateMultiAZEnabled(&dinosaurRequest.MultiAz, "creating dinosaur requests"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convDinosaur := presenters.ConvertDinosaurRequest(dinosaurRequest)

			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convDinosaur.Owner = auth.GetUsernameFromClaims(claims)
			convDinosaur.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convDinosaur.OwnerAccountId = auth.GetAccountIdFromClaims(claims)

			svcErr := h.service.RegisterDinosaurJob(convDinosaur)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentDinosaurRequest(convDinosaur), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h dinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequest(dinosaurRequest), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a dinosaur request
func (h dinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h dinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
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

			dinosaurRequestList := public.DinosaurRequestList{
				Kind:  "DinosaurRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []public.DinosaurRequest{},
			}

			for _, dinosaurRequest := range dinosaurRequests {
				converted := presenters.PresentDinosaurRequest(dinosaurRequest)
				dinosaurRequestList.Items = append(dinosaurRequestList.Items, converted)
			}

			return dinosaurRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Update is the handler for updating a dinosaur request
func (h dinosaurHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dinosaurUpdateReq public.DinosaurUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurUpdateReq,
		Validate: []handlers.Validate{
			handlers.ValidateMinLength(&dinosaurUpdateReq.Owner, "owner", 1),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err2 := h.service.Get(ctx, id)
			if err2 != nil {
				return nil, err2
			}
			if dinosaurRequest.Owner != dinosaurUpdateReq.Owner {
				dinosaurRequest.Owner = dinosaurUpdateReq.Owner
				err3 := h.service.VerifyAndUpdateDinosaur(ctx, dinosaurRequest)
				if err3 != nil {
					return nil, err3
				}
			}
			return presenters.PresentDinosaurRequest(dinosaurRequest), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
