package handlers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

// handlerConfig defines the common things each REST controller must do.
// The corresponding handle() func runs the basic handlerConfig.
// This is not meant to be an HTTP framework or anything larger than simple CRUD in handlers.
//
//   MarshalInto is a pointer to the object to hold the unmarshaled JSON.
//   Validate is a list of validation function that run in order, returning fast on the first error.
//   Action is the specific logic a handler must take (e.g, find an object, save an object)
//   ErrorHandler is the way errors are returned to the client
type handlerConfig struct {
	MarshalInto  interface{}
	Validate     []validate
	Action       httpAction
	ErrorHandler errorHandlerFunc
}

type validate func() *errors.ServiceError
type errorHandlerFunc func(ctx context.Context, w http.ResponseWriter, err *errors.ServiceError)
type httpAction func() (interface{}, *errors.ServiceError)

func handleError(ctx context.Context, w http.ResponseWriter, err *errors.ServiceError) {
	ulog := logger.NewUHCLogger(ctx)
	operationID := logger.GetOperationID(ctx)
	// If this is a 400 error, its the user's issue, log as info rather than error
	if err.HttpCode >= 400 && err.HttpCode <= 499 {
		ulog.Infof(err.Error())
	} else {
		ulog.Errorf(err.Error())
	}
	shared.WriteJSONResponse(w, err.HttpCode, err.AsOpenapiError(operationID))
}

func handle(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, httpStatus int) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = handleError
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handleError(r.Context(), w, errors.MalformedRequest("Unable to read request body: %s", err))
		return
	}

	err = json.Unmarshal(bytes, &cfg.MarshalInto)
	if err != nil {
		handleError(r.Context(), w, errors.MalformedRequest("Invalid request format: %s", err))
		return
	}

	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			cfg.ErrorHandler(r.Context(), w, err)
			return
		}
	}

	result, serviceErr := cfg.Action()

	switch {
	case serviceErr != nil:
		cfg.ErrorHandler(r.Context(), w, serviceErr)
	default:
		shared.WriteJSONResponse(w, httpStatus, result)
	}

}

func handleDelete(w http.ResponseWriter, r *http.Request, cfg *handlerConfig, httpStatus int) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = handleError
	}
	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			cfg.ErrorHandler(r.Context(), w, err)
			return
		}
	}

	result, serviceErr := cfg.Action()

	switch {
	case serviceErr != nil:
		cfg.ErrorHandler(r.Context(), w, serviceErr)
	default:
		shared.WriteJSONResponse(w, httpStatus, result)
	}

}

func handleGet(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = handleError
	}

	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			cfg.ErrorHandler(r.Context(), w, err)
			return
		}
	}

	result, serviceErr := cfg.Action()
	switch {
	case serviceErr == nil:
		shared.WriteJSONResponse(w, http.StatusOK, result)
	default:
		cfg.ErrorHandler(r.Context(), w, serviceErr)
	}
}

func handleList(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = handleError
	}

	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			cfg.ErrorHandler(r.Context(), w, err)
			return
		}
	}

	results, serviceError := cfg.Action()
	if serviceError != nil {
		cfg.ErrorHandler(r.Context(), w, serviceError)
		return
	}
	shared.WriteJSONResponse(w, http.StatusOK, results)
}
