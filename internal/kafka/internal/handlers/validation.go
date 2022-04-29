package handlers

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/golang-jwt/jwt/v4"
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

var ValidKafkaClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var MaxKafkaNameLength = 32

func ValidKafkaClusterName(value *string, field string) handlers.Validate {
	return func() *errors.ServiceError {
		if !ValidKafkaClusterNameRegexp.MatchString(*value) {
			return errors.MalformedKafkaClusterName("%s does not match %s", field, ValidKafkaClusterNameRegexp.String())
		}
		return nil
	}
}

// ValidateKafkaClusterNameIsUnique returns a validator that validates that the kafka cluster name is unique
func ValidateKafkaClusterNameIsUnique(name *string, kafkaService services.KafkaService, context context.Context) handlers.Validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := kafkaService.List(context, &coreServices.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateKafkaClusterName()
		}

		return nil
	}
}

func getCloudProviderAndRegion(
	ctx context.Context,
	kafkaService *services.KafkaService,
	kafkaRequest *public.KafkaRequestPayload,
	providerConfig *config.ProviderConfig) (string, string, *errors.ServiceError) {

	// Set Cloud Provider default if not received in the request
	supportedProviders := providerConfig.ProvidersConfig.SupportedProviders

	defaultProvider, _ := supportedProviders.GetDefault()
	providerName := arrays.FirstNonEmptyOrDefault(defaultProvider.Name, kafkaRequest.CloudProvider)
	// Validation for Cloud Provider
	provider, providerSupported := supportedProviders.GetByName(providerName)
	if !providerSupported {
		return "", "", errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", kafkaRequest.CloudProvider, supportedProviders)
	}

	// Validation for Cloud Region
	if kafkaRequest.Region != "" { // if region is empty, default region will be chosen, so no validation is needed
		regionSupported := provider.IsRegionSupported(kafkaRequest.Region)
		if !regionSupported {
			return "", "", errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", kafkaRequest.Region, kafkaRequest.CloudProvider, provider.Regions)
		}
	}

	claims, err := getClaims(ctx)
	if err != nil {
		return "", "", err
	}

	owner := auth.GetUsernameFromClaims(claims)
	organisationId := auth.GetOrgIdFromClaims(claims)

	// Validate Region/InstanceType
	instanceType, err := (*kafkaService).AssignInstanceType(owner, organisationId)
	if err != nil {
		return "", "", errors.NewWithCause(errors.ErrorGeneral, err, "error assigning instance type: %s", err.Error())
	}

	var region config.Region
	if kafkaRequest.Region == "" {
		region, _ = provider.GetDefaultRegion()
	} else {
		region, _ = provider.Regions.GetByName(kafkaRequest.Region)
	}

	if !region.IsInstanceTypeSupported(config.InstanceType(instanceType)) {
		return "", "", errors.InstanceTypeNotSupported("instance type '%s' not supported for region '%s'", instanceType.String(), region.Name)
	}

	return providerName, region.Name, nil
}

// ValidateCloudProvider returns a validator that validates provided provider and region
func ValidateCloudProvider(ctx context.Context, kafkaService *services.KafkaService, kafkaRequest *public.KafkaRequestPayload, providerConfig *config.ProviderConfig, action string) handlers.Validate {
	return func() *errors.ServiceError {
		_, _, err := getCloudProviderAndRegion(ctx, kafkaService, kafkaRequest, providerConfig)
		return err
	}
}

func getInstanceTypeAndSize(ctx context.Context, kafkaService *services.KafkaService, kafkaConfig *config.KafkaConfig, kafkaRequestPayload *public.KafkaRequestPayload) (string, string, *errors.ServiceError) {
	claims, err := getClaims(ctx)
	if err != nil {
		return "", "", err
	}

	owner := auth.GetUsernameFromClaims(claims)
	organisationId := auth.GetOrgIdFromClaims(claims)
	instanceType, err := (*kafkaService).AssignInstanceType(owner, organisationId)
	if err != nil {
		return "", "", err
	}
	if stringSet(&kafkaRequestPayload.Plan) {
		plan := config.Plan(kafkaRequestPayload.Plan)
		instTypeFromPlan, err := plan.GetInstanceType()
		if err != nil || instTypeFromPlan != string(instanceType) {
			return "", "", errors.New(errors.ErrorBadRequest, fmt.Sprintf("Unable to detect instance type in plan provided: '%s'", kafkaRequestPayload.Plan))
		}
		size, err := plan.GetSizeID()
		if err != nil {
			return "", "", errors.New(errors.ErrorBadRequest, fmt.Sprintf("Unable to detect instance size in plan provided: '%s'", kafkaRequestPayload.Plan))
		}
		_, err = kafkaConfig.GetKafkaInstanceSize(instTypeFromPlan, size)

		if err != nil {
			return "", "", errors.InstancePlanNotSupported("Unsupported plan provided: '%s'", kafkaRequestPayload.Plan)
		}
		return instanceType.String(), size, nil
	} else {
		rSize, err := kafkaConfig.GetFirstAvailableSize(instanceType.String())
		if err != nil {
			return "", "", errors.InstanceTypeNotSupported("Unsupported kafka instance type: '%s' provided", instanceType.String())
		}
		return instanceType.String(), rSize.Id, nil
	}
}

// ValidateKafkaPlan - validate the requested Kafka Plan
func ValidateKafkaPlan(ctx context.Context, kafkaService *services.KafkaService, kafkaConfig *config.KafkaConfig, kafkaRequestPayload *public.KafkaRequestPayload) handlers.Validate { // Validate plan
	return func() *errors.ServiceError {
		_, _, err := getInstanceTypeAndSize(ctx, kafkaService, kafkaConfig, kafkaRequestPayload)
		return err
	}
}

func ValidateKafkaUpdateFields(kafkaUpdateRequest *private.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		if !(stringSet(&kafkaUpdateRequest.StrimziVersion) ||
			stringSet(&kafkaUpdateRequest.KafkaVersion) ||
			stringSet(&kafkaUpdateRequest.KafkaIbpVersion) ||
			stringSet(&kafkaUpdateRequest.KafkaStorageSize)) {
			return errors.FieldValidationError("Failed to update Kafka Request. Expecting at least one of the following fields: strimzi_version, kafka_version, kafka_ibp_version or kafka_storage_size to be provided")
		}
		return nil
	}
}

func stringSet(value *string) bool {
	return value != nil && len(strings.Trim(*value, " ")) > 0
}

func ValidateKafkaUserFacingUpdateFields(ctx context.Context, authService authorization.Authorization, kafkaRequest *dbapi.KafkaRequest, kafkaUpdateReq *public.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		claims, claimsErr := auth.GetClaimsFromContext(ctx)
		if claimsErr != nil {
			return errors.NewWithCause(errors.ErrorUnauthenticated, claimsErr, "User not authenticated")
		}

		username := auth.GetUsernameFromClaims(claims)
		orgId := auth.GetOrgIdFromClaims(claims)
		isOrgAdmin := auth.GetIsOrgAdminFromClaims(claims)
		// only Kafka owner or organisation admin is allowed to perform the action
		isOwner := (isOrgAdmin || kafkaRequest.Owner == username) && kafkaRequest.OrganisationId == orgId
		if !isOwner {
			return errors.New(errors.ErrorUnauthorized, "User not authorized to perform this action")
		}

		if kafkaUpdateReq.Owner != nil {
			validationError := handlers.ValidateMinLength(kafkaUpdateReq.Owner, "owner", 1)()
			if validationError != nil {
				return validationError
			}

			userValid, err := authService.CheckUserValid(*kafkaUpdateReq.Owner, orgId)
			if err != nil {
				return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to update kafka request owner")
			}
			if !userValid {
				return errors.NewWithCause(errors.ErrorBadRequest, err, "User %s does not belong in your organization", *kafkaUpdateReq.Owner)
			}
		}

		return nil
	}
}

func getClaims(ctx context.Context) (jwt.MapClaims, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated")
	}
	return claims, nil
}

type ValidateKafkaClaimsOptions func(claims *jwt.MapClaims) *errors.ServiceError

func ValidateUsername() ValidateKafkaClaimsOptions {
	return func(claims *jwt.MapClaims) *errors.ServiceError {
		username := auth.GetUsernameFromClaims(*claims)
		if username == "" {
			return errors.New(errors.ErrorBadRequest, "can't find 'user' attribute in claims")
		}
		return nil
	}
}

func ValidateOrganisationId() ValidateKafkaClaimsOptions {
	return func(claims *jwt.MapClaims) *errors.ServiceError {
		orgId := auth.GetOrgIdFromClaims(*claims)
		if orgId == "" {
			return errors.New(errors.ErrorBadRequest, "can't find 'organisation id' attribute in claims")
		}
		return nil
	}
}

// ValidateKafkaClaims - Verifies that the context contains the required claims
func ValidateKafkaClaims(ctx context.Context, validations ...ValidateKafkaClaimsOptions) handlers.Validate {
	return func() *errors.ServiceError {
		claims, err := getClaims(ctx)

		for _, validation := range validations {
			if err = validation(&claims); err != nil {
				return err
			}
		}
		return err
	}
}

func ValidateKafkaStorageSize(kafkaRequest *dbapi.KafkaRequest, kafkaUpdateReq *private.KafkaUpdateRequest) handlers.Validate {
	return func() *errors.ServiceError {
		if stringSet(&kafkaUpdateReq.KafkaStorageSize) {
			currentSize, err := resource.ParseQuantity(kafkaRequest.KafkaStorageSize)
			if err != nil {
				return errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current storage size: '%s'", kafkaRequest.KafkaStorageSize)
			}
			requestedSize, err := resource.ParseQuantity(kafkaUpdateReq.KafkaStorageSize)
			if err != nil {
				return errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current requested size: '%s'", kafkaUpdateReq.KafkaStorageSize)
			}
			currSize, _ := currentSize.AsInt64()
			if requestedSize.CmpInt64(currSize) < 0 {
				return errors.FieldValidationError("Failed to update Kafka Request. Requested size: '%s' should be greater than current size: '%s'", kafkaUpdateReq.KafkaStorageSize, kafkaRequest.KafkaStorageSize)
			}
		}
		return nil
	}
}
