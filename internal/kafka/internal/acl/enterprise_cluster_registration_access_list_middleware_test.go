package acl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

func Test_EnterpriseClusterRegistrationAccessListMiddleware(t *testing.T) {
	type args struct {
		config EnterpriseClusterRegistrationAccessControlListConfig
	}
	tests := []struct {
		name    string
		args    args
		token   *jwt.Token
		next    http.Handler
		request *http.Request
		want    int
	}{
		{
			name: "should allow access when organization ID is allowed in the config",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id": "13640203",
				},
			},
			args: args{
				config: EnterpriseClusterRegistrationAccessControlListConfig{
					EnterpriseClusterRegistrationAccessControlList: EnterpriseClusterRegistrationAcceptedOrganizations{
						"13640203",
					},
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodPost, "http://example.com", nil),
			want:    http.StatusOK,
		},
		{
			name: "should not allow access when organization ID is not allowed in the config",
			args: args{
				config: EnterpriseClusterRegistrationAccessControlListConfig{
					EnterpriseClusterRegistrationAccessControlList: EnterpriseClusterRegistrationAcceptedOrganizations{
						"13640203",
					},
				},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"org_id": "invalid",
				},
			},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			request: httptest.NewRequest(http.MethodGet, "http://example.com", nil),
			want:    http.StatusForbidden,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			h := NewEnterpriseClusterRegistrationAccessListMiddleware(&tt.args.config)
			toTest := setContextToken(h.Authorize(tt.next), tt.token)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, tt.request)
			resp := recorder.Result()
			resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, resp.StatusCode)
			}
		})
	}
}

func setContextToken(next http.Handler, token *jwt.Token) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		ctx = authentication.ContextWithToken(ctx, token)
		request = request.WithContext(ctx)
		next.ServeHTTP(writer, request)
	})
}
