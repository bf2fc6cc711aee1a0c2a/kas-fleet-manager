package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

func setContextToken(next http.Handler, token *jwt.Token) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		ctx = authentication.ContextWithToken(ctx, token)
		request = request.WithContext(ctx)
		next.ServeHTTP(writer, request)
	})
}

func TestRolesAuthMiddleware_RequireRealmRole(t *testing.T) {
	tests := []struct {
		name     string
		token    *jwt.Token
		next     http.Handler
		wantRole string
		want     int
	}{
		{
			name: "should success when required role is presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"realm_access": map[string]interface{}{
						"roles": []interface{}{"test"},
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantRole: "test",
			want:     http.StatusOK,
		},
		{
			name: "should fail when required role is not presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantRole: "test",
			want:     401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesHandler := NewRolesAuhzMiddleware()
			toTest := setContextToken(rolesHandler.RequireRealmRole(tt.wantRole, errors.ErrorUnauthenticated)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			if recorder.Result().StatusCode != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, recorder.Result().StatusCode)
			}
		})
	}
}

func TestRolesAuthMiddleware_RequireRolesForMethods(t *testing.T) {
	tests := []struct {
		name     string
		token    *jwt.Token
		next     http.Handler
		rolesMap map[string][]string
		request  *http.Request
		want     int
	}{
		{
			name: "should allow access when required role is presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"realm_access": map[string]interface{}{
						"roles": []interface{}{"test"},
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			rolesMap: map[string][]string{
				http.MethodGet: {"test"},
			},
			request: httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			want:    http.StatusOK,
		},
		{
			name: "should allow access when any of the required role is presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"realm_access": map[string]interface{}{
						"roles": []interface{}{"test1"},
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			rolesMap: map[string][]string{
				http.MethodGet: {"test", "test1"},
			},
			request: httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			want:    http.StatusOK,
		},
		{
			name: "should not allow access when method is not defined in the roles mapping",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"realm_access": map[string]interface{}{
						"roles": []interface{}{"test"},
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			rolesMap: map[string][]string{
				http.MethodPost: {"test"},
			},
			request: httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			want:    http.StatusUnauthorized,
		},
		{
			name: "should not allow access when required role is not presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			rolesMap: map[string][]string{
				http.MethodGet: {"test"},
			},
			request: httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			want:    http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesHandler := NewRolesAuhzMiddleware()
			toTest := setContextToken(rolesHandler.RequireRolesForMethods(tt.rolesMap, errors.ErrorUnauthenticated)(tt.next), tt.token)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, tt.request)
			if recorder.Result().StatusCode != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, recorder.Result().StatusCode)
			}
		})
	}
}
