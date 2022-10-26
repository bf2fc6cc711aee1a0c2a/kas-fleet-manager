package handlers

import (
	"context"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/xeipuuv/gojsonschema"
)

func validateConnectorClusterId(ctx context.Context, clusterService services.ConnectorClusterService) handlers.ValidateOption {
	return func(field string, value *string) *errors.ServiceError {
		if _, err := clusterService.Get(ctx, *value); err != nil {
			return err
		}
		return nil
	}
}

func validateConnectorRequest(connectorTypesService services.ConnectorTypesService, resource *public.ConnectorRequest) handlers.Validate {
	return connectorValidationFunction(connectorTypesService, &resource.ConnectorTypeId, &resource.Channel, &resource.Connector)
}

func validateConnector(connectorTypesService services.ConnectorTypesService, resource *public.Connector) handlers.Validate {
	return connectorValidationFunction(connectorTypesService, &resource.ConnectorTypeId, &resource.Channel, &resource.Connector)
}

func connectorValidationFunction(connectorTypesService services.ConnectorTypesService, connectorTypeId *string, channel *public.Channel, connectorConfiguration *map[string]interface{}) handlers.Validate {
	return func() *errors.ServiceError {
		ct, err := connectorTypesService.Get(*connectorTypeId)
		if err != nil {
			return errors.BadRequest("YYY invalid connector type id %v : %s", connectorTypeId, err)
		}

		if !arrays.Contains(ct.ChannelNames(), string(*channel)) {
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
