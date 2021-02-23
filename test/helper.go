package test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/workers"

	"github.com/bxcodec/faker/v3"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
	"github.com/spf13/pflag"

	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"

	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/environments"
	"gitlab.cee.redhat.com/service/managed-services-api/cmd/managed-services-api/server"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

const (
	jwtKeyFile     = "test/support/jwt_private_key.pem"
	jwtCAFile      = "test/support/jwt_ca.pem"
	jwkKID         = "uhctestkey"
	tokenClaimType = "Bearer"
	tokenExpMin    = 15
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
	KafkaWorker       *workers.KafkaManager
	ClusterWorker     *workers.ClusterManager
	LeaderEleWorker   *workers.LeaderElectionManager
	TimeFunc          TimeFunc
	JWTPrivateKey     *rsa.PrivateKey
	JWTCA             *rsa.PublicKey
	T                 *testing.T
	teardowns         []func()
}

func NewHelper(t *testing.T, server *httptest.Server) *Helper {
	once.Do(func() {
		jwtKey, jwtCA, err := parseJWTKeys()
		if err != nil {
			fmt.Println("Unable to read JWT keys - this may affect tests that make authenticated server requests")
		}

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
		err = env.AddFlags(pflag.CommandLine)
		if err != nil {
			glog.Fatalf("Unable to add environment flags: %s", err.Error())
		}
		if logLevel := os.Getenv("LOGLEVEL"); logLevel != "" {
			glog.Infof("Using custom loglevel: %s", logLevel)
			err = pflag.CommandLine.Set("-v", logLevel)
			if err != nil {
				glog.Warningf("Unable to set custom logLevel: %s", err.Error())
			}
		}
		pflag.Parse()

		err = env.Initialize()
		if err != nil {
			glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
		}

		helper = &Helper{
			AppConfig:     environments.Environment().Config,
			DBFactory:     environments.Environment().DBFactory,
			JWTPrivateKey: jwtKey,
			JWTCA:         jwtCA,
		}

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
	helper.Env().Config.Server.JwkCertURL = jwkURL
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

func (helper *Helper) startKafkaWorker() {
	ocmClient := ocm.NewClient(environments.Environment().Clients.OCM.Connection)
	helper.KafkaWorker = workers.NewKafkaManager(helper.Env().Services.Kafka, helper.Env().Services.Cluster, ocmClient, uuid.New().String(), helper.Env().Services.Keycloak, helper.Env().Services.Observatorium)
	go func() {
		glog.V(10).Info("Test Metrics server started")
		helper.KafkaWorker.Start()
		glog.V(10).Info("Test Metrics server stopped")
	}()
}

func (helper *Helper) stopKafkaWorker() {
	if helper.KafkaWorker == nil {
		return
	}
	helper.KafkaWorker.Stop()
}

func (helper *Helper) startClusterWorker() {
	ocmClient := ocm.NewClient(environments.Environment().Clients.OCM.Connection)

	// start cluster worker
	helper.ClusterWorker = workers.NewClusterManager(helper.Env().Services.Cluster, helper.Env().Services.CloudProviders,
		ocmClient, environments.Environment().Services.Config, uuid.New().String(), &services.KasFleetshardOperatorAddonMock{})
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

func (helper *Helper) startLeaderElectionWorker() {

	ocmClient := ocm.NewClient(environments.Environment().Clients.OCM.Connection)
	helper.ClusterWorker = workers.NewClusterManager(helper.Env().Services.Cluster, helper.Env().Services.CloudProviders,
		ocmClient, environments.Environment().Services.Config, uuid.New().String(), &services.KasFleetshardOperatorAddonMock{})

	ocmClient = ocm.NewClient(environments.Environment().Clients.OCM.Connection)
	helper.KafkaWorker = workers.NewKafkaManager(helper.Env().Services.Kafka, helper.Env().Services.Cluster, ocmClient, uuid.New().String(), helper.Env().Services.Keycloak, helper.Env().Services.Observatorium)

	var workerLst []workers.Worker
	workerLst = append(workerLst, helper.ClusterWorker)
	workerLst = append(workerLst, helper.KafkaWorker)

	helper.LeaderEleWorker = workers.NewLeaderElectionManager(workerLst, helper.DBFactory)
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
	helper.stopKafkaWorker()
	helper.startKafkaWorker()
}

func (helper *Helper) StopKafkaWorker() {
	helper.stopKafkaWorker()
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
	return fmt.Sprintf("%s://%s/api/managed-services-api/v1%s", protocol, helper.AppConfig.Server.BindAddress, path)
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

// NewRandAccount returns a random account that has the control plane team org id as its organisation id
// The org id value is taken from config/allow-list-configuration.yaml
func (helper *Helper) NewRandAccount() *amv1.Account {
	// this value if taken from config/allow-list-configuration.yaml
	orgId := "13640203"
	return helper.NewAccount(helper.NewID(), faker.Name(), faker.Email(), orgId)
}

func (helper *Helper) NewAllowedServiceAccount() *amv1.Account {
	// this value if taken from config/allow-list-configuration.yaml
	allowedSA := "testuser1@example.com"
	return helper.NewAccount(allowedSA, allowedSA, allowedSA, "")
}

func (helper *Helper) NewAccount(username, name, email string, orgId string) *amv1.Account {
	var firstName string
	var lastName string
	names := strings.SplitN(name, " ", 2)
	if len(names) < 2 {
		firstName = name
		lastName = ""
	} else {
		firstName = names[0]
		lastName = names[1]
	}

	builder := amv1.NewAccount().
		Username(username).
		FirstName(firstName).
		LastName(lastName).
		Email(email).
		Organization(amv1.NewOrganization().ExternalID(orgId))

	acct, err := builder.Build()
	if err != nil {
		helper.T.Errorf(fmt.Sprintf("Unable to build account: %s", err))
	}
	return acct
}

func (helper *Helper) NewAuthenticatedContext(account *amv1.Account) context.Context {
	tokenString := helper.CreateJWTString(account)
	return context.WithValue(context.Background(), openapi.ContextAccessToken, tokenString)
}

func (helper *Helper) StartJWKCertServerMock() (teardown func()) {
	jwkURL, teardown = mocks.NewJWKCertServerMock(helper.T, helper.JWTCA, jwkKID)
	helper.Env().Config.Server.JwkCertURL = jwkURL
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

func (helper *Helper) Count(table string) int {
	gorm := helper.DBFactory.New()
	var count int
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
	gorm := helper.DBFactory.New()

	for _, table := range []string{
		"clusters",
		"kafka_requests",
		"migrations",
		"leader_leases",
	} {
		if gorm.HasTable(table) {
			err := gorm.DropTable(table).Error
			if err != nil {
				helper.T.Errorf("error dropping table %s: %v", table, err)
			}
		} else {
			helper.T.Errorf("Unable to drop table %q, it does not exist", table)
		}
	}
}

func (helper *Helper) ResetDB() {
	helper.CleanDB()
	helper.MigrateDB()
}

func (helper *Helper) CreateJWTStringWithAdditionalClaims(account *amv1.Account, moreClaims jwt.MapClaims) string {
	// Use an RH SSO JWT by default since we are phasing RHD out
	claims := jwt.MapClaims{
		"iss":        helper.Env().Config.OCM.TokenURL,
		"username":   account.Username(),
		"first_name": account.FirstName(),
		"last_name":  account.LastName(),
		"typ":        tokenClaimType,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Minute * time.Duration(tokenExpMin)).Unix(),
	}

	org, ok := account.GetOrganization()

	if ok {
		claims["org_id"] = org.ExternalID()
	}

	if account.Email() != "" {
		claims["email"] = account.Email()
	}
	/* TODO the ocm api model needs to be updated to expose this
	if account.ServiceAccount {
		claims["clientId"] = account.Username()
	}
	*/
	for k, v := range moreClaims {
		claims[k] = v
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Set the token header kid to the same value we expect when validating the token
	// The kid is an arbitrary identifier for the key
	// See https://tools.ietf.org/html/rfc7517#section-4.5
	token.Header["kid"] = jwkKID
	token.Header["alg"] = jwt.SigningMethodRS256.Alg()

	// private key and public key taken from http://kjur.github.io/jsjws/tool_jwt.html
	// the go-jwt-middleware pkg we use does the same for their tests
	signedToken, err := token.SignedString(helper.JWTPrivateKey)
	if err != nil {
		helper.T.Errorf("Unable to sign test jwt: %s", err)
		return ""
	}
	return signedToken
}

func (helper *Helper) CreateJWTString(account *amv1.Account) string {
	return helper.CreateJWTStringWithAdditionalClaims(account, nil)
}

func (helper *Helper) CreateJWTStringWithClaim(account *amv1.Account, jwtClaims jwt.MapClaims) string {
	// Use an RH SSO JWT by default since we are phasing RHD out
	claims := jwtClaims
	if account.Email() != "" {
		claims["email"] = account.Email()
	}
	/* TODO the ocm api model needs to be updated to expose this
	if account.ServiceAccount {
		claims["clientId"] = account.Username()
	}
	*/

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Set the token header kid to the same value we expect when validating the token
	// The kid is an arbitrary identifier for the key
	// See https://tools.ietf.org/html/rfc7517#section-4.5
	token.Header["kid"] = jwkKID
	token.Header["alg"] = jwt.SigningMethodRS256.Alg()

	// private key and public key taken from http://kjur.github.io/jsjws/tool_jwt.html
	// the go-jwt-middleware pkg we use does the same for their tests
	signedToken, err := token.SignedString(helper.JWTPrivateKey)
	if err != nil {
		helper.T.Errorf("Unable to sign test jwt: %s", err)
		return ""
	}
	return signedToken
}

func (helper *Helper) CreateJWTToken(account *amv1.Account) *jwt.Token {
	tokenString := helper.CreateJWTString(account)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return helper.JWTCA, nil
	})
	if err != nil {
		helper.T.Errorf("Unable to parse signed jwt: %s", err)
		return nil
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

func parseJWTKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	projectRootDir := config.GetProjectRootDir()
	privateBytes, err := ioutil.ReadFile(filepath.Join(projectRootDir, jwtKeyFile))
	if err != nil {
		err = fmt.Errorf("Unable to read JWT key file %s: %s", jwtKeyFile, err)
		return nil, nil, err
	}
	pubBytes, err := ioutil.ReadFile(filepath.Join(projectRootDir, jwtCAFile))
	if err != nil {
		err = fmt.Errorf("Unable to read JWT ca file %s: %s", jwtKeyFile, err)
		return nil, nil, err
	}

	// Parse keys
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEMWithPassword(privateBytes, "passwd")
	if err != nil {
		err = fmt.Errorf("Unable to parse JWT private key: %s", err)
		return nil, nil, err
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		err = fmt.Errorf("Unable to parse JWT ca: %s", err)
		return nil, nil, err
	}

	return privateKey, pubKey, nil
}
