package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/dgrijalva/jwt-go"
)

func TestOCMAuthorizationMiddleware_RequireIssuer(t *testing.T) {
	tests := []struct {
		name       string
		token      *jwt.Token
		next       http.Handler
		errCode    errors.ServiceErrorCode
		wantIssuer string
		wantCode   int
	}{
		{
			name: "should success when required issuer is set in JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "desiredIssuer",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: "desiredIssuer",
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should fail with the provided error when required issuer is different than the one set in the JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "anotherIssuer",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: "desiredIssuer",
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "should fail with the provided error when issuer field is not set in the JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: "desiredIssuer",
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesHandler := NewOCMAuthorizationMiddleware()
			toTest := setContextToken(rolesHandler.RequireIssuer(tt.wantIssuer, tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			if recorder.Result().StatusCode != tt.wantCode {
				t.Errorf("expected status code %d but got %d", tt.wantCode, recorder.Result().StatusCode)
			}
		})
	}
}
