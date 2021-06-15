package test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/goava/di"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
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
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
var jwkURL string

// TimeFunc defines a way to get a new Time instance common to the entire test suite.
// Aria's environment has Virtual Time that may not be actual time. We compensate
// by synchronizing on a common time func attached to the test harness.
type TimeFunc func() time.Time

type Services struct {
	di.Inject
	DBFactory         *db.ConnectionFactory
	AppConfig         *config.ApplicationConfig
	MetricsServer     *server.MetricsServer
	HealthCheckServer *server.HealthCheckServer
	ClusterWorker     *workers.ClusterManager
	KafkaWorkers      []workers.Worker
	LeaderEleWorker   *workers.LeaderElectionManager
}

type Helper struct {
	AuthHelper    *auth.AuthHelper
	JWTPrivateKey *rsa.PrivateKey
	JWTCA         *rsa.PublicKey
	T             *testing.T
	Env           *environments.Env

	APIServer *server.ApiServer
	Services
}

func NewHelper(t *testing.T) *Helper {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, env.Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Errorf("failed to create a new auth helper %s", err.Error())
	}
	env.Config.OSDClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
	env.Config.Kafka.KafkaLifespan.EnableDeletionOfExpiredKafka = true
	db.KafkaAdditionalLeasesExpireTime = time.Now().Add(-time.Minute) // set kafkas lease as expired so that a new leader is elected for each of the leases

	return &Helper{
		//AppConfig:     env.Config,
		//DBFactory:     env.DBFactory,
		JWTPrivateKey: authHelper.JWTPrivateKey,
		JWTCA:         authHelper.JWTCA,
		AuthHelper:    authHelper,
		T:             t,
		Env:           env,
	}
}

func (helper *Helper) startAPIServer() {
	// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
	helper.Env.Config.Server.JwksURL = jwkURL
	helper.Env.Config.Keycloak.EnableAuthenticationOnKafka = false

	if err := helper.Env.ServiceContainer.Resolve(&helper.APIServer); err != nil {
		glog.Fatalf("di failure: %v", err)
	}

	listener, err := helper.APIServer.Listen()
	if err != nil {
		glog.Fatalf("Unable to start Test API server: %s", err)
	}
	go func() {
		glog.V(10).Info("Test API server started")
		helper.APIServer.Serve(listener)
		glog.V(10).Info("Test API server stopped")
	}()
}

func (helper *Helper) stopAPIServer() {
	if err := helper.APIServer.Stop(); err != nil {
		glog.Fatalf("Unable to stop api server: %s", err.Error())
	}
}

func (helper *Helper) startMetricsServer() {
	go func() {
		glog.V(10).Info("Test Metrics server started")
		helper.MetricsServer.Start()
		glog.V(10).Info("Test Metrics server stopped")
	}()
}

func (helper *Helper) stopMetricsServer() {
	if err := helper.MetricsServer.Stop(); err != nil {
		glog.Fatalf("Unable to stop metrics server: %s", err.Error())
	}
}

func (helper *Helper) startHealthCheckServer() {
	go func() {
		glog.V(10).Info("Test health check server started")
		helper.HealthCheckServer.Start()
		glog.V(10).Info("Test health check server stopped")
	}()
}
func (helper *Helper) stopHealthCheckServer() {
	if err := helper.HealthCheckServer.Stop(); err != nil {
		glog.Fatalf("Unable to stop heal check server: %s", err.Error())
	}
}

func (helper *Helper) StartSignalBusWorker() {
	env := helper.Env
	glog.V(10).Info("Signal bus worker started")
	env.Services.SignalBus.(*signalbus.PgSignalBus).Start()
}

func (helper *Helper) StopSignalBusWorker() {
	env := helper.Env
	env.Services.SignalBus.(*signalbus.PgSignalBus).Stop()
	glog.V(10).Info("Signal bus worker stopped")
}

func (helper *Helper) startLeaderElectionWorker() {
	helper.LeaderEleWorker.Start()
	glog.V(10).Info("Test Leader Election Manager started")
}

func (helper *Helper) stopLeaderElectionWorker() {
	if helper.LeaderEleWorker == nil {
		return
	}
	helper.LeaderEleWorker.Stop()
}

func (helper *Helper) StartLeaderElectionWorker() {
	helper.stopLeaderElectionWorker()
	helper.startLeaderElectionWorker()
}

func (helper *Helper) StopLeaderElectionWorker() {
	helper.stopLeaderElectionWorker()
}

func (helper *Helper) StartServer() {
	helper.startAPIServer()
	glog.V(10).Info("Test API server started")
}

func (helper *Helper) StopServer() {
	helper.stopAPIServer()
	glog.V(10).Info("Test API server stopped")
}

func (helper *Helper) RestartServer() {
	helper.stopAPIServer()
	helper.startAPIServer()
	glog.V(10).Info("Test API server restarted")
}

func (helper *Helper) RestartMetricsServer() {
	helper.stopMetricsServer()
	helper.startMetricsServer()
	glog.V(10).Info("Test metrics server restarted")
}

// ResetMetrics metrics. Note this will only reset metrics defined in pkg/metrics
func (helper *Helper) ResetMetrics() {
	metrics.Reset()
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
	protocol := "http"
	if helper.AppConfig.Server.EnableHTTPS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/api/kafkas_mgmt/v1%s", protocol, helper.AppConfig.Server.BindAddress, path)
}

func (helper *Helper) MetricsURL(path string) string {
	return fmt.Sprintf("http://%s%s", helper.AppConfig.Metrics.BindAddress, path)
}

func (helper *Helper) HealthCheckURL(path string) string {
	return fmt.Sprintf("http://%s%s", helper.AppConfig.HealthCheck.BindAddress, path)
}

func (helper *Helper) NewApiClient() *openapi.APIClient {
	config := openapi.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", helper.AppConfig.Server.BindAddress)
	client := openapi.NewAPIClient(config)
	return client
}

func (helper *Helper) NewPrivateAPIClient() *privateopenapi.APIClient {
	config := privateopenapi.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", helper.AppConfig.Server.BindAddress)
	client := privateopenapi.NewAPIClient(config)
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

func (helper *Helper) StartJWKCertServerMock() (teardown func()) {
	jwkURL, teardown = mocks.NewJWKCertServerMock(helper.T, helper.JWTCA, auth.JwkKID)
	helper.Env.Config.Server.JwksURL = jwkURL
	return teardown
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
