package handlers

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type serviceAccountsHandler struct {
	service services.KeycloakService
}

func NewServiceAccountHandler(service services.KeycloakService) *serviceAccountsHandler {
	return &serviceAccountsHandler{
		service: service,
	}
}

func (s serviceAccountsHandler) ListServiceAccounts(w http.ResponseWriter, r *http.Request) {
	cfg := &HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			Page, Size := s.handleParams(r.URL.Query())
			sa, err := s.service.ListServiceAcc(ctx, Page, Size)
			if err != nil {
				return nil, err
			}

			serviceAccountList := openapi.ServiceAccountList{
				Kind:  "ServiceAccountList",
				Items: []openapi.ServiceAccountListItem{},
			}

			for _, account := range sa {
				converted := presenters.PresentServiceAccountListItem(&account)
				serviceAccountList.Items = append(serviceAccountList.Items, converted)
			}

			return serviceAccountList, nil
		},
	}
	HandleList(w, r, cfg)
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
	var serviceAccountRequest openapi.ServiceAccountRequest
	cfg := &HandlerConfig{
		MarshalInto: &serviceAccountRequest,
		Validate: []Validate{
			ValidateLength(&serviceAccountRequest.Name, "name", &minRequiredFieldLength, &maxServiceAccountNameLength),
			ValidateMaxLength(&serviceAccountRequest.Description, "description", &maxServiceAccountDescLength),
			ValidateServiceAccountName(&serviceAccountRequest.Name, "name"),
			ValidateServiceAccountDesc(&serviceAccountRequest.Description, "description"),
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
	Handle(w, r, cfg, http.StatusAccepted)
}

func (s serviceAccountsHandler) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &HandlerConfig{
		Validate: []Validate{
			ValidateLength(&id, "id", &minRequiredFieldLength, &maxServiceAccountId),
			ValidateServiceAccountId(&id, "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := s.service.DeleteServiceAccount(ctx, id)
			return nil, err
		},
	}

	HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (s serviceAccountsHandler) ResetServiceAccountCredential(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &HandlerConfig{
		Validate: []Validate{
			ValidateLength(&id, "id", &minRequiredFieldLength, &maxServiceAccountId),
			ValidateServiceAccountId(&id, "id"),
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

	HandleGet(w, r, cfg)
}

func (s serviceAccountsHandler) GetServiceAccountById(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &HandlerConfig{
		Validate: []Validate{
			ValidateLength(&id, "id", &minRequiredFieldLength, &maxServiceAccountId),
			ValidateServiceAccountId(&id, "id"),
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

	HandleGet(w, r, cfg)
}
