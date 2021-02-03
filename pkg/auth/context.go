package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/dgrijalva/jwt-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

// Context key type defined to avoid collisions in other pkgs using context
// See https://golang.org/pkg/context/#WithValue
type contextKey string

const (
	contextUsernameKey contextKey = "username"
	contextOrgIdKey    contextKey = "org_id"

	contextIsAllowedAsServiceAccount contextKey = "user-is-allowed-as-service-account"
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
	return context.WithValue(ctx, contextIsAllowedAsServiceAccount, isAllowedAsServiceAccount)
}

func GetUserIsAllowedAsServiceAccountFromContext(ctx context.Context) bool {
	isAllowedAsServiceAccount := ctx.Value(contextIsAllowedAsServiceAccount)
	if isAllowedAsServiceAccount == nil {
		return false
	}
	return isAllowedAsServiceAccount.(bool)
}

// Sets claim properties in context.
func SetClaimsInContextHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, err := getClaimsFromContext(ctx)
		if err != nil {
			shared.HandleError(ctx, w, errors.ErrorUnauthenticated, err.Error())
			return
		}

		ctx = context.WithValue(ctx, contextUsernameKey, claims[string(contextUsernameKey)])
		ctx = context.WithValue(ctx, contextOrgIdKey, claims[string(contextOrgIdKey)])

		*r = *r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func getClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
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

func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return jwt.MapClaims{}, fmt.Errorf("failed to get jwt token from context: %s", err.Error())
	}
	return token.Claims.(jwt.MapClaims), nil
}
