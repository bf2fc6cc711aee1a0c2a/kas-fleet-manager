package handlers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
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
func ValidateCloudProvider(kafkaRequest *public.KafkaRequestPayload, providerConfig *config.ProviderConfig, action string) handlers.Validate {
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

		return nil
	}
}

func ValidateKafkaUpdateFields(strimziVersion *string, kafkaVersion *string) handlers.Validate {
	return func() *errors.ServiceError {
		if (strimziVersion == nil || len(*strimziVersion) < 1) && (kafkaVersion == nil || len(*kafkaVersion) < 1) {
			return errors.FieldValidationError("Failed to update Kafka Request. Expecting at least one of the following fields: strimzi_version or kafka_version to be provided")
		}
		return nil
	}
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

		if kafkaUpdateReq.Owner != nil {
			validationError := handlers.ValidateMinLength(kafkaUpdateReq.Owner, "owner", 1)()
			if validationError != nil {
				return validationError
			}
			// only organisation admin where the kafka was created is allowed to change the owner of a Kafka
			if !isOrgAdmin || kafkaRequest.OrganisationId != orgId {
				return errors.New(errors.ErrorUnauthorized, "User not authorized to perform this action")
			}

			userValid, err := authService.CheckUserValid(*kafkaUpdateReq.Owner, orgId)
			if err != nil {
				return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to update kafka request owner")
			}
			if !userValid {
				return errors.NewWithCause(errors.ErrorBadRequest, err, "User %s does not belong in your organization", *kafkaUpdateReq.Owner)
			}
		}

		if kafkaUpdateReq.ReauthenticationEnabled != nil {
			// only Kafka owner or organisation admin is allowed to perform the action
			isOwner := (isOrgAdmin || kafkaRequest.Owner == username) && kafkaRequest.OrganisationId == orgId
			if !isOwner {
				return errors.New(errors.ErrorUnauthorized, "User not authorized to perform this action")
			}
		}

		return nil
	}
}
