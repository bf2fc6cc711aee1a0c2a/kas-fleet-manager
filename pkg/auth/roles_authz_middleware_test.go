package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/shared"
	"net/http"
	"net/http/httptest"
	"testing"
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
