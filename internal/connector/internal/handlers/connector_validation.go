package handlers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"k8s.io/apimachinery/pkg/util/validation"
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

// annotations are mapped to k8s labels, check that it's not used to set any reserved domain labels
var reservedDomains = []string{"kubernetes\\.io/", "k8s\\.io/", "openshift\\.io/"}

// user is not allowed to create or patch system generated annotations
var reservedAnnotations = []string{dbapi.ConnectorClusterOrgIdAnnotation}

// validateCreateAnnotations returns an error if user attempts to override a system annotation when creating a resource
func validateCreateAnnotations(annotations map[string]string) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateAnnotations(annotations); err != nil {
			return err
		}
		for _, k := range reservedAnnotations {
			if _, found := annotations[k]; found {
				return errors.BadRequest("cannot override reserved annotation %s", k)
			}
		}
		return nil
	}
}

// validateAnnotations returns an error for invalid k8s annotations, or reserved domains
func validateAnnotations(annotations map[string]string) *errors.ServiceError {
	for k, v := range annotations {
		errs := validation.IsQualifiedName(k)
		if len(errs) != 0 {
			return errors.BadRequest("invalid annotation key %s: %s", k, strings.Join(errs, "; "))
		}
		errs = validation.IsValidLabelValue(v)
		if len(errs) != 0 {
			return errors.BadRequest("invalid annotation value %s: %s", v, strings.Join(errs, "; "))
		}
		for _, d := range reservedDomains {
			if strings.Contains(k, d) {
				return errors.BadRequest("cannot use reserved annotation domain %s", k)
			}
		}
	}
	return nil
}

// validatePatchAnnotations returns an error if user attempts to add/delete/override a system annotation when patching a resource
func validatePatchAnnotations(patched, existing map[string]string) handlers.Validate {
	return func() *errors.ServiceError {
		if err := validateAnnotations(patched); err != nil {
			return err
		}
		for _, k := range reservedAnnotations {
			v, exists := existing[k]
			v1, copied := patched[k]
			if exists != copied || v != v1 {
				return errors.BadRequest("cannot override reserved annotation %s", k)
			}
		}
		return nil
	}
}
