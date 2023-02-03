package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

func validateProcessorRequest(request *public.ProcessorRequest) handlers.Validate {
	//TODO [manstis] No validation of Processors at the moment
	return func() *errors.ServiceError {
		return nil
	}
}

func validateProcessor(resource *public.Processor) handlers.Validate {
	//TODO [manstis] No validation of Processors at the moment
	return func() *errors.ServiceError {
		return nil
	}
}
