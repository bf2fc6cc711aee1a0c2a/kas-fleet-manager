package handlers

import (
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/xeipuuv/gojsonschema"
)

func validateConnectorRequest(connectorTypesService services.ConnectorTypesService, resource *public.ConnectorRequest, tid string) handlers.Validate {
	return connectorValidationFunction(connectorTypesService, &resource.ConnectorTypeId, &resource.Channel, &resource.Connector, tid)
}

func validateConnector(connectorTypesService services.ConnectorTypesService, resource *public.Connector, tid string) handlers.Validate {
	return connectorValidationFunction(connectorTypesService, &resource.ConnectorTypeId, &resource.Channel, &resource.Connector, tid)
}

func connectorValidationFunction(connectorTypesService services.ConnectorTypesService, connectorTypeId *string, channel *public.Channel, connectorConfiguration *map[string]interface{}, tid string) handlers.Validate {
	return func() *errors.ServiceError {

		// If a tid was defined on the URL verify that it matches the posted resource connector type
		if tid != "" && tid != *connectorTypeId {
			return errors.BadRequest("resource type id should be: %s", tid)
		}
		ct, err := connectorTypesService.Get(*connectorTypeId)
		if err != nil {
			return errors.BadRequest("YYY invalid connector type id %v : %s", connectorTypeId, err)
		}

		if !shared.Contains(ct.ChannelNames(), string(*channel)) {
			return errors.BadRequest("channel is not valid. Must be one of: %s", strings.Join(ct.ChannelNames(), ", "))
		}

		schemaDom, err := ct.JsonSchemaAsMap()
		if err != nil {
			return err
		}
		schemaLoader := gojsonschema.NewGoLoader(schemaDom)
		documentLoader := gojsonschema.NewGoLoader(connectorConfiguration)

		return handlers.ValidateJsonSchema("connector type schema", schemaLoader, "connector spec", documentLoader)
	}
}
