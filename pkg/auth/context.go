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
	contextUsernameKey                  contextKey = "username"
	contextOrgIdKey                     contextKey = "org_id"
	contextIsAllowedAsServiceAccountKey contextKey = "user-is-allowed-as-service-account"
)

func SetUsernameContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, contextUsernameKey, username)
}

func GetUsernameFromContext(ctx context.Context) string {
	username := ctx.Value(contextUsernameKey)
	if username == nil {
		return ""
	}
	return fmt.Sprintf("%v", username)
}

func SetOrgIdContext(ctx context.Context, orgId string) context.Context {
	return context.WithValue(ctx, contextOrgIdKey, orgId)
}

func GetOrgIdFromContext(ctx context.Context) string {
	orgId := ctx.Value(contextOrgIdKey)
	if orgId == nil {
		return ""
	}
	return fmt.Sprintf("%v", orgId)
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
