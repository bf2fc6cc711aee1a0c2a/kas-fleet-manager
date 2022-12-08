package acl

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

type EnterpriseClusterRegistrationAccessListMiddleware struct {
	enterpriseClusterRegistrationAccessControlListConfig *EnterpriseClusterRegistrationAccessControlListConfig
}

func NewEnterpriseClusterRegistrationAccessListMiddleware(enterpriseClusterRegistrationAccessControlListConfig *EnterpriseClusterRegistrationAccessControlListConfig) *EnterpriseClusterRegistrationAccessListMiddleware {
	middleware := EnterpriseClusterRegistrationAccessListMiddleware{
		enterpriseClusterRegistrationAccessControlListConfig: enterpriseClusterRegistrationAccessControlListConfig,
	}
	return &middleware
}

// Middleware handler to authorize users based on the provided ACL configuration
func (middleware *EnterpriseClusterRegistrationAccessListMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := r.Context()
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			shared.HandleError(r, w, errors.NewWithCause(errors.ErrorForbidden, err, ""))
			return
		}

		orgId, _ := claims.GetOrgId()

		orgIsAccepted := middleware.enterpriseClusterRegistrationAccessControlListConfig.EnterpriseClusterRegistrationAccessControlList.IsOrganizationAccepted(orgId)
		if !orgIsAccepted {
			shared.HandleError(r, w, errors.New(errors.ErrorForbidden, "organization '%s' is not authorized to access the service", orgId))
			return
		}

		next.ServeHTTP(w, r)
	})
}
