package shared

import (
	"context"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

// HandleError handles a service error by returning an appropriate HTTP response with error reason
func HandleError(ctx context.Context, w http.ResponseWriter, err *errors.ServiceError) {
	ulog := logger.NewUHCLogger(ctx)
	operationID := logger.GetOperationID(ctx)
	if err.HttpCode >= 400 && err.HttpCode <= 499 {
		ulog.Infof(err.Error())
	} else {
		ulog.Error(err)
	}

	WriteJSONResponse(w, err.HttpCode, err.AsOpenapiError(operationID))
}
