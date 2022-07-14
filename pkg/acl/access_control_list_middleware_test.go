package acl_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/golang/glog"
	"github.com/onsi/gomega"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

var env *environments.Env
var serverConfig *server.ServerConfig

func TestMain(m *testing.M) {
	var err error
	env, err = environments.New(environments.GetEnvironmentStrFromEnv(),
		kafka.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	env.MustResolveAll(&serverConfig)
	os.Exit(m.Run())
}

func TestAccessControlListMiddleware_Authorize(t *testing.T) {
	g := gomega.NewWithT(t)
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, serverConfig.TokenIssuerURL)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	type fields struct {
		accessControlListConfig *acl.AccessControlListConfig
	}
	tests := []struct {
		name            string
		fields          fields
		wantErr         bool
		wantHttpStatus1 http.Handler
		wantHttpStatus  int
	}{
		{
			name: "returns 403 Forbidden response when user is not allowed to access service",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableDenyList: true,
					DenyList:       acl.DeniedUsers{"username"},
				},
			},
			wantErr:        true,
			wantHttpStatus: http.StatusForbidden,
		},
		{
			name: "returns 200 status if denyList is disabled",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableDenyList: false,
				},
			},
			wantErr:        false,
			wantHttpStatus: http.StatusOK,
		},
		{
			name: "returns 200 status if denyList is enabled and deny list is empty",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableDenyList: true,
				},
			},
			wantErr:        false,
			wantHttpStatus: http.StatusOK,
		},
		{
			name: "returns 200 status ok response when organisation is allowed to access service",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableAccessList: true,
					AccessList:       acl.AcceptedOrganisations{"org-id-test"},
				},
			},
			wantErr:        false,
			wantHttpStatus: http.StatusOK,
		},
		{
			name: "returns 200 status if accessList is disabled",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableAccessList: false,
				},
			},
			wantErr:        false,
			wantHttpStatus: http.StatusOK,
		},
		{
			name: "returns 403 status forbidden if accessList is enabled and access list is empty",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableAccessList: true,
				},
			},
			wantErr:        true,
			wantHttpStatus: http.StatusForbidden,
		},
		{
			name: "returns 403 status forbidden response when organisation is not allowed to access service",
			fields: fields{
				&acl.AccessControlListConfig{
					EnableAccessList: true,
					AccessList:       acl.AcceptedOrganisations{"not-test-organisation"},
				},
			},
			wantErr:        true,
			wantHttpStatus: http.StatusForbidden,
		},
	}

	for _, testcase := range tests {
		g := gomega.NewWithT(t)
		tt := testcase
		req, err := http.NewRequest("GET", "/api/kafkas_mgmt/kafkas", nil)
		g.Expect(err).NotTo(gomega.HaveOccurred())

		rr := httptest.NewRecorder()

		middleware := acl.NewAccessControlListMiddleware(tt.fields.accessControlListConfig)
		handler := middleware.Authorize(http.HandlerFunc(NextHandler))

		// create a jwt and set it in the context
		ctx := req.Context()
		acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-test")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		token, err := authHelper.CreateJWTWithClaims(acc, nil)
		g.Expect(err).NotTo(gomega.HaveOccurred())

		ctx = auth.SetTokenInContext(ctx, token)
		req = req.WithContext(ctx)
		handler.ServeHTTP(rr, req)

		body, err := ioutil.ReadAll(rr.Body)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(rr.Code).To(gomega.Equal(tt.wantHttpStatus))

		if tt.wantErr {
			g.Expect(rr.Header().Get("Content-Type")).To(gomega.Equal("application/json"))
			var data map[string]string
			err = json.Unmarshal(body, &data)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(data["kind"]).To(gomega.Equal("Error"))
			// verify that context about user being allowed as service account is set to false always
			ctxAfterMiddleware := req.Context()
			g.Expect(auth.GetFilterByOrganisationFromContext(ctxAfterMiddleware)).To(gomega.Equal(false))
		}
	}
}

// NextHandler is a dummy handler that returns OK when QuotaList middleware has passed
func NextHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) //nolint
	_, err := io.WriteString(w, "OK")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
