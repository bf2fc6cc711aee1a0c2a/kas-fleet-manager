package config

import (
	"reflect"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/go-playground/validator/v10"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

// use a single instance of go-playground/validator Validate, it
// caches struct info
var validate *validator.Validate

func init() {
	validate = validator.New()

	err := validate.RegisterValidation("ams_product_validator", amsProductValidator)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("ams_resource_validator", amsResourceValidator)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("ams_billing_models_validator", amsBillingModelsValidator)
	if err != nil {
		panic(err)
	}
}

// amsProductValidator is a github.com/go-playground/validator/v10 custom
// validator.
// It validates that the provided field is one of the
// AMS Products that it validates. It returns true if complies with the
// validation criteria.
// If the provided value is not a string false is returned.
func amsProductValidator(fl validator.FieldLevel) bool {
	val, kind, _ := fl.ExtractType(fl.Field())
	if kind != reflect.String {
		return false
	}

	strVal := val.String()
	switch strVal {
	case string(ocm.RHOSAKProduct):
		return true
	case string(ocm.RHOSAKTrialProduct):
		return true
	case string(ocm.RHOSAKEvalProduct):
		return true
	case string(ocm.RHOSAKCCProduct):
		return true
	default:
		return false
	}
}

var _ validator.Func = amsProductValidator

// amsResourceValidator is a github.com/go-playground/validator/v10 custom
// validator.
// It validates that the provided field is one of the
// AMS Resources that it validates. It returns true if complies with the
// validation criteria.
// If the provided value is not a string false is returned.
func amsResourceValidator(fl validator.FieldLevel) bool {
	val, kind, _ := fl.ExtractType(fl.Field())
	if kind != reflect.String {
		return false
	}

	strVal := val.String()
	switch strVal {
	case string(ocm.RHOSAKResourceName):
		return true
	default:
		return false
	}
}

var _ validator.Func = amsResourceValidator

// amsBillingModelsValidator is a github.com/go-playground/validator/v10 custom
// validator.
// It validates that the provided field is one of the
// AMS Resources that it validates. It returns true if complies with the
// validation criteria.
// It accepts a string or a []string. If the provided type is not one of
// those values false is returned.
func amsBillingModelsValidator(fl validator.FieldLevel) bool {
	val, kind, _ := fl.ExtractType(fl.Field())

	var elems []string
	switch kind {
	case reflect.String:
		elems = append(elems, val.String())
	case reflect.Slice:
		slice, ok := val.Interface().([]string)
		if !ok {
			return false
		}
		elems = slice
	default:
		return false
	}

	for _, strVal := range elems {
		switch strVal {
		// TODO "marketplace" is a legacy AMS billing model value. Make sure
		// it is removed when we start sending
		// its equivalent "marketplace-rhm instead"
		case string(amsv1.BillingModelMarketplace):
			continue
		case string(amsv1.BillingModelMarketplaceAWS):
			continue
		case string(amsv1.BillingModelMarketplaceRHM):
			continue
		case string(amsv1.BillingModelStandard):
			continue
		default:
			return false
		}
	}

	return true
}

var _ validator.Func = amsBillingModelsValidator
