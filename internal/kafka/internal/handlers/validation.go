package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"

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

// ValidateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func ValidateCloudProvider(kafkaService *services.KafkaService, kafkaRequest *dbapi.KafkaRequest, providerConfig *config.ProviderConfig, action string) handlers.Validate {
	return func() *errors.ServiceError {
		// Set Cloud Provider default if not received in the request
		supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
		if kafkaRequest.CloudProvider == "" {
			defaultProvider, _ := supportedProviders.GetDefault()
			kafkaRequest.CloudProvider = defaultProvider.Name
		}

		// Validation for Cloud Provider
		provider, providerSupported := supportedProviders.GetByName(kafkaRequest.CloudProvider)
		if !providerSupported {
			return errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", kafkaRequest.CloudProvider, supportedProviders)
		}

		// Set Cloud Region default if not received in the request
		if kafkaRequest.Region == "" {
			defaultRegion, _ := provider.GetDefaultRegion()
			kafkaRequest.Region = defaultRegion.Name
		}

		// Validation for Cloud Region
		regionSupported := provider.IsRegionSupported(kafkaRequest.Region)
		if !regionSupported {
			return errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", kafkaRequest.Region, kafkaRequest.CloudProvider, provider.Regions)
		}

		// Validate Region/InstanceType
		instanceType, err := (*kafkaService).AssignInstanceType(kafkaRequest)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error assigning instance type: %s", err.Error())
		}

		region, _ := provider.Regions.GetByName(kafkaRequest.Region)
		if !region.IsInstanceTypeSupported(config.InstanceType(instanceType)) {
			return errors.InstanceTypeNotSupported("instance type '%s' not supported for region '%s'", instanceType.String(), region.Name)
		}
		return nil
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

func ValidateKafkaClaims(ctx context.Context, kafkaRequestPayload *public.KafkaRequestPayload, kafkaRequest *dbapi.KafkaRequest) handlers.Validate {
	return func() *errors.ServiceError {
		kafkaRequest = presenters.ConvertKafkaRequest(*kafkaRequestPayload, kafkaRequest)
		claims, err := auth.GetClaimsFromContext(ctx)
		if err != nil {
			return errors.Unauthenticated("user not authenticated")
		}
		(*kafkaRequest).Owner = auth.GetUsernameFromClaims(claims)
		(*kafkaRequest).OrganisationId = auth.GetOrgIdFromClaims(claims)
		(*kafkaRequest).OwnerAccountId = auth.GetAccountIdFromClaims(claims)

		return nil
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
