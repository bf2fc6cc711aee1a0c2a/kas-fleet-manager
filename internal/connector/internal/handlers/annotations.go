package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

// annotations are mapped to k8s labels, check that it's not used to set any reserved domain labels
var reservedDomains = []string{"kubernetes.io/", "k8s.io/", "openshift.io/"}

// user is not allowed to create or patch system generated annotations
var reservedAnnotations = []string{dbapi.ConnectorClusterOrgIdAnnotation, dbapi.ConnectorTypePricingTierAnnotation}

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
				return errors.BadRequest("cannot use reserved annotation %s from domain %s", k, d)
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
