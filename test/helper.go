package test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"github.com/goava/di"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers/kafka_mgrs"

	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
	"github.com/spf13/pflag"

	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	privateopenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
)

var helper *Helper
var once sync.Once

// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
var jwkURL string

// TimeFunc defines a way to get a new Time instance common to the entire test suite.
// Aria's environment has Virtual Time that may not be actual time. We compensate
// by synchronizing on a common time func attached to the test harness.
type TimeFunc func() time.Time

type Helper struct {
	DBFactory         *db.ConnectionFactory
	AppConfig         *config.ApplicationConfig
	APIServer         server.Server
	MetricsServer     server.Server
	HealthCheckServer server.Server
	KafkaWorkers      []workers.Worker
	ClusterWorker     *workers.ClusterManager
	LeaderEleWorker   *workers.LeaderElectionManager
	AuthHelper        *auth.AuthHelper
	TimeFunc          TimeFunc
	JWTPrivateKey     *rsa.PrivateKey
	JWTCA             *rsa.PublicKey
	T                 *testing.T
	teardowns         []func()
}

func NewHelper(t *testing.T, server *httptest.Server) *Helper {
	once.Do(func() {
		env := environments.Environment()

		// Set server if provided
		if server != nil {
			fmt.Printf("Setting OCM base URL to %s\n", server.URL)
			env.Config.OCM.BaseURL = server.URL
		}

		// Manually set environment name, ignoring environment variables
		validTestEnv := false
		for _, testEnv := range []string{environments.TestingEnv, environments.IntegrationEnv, environments.DevelopmentEnv} {
			if env.Name == testEnv {
				validTestEnv = true
				break
			}
		}
		if !validTestEnv {
			fmt.Println("OCM_ENV environment variable not set to a valid test environment, using default testing environment")
			env.Name = environments.TestingEnv
		}
		err := env.AddFlags(pflag.CommandLine)
		if err != nil {
			glog.Fatalf("Unable to add environment flags: %s", err.Error())
		}
		if logLevel := os.Getenv("LOGLEVEL"); logLevel != "" {
			glog.Infof("Using custom loglevel: %s", logLevel)
			err = pflag.CommandLine.Set("v", logLevel)
			if err != nil {
				glog.Warningf("Unable to set custom logLevel: %s", err.Error())
			}
		}
		pflag.Parse()

		err = env.Initialize()
		if err != nil {
			glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
		}

		authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, environments.Environment().Config.OCM.TokenIssuerURL)
		if err != nil {
			helper.T.Errorf("failed to create a new auth helper %s", err.Error())
		}

		helper = &Helper{
			AppConfig:     environments.Environment().Config,
			DBFactory:     environments.Environment().DBFactory,
			JWTPrivateKey: authHelper.JWTPrivateKey,
			JWTCA:         authHelper.JWTCA,
			AuthHelper:    authHelper,
		}

		helper.Env().Config.OSDClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
		helper.Env().Config.Kafka.KafkaLifespan.EnableDeletionOfExpiredKafka = true

		db.KafkaAdditionalLeasesExpireTime = time.Now().Add(-time.Minute) // set kafkas lease as expired so that a new leader is elected for each of the leases

		// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
		jwkMockTeardown := helper.StartJWKCertServerMock()
		helper.teardowns = []func(){
			helper.CleanDB,
			jwkMockTeardown,
			helper.stopAPIServer,
		}
		helper.startMetricsServer()
		helper.startHealthCheckServer()
	})
	helper.T = t
	return helper
}

func (helper *Helper) SetServer(server *httptest.Server) {
	helper.Env().Config.OCM.BaseURL = server.URL
	err := helper.Env().LoadClients()
	if err != nil {
		glog.Fatalf("Unable to load clients: %s", err.Error())
	}
	err = helper.Env().LoadServices()
	if err != nil {
		glog.Fatalf("Unable to load services: %s", err.Error())
	}
}

func (helper *Helper) Env() *environments.Env {
	return environments.Environment()
}

func (helper *Helper) Teardown() {
	for _, f := range helper.teardowns {
		f()
	}
}

func (helper *Helper) startAPIServer() {
	// TODO jwk mock server needs to be refactored out of the helper and into the testing environment
	helper.Env().Config.Server.JwksURL = jwkURL
	helper.Env().Config.Keycloak.EnableAuthenticationOnKafka = false
	helper.APIServer = server.NewAPIServer()
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
	helper.MetricsServer = server.NewMetricsServer()
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
	helper.HealthCheckServer = server.NewHealthCheckServer()
	go func() {
		glog.V(10).Info("Test health check server started")
		helper.HealthCheckServer.Start()
		glog.V(10).Info("Test health check server stopped")
	}()
}

func (helper *Helper) startKafkaWorkers() {
	env := helper.Env()
	var kafkaWorkerList []workers.Worker
	kafkaWorker := kafka_mgrs.NewKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.Config, env.Services.SignalBus)
	acceptedKafkaManager := kafka_mgrs.NewAcceptedKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.Config, env.QuotaServiceFactory, env.Services.ClusterPlmtStrategy, env.Services.SignalBus)
	preparingKafkaManager := kafka_mgrs.NewPreparingKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.SignalBus)
	deletingKafkaManager := kafka_mgrs.NewDeletingKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.Config, env.QuotaServiceFactory, env.Services.SignalBus)
	provisioningKafkaManager := kafka_mgrs.NewProvisioningKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.Observatorium, env.Services.Config, env.Services.SignalBus)
	readyKafkaManager := kafka_mgrs.NewReadyKafkaManager(env.Services.Kafka, uuid.New().String(), env.Services.Keycloak, env.Services.Config, env.Services.SignalBus)
	helper.KafkaWorkers = append(kafkaWorkerList, kafkaWorker, acceptedKafkaManager, preparingKafkaManager, deletingKafkaManager, provisioningKafkaManager, readyKafkaManager)

	go func() {
		glog.V(10).Info("Test Metrics server started")
		for _, worker := range helper.KafkaWorkers {
			worker.Start()
		}
		glog.V(10).Info("Test Metrics server stopped")
	}()
}

func (helper *Helper) stopKafkaWorkers() {
	if len(helper.KafkaWorkers) < 1 {
		return
	}

	for _, worker := range helper.KafkaWorkers {
		worker.Stop()
	}
}

func (helper *Helper) startClusterWorker() {
	// start cluster worker
	services := helper.Env().Services
	helper.ClusterWorker = workers.NewClusterManager(services.Cluster, services.CloudProviders,
		services.Config, uuid.New().String(), services.KasFleetshardAddonService, services.OsdIdpKeycloak, services.SignalBus)
	go func() {
		glog.V(10).Info("Test Metrics server started")
		helper.ClusterWorker.Start()
		glog.V(10).Info("Test Metrics server stopped")
	}()
}

func (helper *Helper) stopClusterWorker() {
	if helper.ClusterWorker == nil {
		return
	}
	helper.ClusterWorker.Stop()
}

//func (helper *Helper) startConnectorWorker() {
//	env := helper.Env()
//	if err := env.ServiceContainer.Resolve(&helper.ConnectorWorker); err != nil {
//		helper.T.Fatalf("ConnectorWorker not found: %s", err)
//	}
//	go func() {
//		glog.V(10).Info("Connector worker started")
//		helper.ConnectorWorker.Start()
//	}()
//}
//
//func (helper *Helper) stopConnectorWorker() {
//	if helper.ConnectorWorker == nil {
//		return
//	}
//	helper.ConnectorWorker.Stop()
//	glog.V(10).Info("Connector worker stopped")
//}

func (helper *Helper) startSignalBusWorker() {
	env := helper.Env()
	glog.V(10).Info("Signal bus worker started")
	env.Services.SignalBus.(*signalbus.PgSignalBus).Start()
}

func (helper *Helper) stopSignalBusWorker() {
	env := helper.Env()
	env.Services.SignalBus.(*signalbus.PgSignalBus).Stop()
	glog.V(10).Info("Signal bus worker stopped")
}

func (helper *Helper) startLeaderElectionWorker() {

	env := helper.Env()
	services := env.Services
	helper.ClusterWorker = workers.NewClusterManager(services.Cluster, services.CloudProviders,
		services.Config, uuid.New().String(), services.KasFleetshardAddonService, services.OsdIdpKeycloak, services.SignalBus)

	var kafkaWorkerList []workers.Worker
	kafkaWorker := kafka_mgrs.NewKafkaManager(services.Kafka, uuid.New().String(), services.Config, services.SignalBus)
	acceptedKafkaManager := kafka_mgrs.NewAcceptedKafkaManager(services.Kafka, uuid.New().String(), services.Config, env.QuotaServiceFactory, services.ClusterPlmtStrategy, env.Services.SignalBus)
	preparingKafkaManager := kafka_mgrs.NewPreparingKafkaManager(services.Kafka, uuid.New().String(), env.Services.SignalBus)
	deletingKafkaManager := kafka_mgrs.NewDeletingKafkaManager(services.Kafka, uuid.New().String(), services.Config, env.QuotaServiceFactory, env.Services.SignalBus)
	provisioningKafkaManager := kafka_mgrs.NewProvisioningKafkaManager(services.Kafka, uuid.New().String(), services.Observatorium, services.Config, env.Services.SignalBus)
	readyKafkaManager := kafka_mgrs.NewReadyKafkaManager(services.Kafka, uuid.New().String(), services.Keycloak, services.Config, env.Services.SignalBus)
	helper.KafkaWorkers = append(kafkaWorkerList, kafkaWorker, acceptedKafkaManager, preparingKafkaManager, deletingKafkaManager, provisioningKafkaManager, readyKafkaManager)

	var workerList []workers.Worker
	workerList = append(workerList, helper.ClusterWorker)
	workerList = append(workerList, helper.KafkaWorkers...)

	// load dependency injected workers.
	var diWorkers []workers.Worker
	if err := env.ServiceContainer.Resolve(&diWorkers); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		panic(err)
	}
	workerList = append(workerList, diWorkers...)

	helper.LeaderEleWorker = workers.NewLeaderElectionManager(workerList, helper.DBFactory)
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

func (helper *Helper) StartKafkaWorker() {
	helper.stopKafkaWorkers()
	helper.startKafkaWorkers()
}

func (helper *Helper) StopKafkaWorker() {
	helper.stopKafkaWorkers()
}

func (helper *Helper) StartClusterWorker() {
	helper.stopClusterWorker()
	helper.startClusterWorker()
}

func (helper *Helper) StopClusterWorker() {
	helper.stopClusterWorker()
}

func (helper *Helper) RestartMetricsServer() {
	helper.stopMetricsServer()
	helper.startMetricsServer()
	glog.V(10).Info("Test metrics server restarted")
}

// Reset metrics. Note this will only reset metrics defined in pkg/metrics
func (helper *Helper) ResetMetrics() {
	metrics.Reset()
}

func (helper *Helper) Reset() {
	glog.Infof("Reseting testing environment")
	env := environments.Environment()
	// Reset the configuration
	env.Config = config.NewApplicationConfig()

	// Re-read command-line configuration into a NEW flagset
	// This new flag set ensures we don't hit conflicts defining the same flag twice
	// Also on reset, we don't care to be re-defining 'v' and other glog flags
	flagset := pflag.NewFlagSet(helper.NewID(), pflag.ContinueOnError)
	err := env.AddFlags(flagset)
	if err != nil {
		glog.Fatalf("Unable to load clients: %s", err.Error())
	}
	pflag.Parse()

	err = env.Initialize()
	if err != nil {
		glog.Fatalf("Unable to reset testing environment: %s", err.Error())
	}
	helper.AppConfig = env.Config
	helper.RestartServer()
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
	helper.Env().Config.Server.JwksURL = jwkURL
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
