package auth

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/golang-jwt/jwt/v4"
)

type KFMClaims jwt.MapClaims

func (c *KFMClaims) VerifyIssuer(cmp string, req bool) bool {
	return jwt.MapClaims(*c).VerifyIssuer(cmp, req)
}

func (c *KFMClaims) GetUsername() (string, error) {
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil }, (*c)[tenantUsernameClaim], (*c)[alternateTenantUsernameClaim]); idx != -1 {
		return val.(string), nil
	}
	return "", fmt.Errorf("can't find neither '%s' or '%s' attribute in claims", tenantUsernameClaim, alternateTenantUsernameClaim)
}

func (c *KFMClaims) GetAccountId() (string, error) {
	if (*c)[tenantUserIdClaim] != nil {
		return (*c)[tenantUserIdClaim].(string), nil
	}
	return "", fmt.Errorf("can't find '%s' attribute in claims", tenantUserIdClaim)
}

func (c *KFMClaims) GetOrgId() (string, error) {
	if (*c)[tenantIdClaim] != nil {
		if orgId, ok := (*c)[tenantIdClaim].(string); ok {
			return orgId, nil
		}
	}

	// NOTE: This should be removed once we migrate to sso.redhat.com as it will no longer be needed (TODO: to be removed as part of MGDSTRM-6159)
	if (*c)[alternateTenantIdClaim] != nil {
		if orgId, ok := (*c)[alternateTenantIdClaim].(string); ok {
			return orgId, nil
		}
	}

	return "", fmt.Errorf("can't find neither '%s' or '%s' attribute in claims", tenantIdClaim, alternateTenantIdClaim)
}

func (c *KFMClaims) IsOrgAdmin() bool {
	if (*c)[tenantOrgAdminClaim] != nil {
		return (*c)[tenantOrgAdminClaim].(bool)
	}
	return false
}
