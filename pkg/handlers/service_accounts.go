package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
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
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sa, err := s.service.ListServiceAcc(ctx)
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
		ErrorHandler: handleError,
	}
	handleList(w, r, cfg)
}

func (s serviceAccountsHandler) CreateServiceAccount(w http.ResponseWriter, r *http.Request) {
	var serviceAccountRequest openapi.ServiceAccountRequest
	cfg := &handlerConfig{
		MarshalInto: &serviceAccountRequest,
		Validate: []validate{
			validateLength(&serviceAccountRequest.Name, "name", &minRequiredFieldLength, nil),
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
		ErrorHandler: handleError,
	}
	handle(w, r, cfg, http.StatusAccepted)
}

func (s serviceAccountsHandler) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := s.service.DeleteServiceAccount(ctx, id)
			return nil, err
		},
		ErrorHandler: handleError,
	}

	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (s serviceAccountsHandler) ResetServiceAccountCredential(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validateLength(&id, "id", &minRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sa, err := s.service.ResetServiceAccountCredentials(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccount(sa), nil
		},
		ErrorHandler: handleError,
	}

	handleGet(w, r, cfg)
}
