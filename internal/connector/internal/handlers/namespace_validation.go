package handlers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

func validateNamespaceID(namespaceService services.ConnectorNamespaceService, ctx context.Context) handlers.ValidateOption {

	return func(name string, value *string) *errors.ServiceError {
		if value != nil && *value != "" {
			if _, err := namespaceService.Get(ctx, *value); err != nil {
				return errors.BadRequest("%s is not valid: %s", name, err)
			}
		}
		return nil
	}
}
