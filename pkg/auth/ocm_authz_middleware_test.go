package auth

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
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

func TestOCMAuthorizationMiddleware_RequireTermsAcceptance(t *testing.T) {
	tests := []struct {
		name     string
		product  bool
		client   ocm.Client
		next     http.Handler
		wantCode int
	}{
		{
			name:    "should fail if we are a product and terms are required",
			product: true,
			client: &ocm.ClientMock{
				GetRequiresTermsAcceptanceFunc: func(username string) (bool, string, error) {
					return true, "", nil
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusForbidden,
		},
		{
			name:    "should succeed if we are not a product even if terms are required",
			product: false,
			client: &ocm.ClientMock{
				GetRequiresTermsAcceptanceFunc: func(username string) (bool, string, error) {
					return true, "", nil
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name:    "should succeed if we are a product and terms are not required",
			product: true,
			client: &ocm.ClientMock{
				GetRequiresTermsAcceptanceFunc: func(username string) (bool, string, error) {
					return false, "", nil
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesHandler := NewOCMAuthorizationMiddleware()
			toTest := rolesHandler.RequireTermsAcceptance(tt.product, tt.client, errors.ErrorTermsNotAccepted)(tt.next)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			if recorder.Result().StatusCode != tt.wantCode {
				t.Errorf("expected status code %d but got %d", tt.wantCode, recorder.Result().StatusCode)
			}
		})
	}
}
