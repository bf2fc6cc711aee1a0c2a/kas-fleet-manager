package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
)

func TestContext_GetAccountIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantUserIdClaim is empty",
			claims: jwt.MapClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantUserIdClaim is not empty",
			claims: jwt.MapClaims{
				tenantUserIdClaim: "Test_user_id",
			},
			want: "Test_user_id",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountId := GetAccountIdFromClaims(tt.claims)
			Expect(tt.want).To(Equal(accountId))
		})
	}
}

func TestContext_GetIsOrgAdminFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   bool
	}{
		{
			name: "Should return true when tenantOrgAdminClaim is true",
			claims: jwt.MapClaims{
				tenantOrgAdminClaim: true,
			},
			want: true,
		},
		{
			name: "Should return false when tenantOrgAdminClaim is false",
			claims: jwt.MapClaims{
				tenantOrgAdminClaim: false,
			},
			want: false,
		},
		{
			name:   "Should return false when tenantOrgAdminClaim is false",
			claims: jwt.MapClaims{},
			want:   false,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAdmin := GetIsOrgAdminFromClaims(tt.claims)
			Expect(tt.want).To(Equal(isAdmin))
		})
	}
}

func TestContext_GetUsernameFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantUsernameClaim and alternateUsernameClaim empty",
			claims: jwt.MapClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantUsernameClaim is not empty",
			claims: jwt.MapClaims{
				tenantUsernameClaim: "Test Username",
			},
			want: "Test Username",
		},
		{
			name: "Should return when alternateUsernameClaim is not empty",
			claims: jwt.MapClaims{
				alternateTenantUsernameClaim: "Test Alternate Username",
			},
			want: "Test Alternate Username",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username := GetUsernameFromClaims(tt.claims)
			Expect(tt.want).To(Equal(username))
		})
	}
}

func TestContext_GetOrgIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantIdClaim and alternateTenantIdClaim empty",
			claims: jwt.MapClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantIdClaim",
			claims: jwt.MapClaims{
				tenantIdClaim: "Test Tenant ID",
			},
			want: "Test Tenant ID",
		},
		{
			name: "Should return empty when orgId != tenantIdClaim",
			claims: jwt.MapClaims{
				tenantIdClaim: "Test Tenant ID",
			},
			want: "Test Tenant ID",
		},
		{
			name: "Should return when alternateTenantIdClaim",
			claims: jwt.MapClaims{
				alternateTenantIdClaim: "Test alternate Tenant ID",
			},
			want: "Test alternate Tenant ID",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantId := GetOrgIdFromClaims(tt.claims)
			Expect(tt.want).To(Equal(tenantId))
		})
	}
}
