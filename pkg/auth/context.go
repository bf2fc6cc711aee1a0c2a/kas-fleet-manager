package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

// Context key type defined to avoid collisions in other pkgs using context
// See https://golang.org/pkg/context/#WithValue
type contextKey string

const (
	// Context Keys
	// FilterByOrganisation is used to determine whether resources are filtered by a user's organisation or as an individual owner
	contextFilterByOrganisation contextKey = "filter-by-organisation"
	contextIsAdmin              contextKey = "is_admin"
)

func GetIsAdminFromContext(ctx context.Context) bool {
	isAdmin := ctx.Value(contextIsAdmin)
	if isAdmin == nil {
		return false
	}
	return isAdmin.(bool)
}

func SetFilterByOrganisationContext(ctx context.Context, filterByOrganisation bool) context.Context {
	return context.WithValue(ctx, contextFilterByOrganisation, filterByOrganisation)
}

func SetIsAdminContext(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, contextIsAdmin, isAdmin)
}

func GetFilterByOrganisationFromContext(ctx context.Context) bool {
	filterByOrganisation := ctx.Value(contextFilterByOrganisation)
	if filterByOrganisation == nil {
		return false
	}
	return filterByOrganisation.(bool)
}

func SetTokenInContext(ctx context.Context, token *jwt.Token) context.Context {
	return authentication.ContextWithToken(ctx, token)
}

func GetClaimsFromContext(ctx context.Context) (KFMClaims, error) {
	var claims KFMClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return claims, fmt.Errorf("failed to get jwt token from context: %v", err)
	}

	if token != nil && token.Claims != nil {
		claims = KFMClaims(token.Claims.(jwt.MapClaims))
	}
	return claims, nil
}
