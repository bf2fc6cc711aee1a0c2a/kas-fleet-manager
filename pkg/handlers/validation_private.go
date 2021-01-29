package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/xeipuuv/gojsonschema"
)

func validateConnectorSpec(connectorTypesService services.ConnectorTypesService, resource *openapi.Connector, tid string) validate {
	return func() *errors.ServiceError {

		// If a a tid was defined on the URL verify that it matches the posted resource connector type
		if tid != "" && tid != resource.ConnectorTypeId {
			return errors.BadRequest("resource type id should be: %s", tid)
		}
		ct, err := connectorTypesService.Get(resource.ConnectorTypeId)
		if err != nil {
			return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
		}
		schemaLoader := gojsonschema.NewGoLoader(ct.JsonSchema)
		documentLoader := gojsonschema.NewGoLoader(resource.ConnectorSpec)

		return validateJsonSchema("connector type schema", schemaLoader, "connector spec", documentLoader)
	}
}
