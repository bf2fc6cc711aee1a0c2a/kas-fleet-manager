package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type serviceAccountsHandler struct {
	service services.KeycloakService
}

func NewServiceAccountHandler(service services.DinosaurKeycloakService) *serviceAccountsHandler {
	return &serviceAccountsHandler{
		service: service,
	}
}

func (s serviceAccountsHandler) ListServiceAccounts(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			Page, Size := s.handleParams(r.URL.Query())
			sa, err := s.service.ListServiceAcc(ctx, Page, Size)
			if err != nil {
				return nil, err
			}

			serviceAccountList := public.ServiceAccountList{
				Kind:  "ServiceAccountList",
				Items: []public.ServiceAccountListItem{},
			}

			for _, account := range sa {
				converted := presenters.PresentServiceAccountListItem(&account)
				serviceAccountList.Items = append(serviceAccountList.Items, converted)
			}

			return serviceAccountList, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (s serviceAccountsHandler) handleParams(params url.Values) (int, int) {
	Page := 0
	Size := 0
	if v := params.Get("page"); v != "" {
		Page, _ = strconv.Atoi(v)
	}
	if v := params.Get("size"); v != "" {
		Size, _ = strconv.Atoi(v)
	}
	return Page, Size
}

func (s serviceAccountsHandler) CreateServiceAccount(w http.ResponseWriter, r *http.Request) {
	var serviceAccountRequest public.ServiceAccountRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &serviceAccountRequest,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&serviceAccountRequest.Name, "name", &handlers.MinRequiredFieldLength, &handlers.MaxServiceAccountNameLength),
			handlers.ValidateMaxLength(&serviceAccountRequest.Description, "description", &handlers.MaxServiceAccountDescLength),
			handlers.ValidateServiceAccountName(&serviceAccountRequest.Name, "name"),
			handlers.ValidateServiceAccountDesc(&serviceAccountRequest.Description, "description"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convSA := presenters.ConvertServiceAccountRequest(serviceAccountRequest)
			serviceAccount, err := s.service.CreateServiceAccount(convSA, ctx)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccount(serviceAccount), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (s serviceAccountsHandler) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, &handlers.MaxServiceAccountId),
			handlers.ValidateServiceAccountId(&id, "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := s.service.DeleteServiceAccount(ctx, id)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (s serviceAccountsHandler) ResetServiceAccountCredential(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, &handlers.MaxServiceAccountId),
			handlers.ValidateServiceAccountId(&id, "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sa, err := s.service.ResetServiceAccountCredentials(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccount(sa), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (s serviceAccountsHandler) GetServiceAccountById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, &handlers.MaxServiceAccountId),
			handlers.ValidateServiceAccountId(&id, "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sa, err := s.service.GetServiceAccountById(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccount(sa), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
