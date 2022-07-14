package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

func TestAuditLogMiddleware_AuditLog(t *testing.T) {
	tests := []struct {
		name     string
		token    *jwt.Token
		next     http.Handler
		errCode  errors.ServiceErrorCode
		wantCode int
	}{
		{
			name: "should pass for tenant Username",
			token: &jwt.Token{Claims: jwt.MapClaims{
				"username": "test user",
			}},
			errCode: errors.ErrorBadRequest,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should pass for ssoUsernameKey",
			token: &jwt.Token{Claims: jwt.MapClaims{
				"preferred_username": "test user",
			}},
			errCode: errors.ErrorBadRequest,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			auditLogMW := NewAuditLogMiddleware()
			toTest := setContextToken(auditLogMW.AuditLog(tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			resp := recorder.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantCode))
			_ = resp.Body.Close()
		})
	}
}
