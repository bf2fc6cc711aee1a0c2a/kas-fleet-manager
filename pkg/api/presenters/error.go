package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentError(err *errors.ServiceError, url string) openapi.Error {
	return err.AsOpenapiError("", url)
}
