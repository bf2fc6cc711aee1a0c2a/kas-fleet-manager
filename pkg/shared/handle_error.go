package shared

import (
	"context"
	"net/http"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/logger"
)

// HandleError handles a service error by returning an appropriate HTTP response with error reason
func HandleError(ctx context.Context, w http.ResponseWriter, code errors.ServiceErrorCode, reason string) {
	ulog := logger.NewUHCLogger(ctx)
	operationID := logger.GetOperationID(ctx)
	err := errors.New(code, reason)
	if err.HttpCode >= 400 && err.HttpCode <= 499 {
		ulog.Infof(err.Error())
	} else {
		ulog.Errorf(err.Error())
	}

	WriteJSONResponse(w, err.HttpCode, err.AsOpenapiError(operationID))
}
