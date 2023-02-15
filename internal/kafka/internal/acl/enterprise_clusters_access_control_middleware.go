package acl

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type EnterpriseClustersAccessControlMiddleware struct {
	enterpriseBillingModel *config.KafkaBillingModel
	quotaService           services.QuotaService
}

func NewEnterpriseClustersAccessControlMiddleware(kafkaConfig *config.KafkaConfig, quotaServiceFactory services.QuotaServiceFactory) *EnterpriseClustersAccessControlMiddleware {
	middleware := &EnterpriseClustersAccessControlMiddleware{}
	standardInstanceTypeConfig, err := kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(types.STANDARD.String())
	if err != nil {
		logger.Logger.Error(err)
	}

	if standardInstanceTypeConfig != nil {
		middleware.enterpriseBillingModel, err = standardInstanceTypeConfig.GetKafkaSupportedBillingModelByID(constants.BillingModelEnterprise.String())
		if err != nil {
			logger.Logger.Error(err)
		}
	}

	quotaService, quotaServiceErrors := quotaServiceFactory.GetQuotaService(api.QuotaType(kafkaConfig.Quota.Type))
	if quotaServiceErrors != nil {
		logger.Logger.Error(quotaServiceErrors)
	} else {
		middleware.quotaService = quotaService
	}

	return middleware
}

// Middleware handler to authorize users based on the granted quota to create Kafka of standard instance types
func (middleware *EnterpriseClustersAccessControlMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.enterpriseBillingModel == nil {
			shared.HandleError(r, w, errors.New(errors.ErrorGeneral, "enterprise billing model configuration error"))
			return
		}

		if middleware.quotaService == nil {
			shared.HandleError(r, w, errors.New(errors.ErrorGeneral, "quota service configuration error"))
			return
		}

		context := r.Context()
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			shared.HandleError(r, w, errors.NewWithCause(errors.ErrorUnauthorized, err, ""))
			return
		}

		orgId, _ := claims.GetOrgId()
		username, _ := claims.GetUsername()

		hasQuota, quotaCheckErrs := middleware.quotaService.CheckIfQuotaIsDefinedForInstanceType(username, orgId, types.STANDARD, *middleware.enterpriseBillingModel)
		if quotaCheckErrs != nil {
			shared.HandleError(r, w, errors.NewWithCause(errors.ErrorGeneral, quotaCheckErrs, ""))
			return
		}

		if !hasQuota {
			shared.HandleError(r, w, errors.New(errors.ErrorUnauthorized, "organization '%s' is not authorized to access the service", orgId))
			return
		}

		next.ServeHTTP(w, r)
	})
}
