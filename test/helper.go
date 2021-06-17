package test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/segmentio/ksuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	privateopenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
//var jwkURL string

// TimeFunc defines a way to get a new Time instance common to the entire test suite.
// Aria's environment has Virtual Time that may not be actual time. We compensate
// by synchronizing on a common time func attached to the test harness.
type TimeFunc func() time.Time

type Services struct {
	di.Inject
	DBFactory             *db.ConnectionFactory
	AppConfig             *config.ApplicationConfig
	MetricsServer         *server.MetricsServer
	HealthCheckServer     *server.HealthCheckServer
	Workers               []workers.Worker
	LeaderElectionManager *workers.LeaderElectionManager
	SignalBus             signalbus.SignalBus
	APIServer             *server.ApiServer
	BootupServices        []provider.BootService
}

type Helper struct {
	AuthHelper    *auth.AuthHelper
	JWTPrivateKey *rsa.PrivateKey
	JWTCA         *rsa.PublicKey
	T             *testing.T
	Env           *environments.Env

	Services
}

// NewID creates a new unique ID used internally to CS
func (helper *Helper) NewID() string {
	return ksuid.New().String()
}

// NewUUID creates a new unique UUID, which has different formatting than ksuid
// UUID is used by telemeter and we validate the format.
func (helper *Helper) NewUUID() string {
	return uuid.New().String()
}

func (helper *Helper) RestURL(path string) string {
	var serverConfig *config.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	protocol := "http"
	if serverConfig.EnableHTTPS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/api/kafkas_mgmt/v1%s", protocol, serverConfig.BindAddress, path)
}

func (helper *Helper) MetricsURL(path string) string {
	var metricsConfig *config.MetricsConfig
	helper.Env.MustResolveAll(&metricsConfig)
	return fmt.Sprintf("http://%s%s", metricsConfig.BindAddress, path)
}

func (helper *Helper) HealthCheckURL(path string) string {
	var healthCheckConfig *config.HealthCheckConfig
	helper.Env.MustResolveAll(&healthCheckConfig)
	return fmt.Sprintf("http://%s%s", healthCheckConfig.BindAddress, path)
}

func (helper *Helper) NewApiClient() *openapi.APIClient {
	var serverConfig *config.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := openapi.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := openapi.NewAPIClient(openapiConfig)
	return client
}

func (helper *Helper) NewPrivateAPIClient() *privateopenapi.APIClient {
	var serverConfig *config.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := privateopenapi.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := privateopenapi.NewAPIClient(openapiConfig)
	return client
}

// NewRandAccount returns a random account that has the control plane team org id as its organisation id
// The org id value is taken from config/allow-list-configuration.yaml
func (helper *Helper) NewRandAccount() *amv1.Account {
	// this value if taken from config/allow-list-configuration.yaml
	orgId := "13640203"
	return helper.NewAccountWithNameAndOrg(faker.Name(), orgId)
}

func (helper *Helper) NewAccountWithNameAndOrg(name string, orgId string) *amv1.Account {
	account, err := helper.AuthHelper.NewAccount(helper.NewID(), name, faker.Email(), orgId)
	if err != nil {
		helper.T.Errorf("failed to create a new account: %s", err.Error())
	}
	return account
}

func (helper *Helper) NewAllowedServiceAccount() *amv1.Account {
	// this value if taken from config/allow-list-configuration.yaml
	allowedSA := "testuser1@example.com"
	account, err := helper.AuthHelper.NewAccount(allowedSA, allowedSA, allowedSA, "")
	if err != nil {
		helper.T.Errorf("failed to create a new service account: %s", err.Error())
	}
	return account
}

func (helper *Helper) NewAccount(username, name, email string, orgId string) *amv1.Account {
	account, err := helper.AuthHelper.NewAccount(username, name, email, orgId)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a new account: %s", err.Error()))
	}
	return account
}

// Returns an authenticated context that can be used with openapi functions
func (helper *Helper) NewAuthenticatedContext(account *amv1.Account, claims jwt.MapClaims) context.Context {
	token, err := helper.AuthHelper.CreateSignedJWT(account, claims)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token: %s", err.Error()))
	}

	return context.WithValue(context.Background(), openapi.ContextAccessToken, token)
}

func (helper *Helper) StartJWKCertServerMock() (string, func()) {
	return mocks.NewJWKCertServerMock(helper.T, helper.JWTCA, auth.JwkKID)
}

func (helper *Helper) DeleteAll(table interface{}) {
	gorm := helper.DBFactory.New()
	err := gorm.Model(table).Unscoped().Delete(table).Error
	if err != nil {
		helper.T.Errorf("error deleting from table %v: %v", table, err)
	}
}

func (helper *Helper) Delete(obj interface{}) {
	gorm := helper.DBFactory.New()
	err := gorm.Unscoped().Delete(obj).Error
	if err != nil {
		helper.T.Errorf("error deleting object %v: %v", obj, err)
	}
}

func (helper *Helper) SkipIfShort() {
	if testing.Short() {
		helper.T.Skip("Skipping execution of test in short mode")
	}
}

func (helper *Helper) Count(table string) int64 {
	gorm := helper.DBFactory.New()
	var count int64
	err := gorm.Table(table).Count(&count).Error
	if err != nil {
		helper.T.Errorf("error getting count for table %s: %v", table, err)
	}
	return count
}

func (helper *Helper) MigrateDB() {
	db.Migrate(helper.DBFactory)
}

func (helper *Helper) MigrateDBTo(migrationID string) {
	db.MigrateTo(helper.DBFactory, migrationID)
}

func (helper *Helper) ClearAllTables() {
	helper.DeleteAll(&api.KafkaRequest{})
}

func (helper *Helper) CleanDB() {
	db.RollbackAll(helper.DBFactory)
}

func (helper *Helper) ResetDB() {
	helper.CleanDB()
	helper.MigrateDB()
}

func (helper *Helper) CreateJWTString(account *amv1.Account) string {
	token, err := helper.AuthHelper.CreateSignedJWT(account, nil)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token: %s", err.Error()))
	}
	return token
}

func (helper *Helper) CreateJWTStringWithClaim(account *amv1.Account, jwtClaims jwt.MapClaims) string {
	token, err := helper.AuthHelper.CreateSignedJWT(account, jwtClaims)
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to create a signed token with the given claims: %s", err.Error()))
	}
	return token
}

func (helper *Helper) CreateJWTToken(account *amv1.Account, jwtClaims jwt.MapClaims) *jwt.Token {
	token, err := helper.AuthHelper.CreateJWTWithClaims(account, jwtClaims)
	if err != nil {
		helper.T.Errorf("Failed to create jwt token: %s", err.Error())
	}
	return token
}

// Convert an error response from the openapi client to an openapi error struct
func (helper *Helper) OpenapiError(err error) openapi.Error {
	generic := err.(openapi.GenericOpenAPIError)
	var exErr openapi.Error
	jsonErr := json.Unmarshal(generic.Body(), &exErr)
	if jsonErr != nil {
		helper.T.Errorf("Unable to convert error response to openapi error: %s", jsonErr)
	}
	return exErr
}
