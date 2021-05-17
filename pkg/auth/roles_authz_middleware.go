package auth

import (
	"net/http"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/dgrijalva/jwt-go"
)

// RolesAuthorizationMiddleware can be used to perform RBAC authorization checks on endpoints
type RolesAuthorizationMiddleware interface {
	// RequireRealmRole will check the given realm role exists in the request token
	RequireRealmRole(roleName string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type rolesAuthMiddleware struct{}

var _ RolesAuthorizationMiddleware = &rolesAuthMiddleware{}

func NewRolesAuhzMiddleware() RolesAuthorizationMiddleware {
	return &rolesAuthMiddleware{}
}

func (m *rolesAuthMiddleware) RequireRealmRole(roleName string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(ctx, writer, serviceErr)
				return
			}
			roles := getRealmRolesClaim(claims)
			if hasRole(roles, roleName) {
				next.ServeHTTP(writer, request)
			} else {
				shared.HandleError(ctx, writer, serviceErr)
				return
			}
		})
	}
}

func getRealmRolesClaim(claims jwt.MapClaims) []string {
	if realmRoles, ok := claims["realm_access"]; ok {
		if roles, ok := realmRoles.(map[string]interface{}); ok {
			if arr, ok := roles["roles"].([]interface{}); ok {
				var r []string
				for _, i := range arr {
					r = append(r, i.(string))
				}
				return r
			}
		}
	}
	return []string{}
}

func hasRole(roles []string, roleName string) bool {
	for _, role := range roles {
		if strings.EqualFold(role, roleName) {
			return true
		}
	}
	return false
}
