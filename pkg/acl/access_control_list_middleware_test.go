package acl_test

import (
	"os"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
)

//const (
//	jwtKeyFile = "test/support/jwt_private_key.pem"
//	jwtCAFile  = "test/support/jwt_ca.pem"
//)

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

// NextHandler is a dummy handler that returns OK when QuotaList middleware has passed
//func NextHandler(w http.ResponseWriter, r *http.Request) {
//	w.WriteHeader(http.StatusOK) //nolint
//	_, err := io.WriteString(w, "OK")
//	if err != nil {
//		panic(err)
//	}
//}
//
//func getAuthenticatedContext(authHelper *auth.AuthHelper, t *testing.T, acc *v1.Account, claims jwt.MapClaims) context.Context {
//	// create a jwt and set it in the context
//	token, err := authHelper.CreateJWTWithClaims(acc, claims)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	return auth.SetTokenInContext(context.Background(), token)
//}
