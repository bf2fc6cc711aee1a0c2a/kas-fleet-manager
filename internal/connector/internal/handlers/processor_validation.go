package handlers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/xeipuuv/gojsonschema"
	"strings"
)

func validateProcessorRequest(processorTypesService services.ProcessorTypesService, request *public.ProcessorRequest) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateProcessorErrorHandler(&request.ErrorHandler); err != nil {
			return err
		}
		if request.Definition == nil {
			return errors.BadRequest("Processor definition is missing.")
		}
		return validationProcessorDefinition(processorTypesService, &request.ProcessorTypeId, &request.Channel, &request.Definition)
	}
}

func validateProcessor(processorTypesService services.ProcessorTypesService, resource *public.Processor) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateProcessorErrorHandler(&resource.ErrorHandler); err != nil {
			return err
		}
		if resource.Definition == nil {
			return errors.BadRequest("Processor definition is missing.")
		}
		return validationProcessorDefinition(processorTypesService, &resource.ProcessorTypeId, &resource.Channel, &resource.Definition)
	}
}

func validateProcessorErrorHandler(eh *public.ErrorHandler) *errors.ServiceError {
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

func validationProcessorDefinition(processorTypesService services.ProcessorTypesService, processorTypeId *string, channel *public.Channel, definition *map[string]interface{}) *errors.ServiceError {
	pt, err := processorTypesService.Get(*processorTypeId)
	if err != nil {
		return errors.BadRequest("invalid processor type id %v : %s", processorTypeId, err)
	}
	if !arrays.Contains(pt.ChannelNames(), string(*channel)) {
		return errors.BadRequest("channel is not valid. Must be one of: %s", strings.Join(pt.ChannelNames(), ", "))
	}

	schemaDom, err := pt.JsonSchemaAsMap()
	if err != nil {
		return err
	}
	schemaLoader := gojsonschema.NewGoLoader(schemaDom)
	documentLoader := gojsonschema.NewGoLoader(definition)
	schema, serr := gojsonschema.NewSchema(schemaLoader)
	if serr != nil {
		return errors.BadRequest("invalid ProcessorType schema: %v", err)
	}
	result, serr := schema.Validate(documentLoader)
	if serr != nil {
		return errors.BadRequest("invalid ProcessorType definition: %v", err)
	}
	if !result.Valid() {
		message := "ProcessorType definition does not conform to the ProcessorType schema. %d errors encountered. Errors: %s"
		for i, e := range result.Errors() {
			message = message + fmt.Sprintf("(%d) %v", i+1, e)

		}
		return errors.BadRequest(message, len(result.Errors()))
	}
	return nil
}
