package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

func ValidateJsonSchema(schemaName string, schemaLoader gojsonschema.JSONLoader, documentName string, documentLoader gojsonschema.JSONLoader) *errors.ServiceError {
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", schemaName, err)
	}

	r, err := schema.Validate(documentLoader)
	if err != nil {
		return errors.BadRequest("invalid %s: %v", documentName, err)
	}
	if !r.Valid() {
		return errors.BadRequest("%s not conform to the %s. %d errors encountered.  1st error: %s",
			documentName, schemaName, len(r.Errors()), r.Errors()[0].String())
	}
	return nil
}
