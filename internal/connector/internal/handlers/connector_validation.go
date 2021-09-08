package handlers

import (
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/xeipuuv/gojsonschema"
)

func validateConnectorSpec(connectorTypesService services.ConnectorTypesService, resource *public.Connector, tid string) handlers.Validate {
	return func() *errors.ServiceError {

		// If a tid was defined on the URL verify that it matches the posted resource connector type
		if tid != "" && tid != resource.ConnectorTypeId {
			return errors.BadRequest("resource type id should be: %s", tid)
		}
		ct, err := connectorTypesService.Get(resource.ConnectorTypeId)
		if err != nil {
			return errors.BadRequest("invalid connector type id %v : %s", resource.ConnectorTypeId, err)
		}

		if !shared.Contains(ct.ChannelNames(), resource.Channel) {
			return errors.BadRequest("channel is not valid. Must be one of: %s", strings.Join(ct.ChannelNames(), ", "))
		}

		schemaDom, err := ct.JsonSchemaAsMap()
		if err != nil {
			return err
		}
		schemaLoader := gojsonschema.NewGoLoader(schemaDom)
		documentLoader := gojsonschema.NewGoLoader(resource.ConnectorSpec)

		return handlers.ValidateJsonSchema("connector type schema", schemaLoader, "connector spec", documentLoader)
	}
}
