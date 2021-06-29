package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentError(err *errors.ServiceError, url string) compat.Error {
	return err.AsOpenapiError("", url)
}
