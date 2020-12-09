package handlers

import (
	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"net/http"
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
			return sa, nil
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
			validateNotEmpty(&serviceAccountRequest.Name, "name"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convSA := presenters.ConvertServiceAccountRequest(serviceAccountRequest)
			serviceAccount, err := s.service.CreateServiceAccount(convSA, ctx)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccountRequest(serviceAccount), nil
		},
		ErrorHandler: handleError,
	}
	handle(w, r, cfg, http.StatusAccepted)
}

func (s serviceAccountsHandler) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	clientId := mux.Vars(r)["clientId"]
	cfg := &handlerConfig{
		Validate: []validate{
			validateNotEmpty(&clientId, "clientId"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			err := s.service.DeleteServiceAccount(clientId)
			return nil, err
		},
		ErrorHandler: handleError,
	}

	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (s serviceAccountsHandler) ResetServiceAccountCredential(w http.ResponseWriter, r *http.Request) {
	clientId := mux.Vars(r)["clientId"]
	cfg := &handlerConfig{
		Validate: []validate{
			validateNotEmpty(&clientId, "clientId"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			sa, err := s.service.ResetServiceAccountCredentials(clientId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentServiceAccountRequest(sa), nil
		},
		ErrorHandler: handleError,
	}

	handleGet(w, r, cfg)
}
