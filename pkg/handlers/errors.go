package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

func NewErrorsHandler() *errorHandler {
	return &errorHandler{}
}

type errorHandler struct{}

var _ RestHandler = errorHandler{}

func (h errorHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			listArgs := services.NewListArguments(r.URL.Query())
			allErrors := errors.Errors()

			// Sort errors by code
			sort.SliceStable(allErrors, func(i, j int) bool {
				return allErrors[i].Code < allErrors[j].Code
			})

			list, total := determineListRange(allErrors, listArgs.Page, listArgs.Size)
			errorList := openapi.ErrorList{
				Kind:  "ErrorList",
				Page:  int32(listArgs.Page),
				Size:  int32(len(list)),
				Total: int32(total),
				Items: []openapi.Error{},
			}
			for _, e := range list {
				err := e.(errors.ServiceError)
				errorList.Items = append(errorList.Items, presenters.PresentError(&err, r.RequestURI))
			}

			return errorList, nil
		},
	}

	handleList(w, r, cfg)
}

func (h errorHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			value, err := strconv.Atoi(id)
			if err != nil {
				return nil, errors.NotFound("No error with id %s exists", id)
			}
			code := errors.ServiceErrorCode(value)
			exists, sErr := errors.Find(code)
			if !exists {
				return nil, errors.NotFound("No error with id %s exists", id)
			}
			return presenters.PresentError(sErr, r.RequestURI), nil
		},
	}

	handleGet(w, r, cfg)
}

func (h errorHandler) Create(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("create"))
}

func (h errorHandler) Patch(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("path"))
}

func (h errorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("delete"))
}
