package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

func TestRequireTermsAcceptanceMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		client   ocm.Client
		next     http.Handler
		wantCode int
	}{
		{
			name:    "should fail if terms checks is enabled and terms are required",
			enabled: true,
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
			name:    "should succeed if terms check is not a enabled even and terms are required",
			enabled: false,
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
			name:    "should succeed if terms checks is enabled and terms are not required",
			enabled: true,
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			requireTermsAcceptanceHandler := NewRequireTermsAcceptanceMiddleware()
			toTest := requireTermsAcceptanceHandler.RequireTermsAcceptance(tt.enabled, tt.client, errors.ErrorTermsNotAccepted)(tt.next)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			resp := recorder.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantCode))
		})
	}
}
