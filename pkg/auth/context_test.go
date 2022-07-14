package auth

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
)

func TestContext_GetAccountIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims KFMClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantUserIdClaim is empty",
			claims: KFMClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantUserIdClaim is not empty",
			claims: KFMClaims{
				tenantUserIdClaim: "Test_user_id",
			},
			want: "Test_user_id",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			accountId, _ := tt.claims.GetAccountId()
			g.Expect(accountId).To(gomega.Equal(tt.want))
		})
	}
}

func TestContext_GetIsOrgAdminFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims KFMClaims
		want   bool
	}{
		{
			name: "Should return true when tenantOrgAdminClaim is true",
			claims: KFMClaims{
				tenantOrgAdminClaim: true,
			},
			want: true,
		},
		{
			name: "Should return false when tenantOrgAdminClaim is false",
			claims: KFMClaims{
				tenantOrgAdminClaim: false,
			},
			want: false,
		},
		{
			name:   "Should return false when tenantOrgAdminClaim is false",
			claims: KFMClaims{},
			want:   false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.claims.IsOrgAdmin()).To(gomega.Equal(tt.want))
		})
	}
}

func TestContext_GetUsernameFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims KFMClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantUsernameClaim and alternateUsernameClaim empty",
			claims: KFMClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantUsernameClaim is not empty",
			claims: KFMClaims{
				tenantUsernameClaim: "Test Username",
			},
			want: "Test Username",
		},
		{
			name: "Should return when alternateUsernameClaim is not empty",
			claims: KFMClaims{
				alternateTenantUsernameClaim: "Test Alternate Username",
			},
			want: "Test Alternate Username",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			username, _ := tt.claims.GetUsername()
			g.Expect(username).To(gomega.Equal(tt.want))
		})
	}
}

func TestContext_GetOrgIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims KFMClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantIdClaim and alternateTenantIdClaim empty",
			claims: KFMClaims{},
			want:   "",
		},
		{
			name: "Should return tenantIdClaim when tenantIdClaim is not empty",
			claims: KFMClaims{
				tenantIdClaim: "Test Tenant ID",
			},
			want: "Test Tenant ID",
		},
		{
			name: "Should return alternateTenantIdClaim when alternateTenantIdClaim is not empty",
			claims: KFMClaims{
				alternateTenantIdClaim: "Test alternate Tenant ID",
			},
			want: "Test alternate Tenant ID",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			orgId, _ := tt.claims.GetOrgId()
			g.Expect(orgId).To(gomega.Equal(tt.want))
		})
	}
}

func TestContext_GetIsAdminFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "return false if isAdmin is false",
			ctx:  SetIsAdminContext(context.TODO(), false),
			want: false,
		},
		{
			name: "return true if isAdmin is true",
			ctx:  SetIsAdminContext(context.TODO(), true),
			want: true,
		},
		{
			name: "return false if isAdmin is nil",
			ctx:  SetFilterByOrganisationContext(context.TODO(), false),
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(GetIsAdminFromContext(tt.ctx)).To(gomega.Equal(tt.want))
		})
	}
}

func TestContext_GetFilterByOrganisationFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "return false if filterByOrganisation is false",
			ctx:  SetFilterByOrganisationContext(context.TODO(), false),
			want: false,
		},
		{
			name: "return true if filterByOrganisation is true",
			ctx:  SetFilterByOrganisationContext(context.TODO(), true),
			want: true,
		},
		{
			name: "return false if filterByOrganisaiton is nil",
			ctx:  SetIsAdminContext(context.TODO(), true),
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(GetFilterByOrganisationFromContext(tt.ctx)).To(gomega.Equal(tt.want))
		})
	}
}
