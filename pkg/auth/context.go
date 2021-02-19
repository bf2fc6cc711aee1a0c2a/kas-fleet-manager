package auth

import (
	"context"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

// Context key type defined to avoid collisions in other pkgs using context
// See https://golang.org/pkg/context/#WithValue
type contextKey string

const (
	// Context Keys
	contextIsAllowedAsServiceAccountKey contextKey = "user-is-allowed-as-service-account"

	// Claims Keys
	ocmUsernameKey string = "username"
	ocmOrgIdKey    string = "org_id"

	masSsoOrgIdKey string = "rh-org-id"
)

func GetUsernameFromClaims(claims jwt.MapClaims) string {
	if claims[ocmUsernameKey] == nil {
		return ""
	}
	return claims[ocmUsernameKey].(string)
}

func GetOrgIdFromClaims(claims jwt.MapClaims) string {
	if claims[ocmOrgIdKey] != nil {
		return claims[ocmOrgIdKey].(string)
	}

	if claims[masSsoOrgIdKey] != nil {
		return claims[masSsoOrgIdKey].(string)
	}

	return ""
}

func SetUserIsAllowedAsServiceAccountContext(ctx context.Context, isAllowedAsServiceAccount bool) context.Context {
	return context.WithValue(ctx, contextIsAllowedAsServiceAccountKey, isAllowedAsServiceAccount)
}

func GetUserIsAllowedAsServiceAccountFromContext(ctx context.Context) bool {
	isAllowedAsServiceAccount := ctx.Value(contextIsAllowedAsServiceAccountKey)
	if isAllowedAsServiceAccount == nil {
		return false
	}
	return isAllowedAsServiceAccount.(bool)
}

func SetTokenInContext(ctx context.Context, token *jwt.Token) context.Context {
	return authentication.ContextWithToken(ctx, token)
}

func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return claims, fmt.Errorf("failed to get jwt token from context: %v", err)
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}
	return claims, nil
}
