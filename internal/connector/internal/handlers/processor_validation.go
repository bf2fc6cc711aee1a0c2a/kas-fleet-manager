package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/xeipuuv/gojsonschema"
)

import _ "embed"

//go:embed processor_schema.json
var schema string

func validateProcessorRequest(request *public.ProcessorRequest) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateErrorHandler(&request.ErrorHandler); err != nil {
			return err
		}
		if request.Definition == nil {
			return errors.BadRequest("Processor definition is missing.")
		}
		return validate(request.Definition)
	}
}

func validateProcessor(resource *public.Processor) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateErrorHandler(&resource.ErrorHandler); err != nil {
			return err
		}
		if resource.Definition == nil {
			return errors.BadRequest("Processor definition is missing.")
		}
		return validate(resource.Definition)
	}
}

func validateErrorHandler(eh *public.ErrorHandler) *errors.ServiceError {
	if eh.Log == nil && eh.Stop == nil && eh.DeadLetterQueue == (public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}) {
		eh.Stop = map[string]interface{}{}
	}
	definitions := 0
	if eh.Log != nil {
		definitions++
	}
	if eh.Stop != nil {
		definitions++
	}
	if eh.DeadLetterQueue != (public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}) {
		definitions++
	}
	if definitions > 1 {
		return errors.Validation("Only one ErrorHandler can be defined.")
	}
	return nil
}

func validate(definition map[string]interface{}) *errors.ServiceError {
	documentLoader := gojsonschema.NewGoLoader(definition)
	schemaLoader := gojsonschema.NewStringLoader(schema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.BadRequest("Invalid: %v", err)
	}

	r, err := schema.Validate(documentLoader)
	if err != nil {
		return errors.BadRequest("Invalid: %v", err)
	}
	if !r.Valid() {
		message := "Processor definition contains errors:\n"
		for i, e := range r.Errors() {
			message = message + fmt.Sprintf("(%d) %v", i+1, e)

		}
		return errors.BadRequest(message)
	}
	return nil
}
