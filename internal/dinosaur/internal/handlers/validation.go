package handlers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/authorization"
)

var ValidDinosaurClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var MaxDinosaurNameLength = 32

func ValidDinosaurClusterName(value *string, field string) handlers.Validate {
	return func() *errors.ServiceError {
		if !ValidDinosaurClusterNameRegexp.MatchString(*value) {
			return errors.MalformedDinosaurClusterName("%s does not match %s", field, ValidDinosaurClusterNameRegexp.String())
		}
		return nil
	}
}

// ValidateDinosaurClusterNameIsUnique returns a validator that validates that the dinosaur cluster name is unique
func ValidateDinosaurClusterNameIsUnique(name *string, dinosaurService services.DinosaurService, context context.Context) handlers.Validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := dinosaurService.List(context, &coreServices.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateDinosaurClusterName()
		}

		return nil
	}
}

// ValidateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func ValidateCloudProvider(dinosaurService *services.DinosaurService, dinosaurRequest *dbapi.DinosaurRequest, providerConfig *config.ProviderConfig, action string) handlers.Validate {
	return func() *errors.ServiceError {
		// Set Cloud Provider default if not received in the request
		supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
		if dinosaurRequest.CloudProvider == "" {
			defaultProvider, _ := supportedProviders.GetDefault()
			dinosaurRequest.CloudProvider = defaultProvider.Name
		}

		// Validation for Cloud Provider
		provider, providerSupported := supportedProviders.GetByName(dinosaurRequest.CloudProvider)
		if !providerSupported {
			return errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", dinosaurRequest.CloudProvider, supportedProviders)
		}

		// Set Cloud Region default if not received in the request
		if dinosaurRequest.Region == "" {
			defaultRegion, _ := provider.GetDefaultRegion()
			dinosaurRequest.Region = defaultRegion.Name
		}

		// Validation for Cloud Region
		regionSupported := provider.IsRegionSupported(dinosaurRequest.Region)
		if !regionSupported {
			return errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", dinosaurRequest.Region, dinosaurRequest.CloudProvider, provider.Regions)
		}

		// Validate Region/InstanceType
		instanceType, err := (*dinosaurService).DetectInstanceType(dinosaurRequest)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error detecting instance type: %s", err.Error())
		}

		region, _ := provider.Regions.GetByName(dinosaurRequest.Region)
		if !region.IsInstanceTypeSupported(config.InstanceType(instanceType)) {
			return errors.InstanceTypeNotSupported("instance type '%s' not supported for region '%s'", instanceType.String(), region.Name)
		}
		return nil
	}
}

func ValidateDinosaurUpdateFields(strimziVersion *string, dinosaurVersion *string, DinosaurIbpVersion *string) handlers.Validate {
	return func() *errors.ServiceError {
		if stringNotSet(strimziVersion) && stringNotSet(dinosaurVersion) && stringNotSet(DinosaurIbpVersion) {
			return errors.FieldValidationError("Failed to update Dinosaur Request. Expecting at least one of the following fields: strimzi_version, dinosaur_version or dinosaur_ibp_version to be provided")
		}
		return nil
	}
}

func stringNotSet(value *string) bool {
	return value == nil || len(*value) < 1
}

func ValidateDinosaurUserFacingUpdateFields(ctx context.Context, authService authorization.Authorization, dinosaurRequest *dbapi.DinosaurRequest, dinosaurUpdateReq *public.DinosaurUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		claims, claimsErr := auth.GetClaimsFromContext(ctx)
		if claimsErr != nil {
			return errors.NewWithCause(errors.ErrorUnauthenticated, claimsErr, "User not authenticated")
		}

		username := auth.GetUsernameFromClaims(claims)
		orgId := auth.GetOrgIdFromClaims(claims)
		isOrgAdmin := auth.GetIsOrgAdminFromClaims(claims)
		// only Dinosaur owner or organisation admin is allowed to perform the action
		isOwner := (isOrgAdmin || dinosaurRequest.Owner == username) && dinosaurRequest.OrganisationId == orgId
		if !isOwner {
			return errors.New(errors.ErrorUnauthorized, "User not authorized to perform this action")
		}

		if dinosaurUpdateReq.Owner != nil {
			validationError := handlers.ValidateMinLength(dinosaurUpdateReq.Owner, "owner", 1)()
			if validationError != nil {
				return validationError
			}

			userValid, err := authService.CheckUserValid(*dinosaurUpdateReq.Owner, orgId)
			if err != nil {
				return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to update dinosaur request owner")
			}
			if !userValid {
				return errors.NewWithCause(errors.ErrorBadRequest, err, "User %s does not belong in your organization", *dinosaurUpdateReq.Owner)
			}
		}

		return nil
	}
}

func ValidateDinosaurClaims(ctx context.Context, dinosaurRequestPayload *public.DinosaurRequestPayload, dinosaurRequest *dbapi.DinosaurRequest) handlers.Validate {
	return func() *errors.ServiceError {
		dinosaurRequest = presenters.ConvertDinosaurRequest(*dinosaurRequestPayload, dinosaurRequest)
		claims, err := auth.GetClaimsFromContext(ctx)
		if err != nil {
			return errors.Unauthenticated("user not authenticated")
		}
		(*dinosaurRequest).Owner = auth.GetUsernameFromClaims(claims)
		(*dinosaurRequest).OrganisationId = auth.GetOrgIdFromClaims(claims)
		(*dinosaurRequest).OwnerAccountId = auth.GetAccountIdFromClaims(claims)

		return nil
	}
}
