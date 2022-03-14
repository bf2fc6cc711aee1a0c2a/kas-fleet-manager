package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"github.com/aws/aws-sdk-go/service/route53"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/onsi/gomega"
	goerrors "github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/gorm"

	. "github.com/onsi/gomega"
)

const (
	JwtKeyFile         = "test/support/jwt_private_key.pem"
	JwtCAFile          = "test/support/jwt_ca.pem"
	MaxClusterCapacity = 1000
)

var (
	testKafkaRequestRegion   = "us-east-1"
	testKafkaRequestProvider = "aws"
	testKafkaRequestName     = "test-cluster"
	testClusterID            = "test-cluster-id"
	testID                   = "test"
	testUser                 = "test-user"
	kafkaRequestTableName    = "kafka_requests"
)

// build a test kafka request
func buildKafkaRequest(modifyFn func(kafkaRequest *dbapi.KafkaRequest)) *dbapi.KafkaRequest {
	kafkaRequest := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID:        testID,
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Region:        testKafkaRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testKafkaRequestProvider,
		Name:          testKafkaRequestName,
		MultiAZ:       false,
		Owner:         testUser,
		SizeId:        "x1",
	}
	if modifyFn != nil {
		modifyFn(kafkaRequest)
	}
	return kafkaRequest
}

func buildDataplaneClusterConfig(clusters []config.ManualCluster) *config.DataplaneClusterConfig {
	dataplane := config.NewDataplaneClusterConfig()
	dataplane.ClusterConfig = config.NewClusterConfig(clusters)
	return dataplane
}

func buildManualCluster(kafkaInstanceLimit int, supportedInstanceType, region string) config.ManualCluster {
	return config.ManualCluster{
		Name:                  api.NewID(),
		ClusterId:             api.NewID(),
		CloudProvider:         testKafkaRequestProvider,
		Region:                region,
		MultiAZ:               true,
		Schedulable:           true,
		KafkaInstanceLimit:    kafkaInstanceLimit,
		Status:                api.ClusterReady,
		ProviderType:          api.ClusterProviderOCM,
		SupportedInstanceType: supportedInstanceType,
	}
}

func buildProviderConfiguration(regionName string, standardLimit, evalLimit int, noLimit bool) *config.ProviderConfig {

	instanceTypeLimits := config.InstanceTypeMap{
		"standard": config.InstanceTypeConfig{
			Limit: &standardLimit,
		},
		"eval": config.InstanceTypeConfig{
			Limit: &evalLimit,
		},
	}

	if noLimit {
		instanceTypeLimits = config.InstanceTypeMap{
			"standard": config.InstanceTypeConfig{},
			"eval":     config.InstanceTypeConfig{},
		}
	}

	return &config.ProviderConfig{
		ProvidersConfig: config.ProviderConfiguration{
			SupportedProviders: config.ProviderList{
				{
					Name:    "aws",
					Default: true,
					Regions: config.RegionList{
						{
							Name:                   regionName,
							Default:                true,
							SupportedInstanceTypes: instanceTypeLimits,
						},
					},
				},
			},
		},
	}
}

// This test should act as a "golden" test to describe the general testing approach taken in the service, for people
// onboarding into development of the service.
func Test_kafkaService_Get(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case kafkaService
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to kafkaService.PrepareKafkaRequest
	type args struct {
		ctx context.Context
		id  string
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount(testUser, "", "", "")
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	ctx := context.TODO()
	authenticatedCtx := auth.SetTokenInContext(ctx, jwt)

	// we define tests as list of structs that contain inputs and expected outputs
	// this means we can execute the same logic on each test struct, and makes adding new tests simple as we only need
	// to provide a new struct to the list instead of defining an entirely new test
	tests := []struct {
		// name is just a description of the test
		name   string
		fields fields
		args   args
		// want (there can be more than one) is the outputs that we expect, they can be compared after the test
		// function has been executed
		want *dbapi.KafkaRequest
		// wantErr is similar to want, but instead of testing the actual returned error, we're just testing than any
		// error has been returned
		wantErr bool
		// setupFn will be called before each test and allows mocket setup to be performed
		setupFn func()
	}{
		// below is a single test case, we define each of the fields that we care about from the anonymous test struct
		// above
		{
			name: "error when id is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: context.TODO(),
				id:  "",
			},
			wantErr: true,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				id:  testID,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful output",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				id:  testID,
			},
			want: buildKafkaRequest(nil),
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE id = $1 AND owner = $2`).
					WithArgs(testID, testUser).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
			},
		},
	}
	RegisterTestingT(t)
	// we loop through each test case defined in the list above and start a new test invocation, using the testing
	// t.Run function
	for _, tt := range tests {
		// tt now contains our test case, we can use the 'fields' to construct the struct that we want to test and the
		// 'args' to pass to the function we want to test
		t.Run(tt.name, func(t *testing.T) {
			// invoke any pre-req logic if needed
			if tt.setupFn != nil {
				tt.setupFn()
			}
			// we're testing the kafkaService struct, so use the 'fields' to create one
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			// we're testing the kafkaService.Get function so use the 'args' to provide arguments to the function
			got, err := k.Get(tt.args.ctx, tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// in our test case we used 'want' to define the output api.KafkaRequest that we expect to be returned, we
			// can use Equal function to compare expected and received result
			Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_kafkaService_GetById(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		id string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *dbapi.KafkaRequest
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when id is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: "",
			},
			wantErr: true,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: testID,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful output",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: testID,
			},
			want: buildKafkaRequest(nil),
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.GetById(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_kafkaService_PrepareKafkaRequest(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		keycloakService   sso.KeycloakService
		kafkaConfig       *config.KafkaConfig
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}

	longKafkaName := "long-kafka-name-which-will-be-truncated-since-route-host-names-are-limited-to-63-characters"

	tests := []struct {
		name                    string
		fields                  fields
		args                    args
		setupFn                 func()
		wantErr                 bool
		wantBootstrapServerHost string
	}{
		{
			name: "successful kafka request preparation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
		},
		{
			name: "failed clusterDNS retrieval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorBadRequest, "")
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: true,
		},
		{
			name: "validate BootstrapServerHost truncate",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Name = longKafkaName
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr:                 false,
			wantBootstrapServerHost: fmt.Sprintf("%s-%s.clusterDNS", TruncateString(longKafkaName, truncatedNameLen), testID),
		},
		{
			name: "failed SSO client creation",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "", errors.FailedToCreateSSOClient("failed to create the sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: true,
		},
		{
			name: "failed to create canary service account",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "dsd", nil
					},
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.FailedToCreateSSOClient("failed to create the sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				keycloakService:   tt.fields.keycloakService,
				kafkaConfig:       tt.fields.kafkaConfig,
				awsConfig:         config.NewAWSConfig(),
			}

			if err := k.PrepareKafkaRequest(tt.args.kafkaRequest); (err != nil) != tt.wantErr {
				t.Errorf("PrepareKafkaRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantBootstrapServerHost != "" && tt.args.kafkaRequest.BootstrapServerHost != tt.wantBootstrapServerHost {
				t.Errorf("BootstrapServerHost error. Actual = %v, wantBootstrapServerHost = %v", tt.args.kafkaRequest.BootstrapServerHost, tt.wantBootstrapServerHost)
			}

			if !tt.wantErr && tt.args.kafkaRequest.Namespace == "" {
				t.Errorf("PrepareKafkaRequest() kafkaRequest.Namespace = \"\", want = %v", fmt.Sprintf("kafka-%s", tt.args.kafkaRequest.ID))
			}
		})
	}
}

func Test_kafkaService_RegisterKafkaDeprovisionJob(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		quotaService      QuotaService
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantErrMsg string
		setupFn    func()
	}{
		{
			name: "error when id is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				quotaService: &QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			wantErr: true,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				quotaService: &QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			wantErr:    true,
			wantErrMsg: "KAFKAS-MGMT-9",
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "kafka_requests"`).WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.RegisterKafkaDeprovisionJob(context.TODO(), tt.args.kafkaRequest.ID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErrMsg != "" {
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Bad error message received: '%s'. Expecting to contain %s", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}

func Test_kafkaService_Delete(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		keycloakService   sso.KeycloakService
		kafkaConfig       *config.KafkaConfig
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "successfully deletes a Kafka request when it has not been assigned to an OSD cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successfully deletes a Kafka request and cleans up all of its dependencies",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "fail to delete kafka request: error when deleting sso client",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return errors.FailedToDeleteSSOClient("failed to delete sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			wantErr: true,
		},
		{
			name: "fail to delete kafka request: error when canary service account",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
					kafkaRequest.CanaryServiceAccountClientID = "canary-id"
				}),
			},
			wantErr: true,
		},
		{
			name: "should not delete internal service account when canary serverice account id is empty",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					DeleteServiceAccountInternalFunc: nil,
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.CanaryServiceAccountClientID = ""
				}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				keycloakService:   tt.fields.keycloakService,
				kafkaConfig:       tt.fields.kafkaConfig,
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Delete(tt.args.kafkaRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

var kafkaSupportedInstanceTypesConfig = config.KafkaSupportedInstanceTypesConfig{
	Configuration: config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			{
				Id: "standard",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x1",
						IngressThroughputPerSec:     "30Mi",
						EgressThroughputPerSec:      "30Mi",
						TotalMaxConnections:         1000,
						MaxDataRetentionSize:        "100Gi",
						MaxPartitions:               1000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 100,
						QuotaConsumed:               1,
						QuotaType:                   "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
			{
				Id: "eval",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x2",
						IngressThroughputPerSec:     "60Mi",
						EgressThroughputPerSec:      "60Mi",
						TotalMaxConnections:         2000,
						MaxDataRetentionSize:        "200Gi",
						MaxPartitions:               2000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 200,
						QuotaConsumed:               2,
						QuotaType:                   "rhosak",
						CapacityConsumed:            2,
					},
				},
			},
		},
	},
}

var defaultKafkaConf = config.KafkaConfig{
	KafkaCapacity:          config.KafkaCapacityConfig{},
	Quota:                  config.NewKafkaQuotaConfig(),
	SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
}

func Test_kafkaService_RegisterKafkaJob(t *testing.T) {

	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterService         ClusterService
		quotaService           QuotaService
		kafkaConfig            config.KafkaConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
		providerConfig         *config.ProviderConfig
		clusterPlmtStrategy    ClusterPlacementStrategy
	}

	type errorCheck struct {
		wantErr  bool
		code     errors.ServiceErrorCode
		httpCode int
	}

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}

	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"
	availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
				{
					Version: "2.8.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
				{
					Version: "2.8",
				},
			},
		},
	})

	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	mockCluster := &api.Cluster{
		Meta: api.Meta{
			ID:        testID,
			CreatedAt: time.Now(),
		},
		Region:                   testKafkaRequestRegion,
		ClusterID:                testClusterID,
		CloudProvider:            testKafkaRequestProvider,
		Status:                   api.ClusterReady,
		AvailableStrimziVersions: availableStrimziVersions,
	}

	defaultDataplaneClusterConfig := []config.ManualCluster{buildManualCluster(1, api.AllInstanceTypeSupport.String(), testKafkaRequestRegion)}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		error   errorCheck
	}{
		{
			name: "registering kafka job succeeds",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				kafkaConfig:            defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "registering kafka job succeeds when kafka limit is set to nil",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				kafkaConfig:            defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, 0, 0, true),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "unsuccessful registering kafka job with limit set to zero for standard instance",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				kafkaConfig:            defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, 0, MaxClusterCapacity, false),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "unsuccessful registering kafka job with limit set to zero for eval instance",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, 0, false),
				kafkaConfig:            defaultKafkaConf,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return false, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.EVAL.String()
					kafkaRequest.SizeId = "x2"
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "registering kafka job eval disabled",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig: config.KafkaConfig{
					KafkaCapacity: config.KafkaCapacityConfig{},
					Quota: &config.KafkaQuotaConfig{
						Type:                   api.QuotaManagementListQuotaType.String(),
						AllowEvaluatorInstance: false,
					},
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						// No RHOSAK quota assigned
						return instanceType != types.STANDARD, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.EVAL.String()
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorGeneral,
				httpCode: http.StatusInternalServerError,
			},
		},
		{
			name: "registering kafka too many instances",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, 0, 0, false),
				kafkaConfig:            defaultKafkaConf,
				clusterService:         nil,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND "kafka_requests"."deleted_at" IS NULL`).WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithQuery("INSERT")
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "registering kafka job fails: postgres error",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig:            defaultKafkaConf,
				clusterService:         nil,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND "kafka_requests"."deleted_at" IS NULL`).WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithQuery("INSERT").WithExecException()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorGeneral,
				httpCode: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				connectionFactory:        tt.fields.connectionFactory,
				clusterService:           tt.fields.clusterService,
				kafkaConfig:              &tt.fields.kafkaConfig,
				awsConfig:                config.NewAWSConfig(),
				providerConfig:           tt.fields.providerConfig,
				clusterPlacementStrategy: tt.fields.clusterPlmtStrategy,
				dataplaneClusterConfig:   tt.fields.dataplaneClusterConfig,
				quotaServiceFactory: &QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}

			err := k.RegisterKafkaJob(tt.args.kafkaRequest)

			if (err != nil) != tt.error.wantErr {
				t.Errorf("RegisterKafkaJob() error = %v, wantErr = %v", err, tt.error.wantErr)
			}

			if tt.error.wantErr {
				if err.Code != tt.error.code {
					t.Errorf("RegisterKafkaJob() received error code %v, expected error %v", err.Code, tt.error.code)
				}
				if err.HttpCode != tt.error.httpCode {
					t.Errorf("RegisterKafkaJob() received http code %v, expected %v", err.HttpCode, tt.error.httpCode)
				}
			}
		})
	}
}

func Test_AssignInstanceType(t *testing.T) {
	type fields struct {
		quotaService QuotaService
		kafkaConfig  config.KafkaConfig
	}

	type errorCheck struct {
		wantErr  bool
		code     errors.ServiceErrorCode
		httpCode int
	}

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		error   errorCheck
	}{
		{
			name: "registering kafka job fails: quota error",
			fields: fields{
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(kafka *dbapi.KafkaRequest, instanceType types.KafkaInstanceType) (bool, *errors.ServiceError) {
						return false, errors.InsufficientQuotaError("insufficient quota error")
					},
				},
				kafkaConfig: defaultKafkaConf,
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorInsufficientQuota,
				httpCode: http.StatusForbidden,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				kafkaConfig: &tt.fields.kafkaConfig,
				quotaServiceFactory: &QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}

			_, err := k.AssignInstanceType(tt.args.kafkaRequest)

			if (err != nil) != tt.error.wantErr {
				t.Errorf("AssignInstanceType() error = %v, wantErr = %v", err, tt.error.wantErr)
			}

			if tt.error.wantErr {
				if err.Code != tt.error.code {
					t.Errorf("AssignInstanceType() received error code %v, expected error %v", err.Code, tt.error.code)
				}
				if err.HttpCode != tt.error.httpCode {
					t.Errorf("AssignInstanceType() received http code %v, expected %v", err.HttpCode, tt.error.httpCode)
				}
			}
		})
	}
}

func Test_kafkaService_List(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		ctx      context.Context
		listArgs *services.ListArguments
	}

	type want struct {
		kafkaList  dbapi.KafkaList
		pagingMeta *api.PagingMeta
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount(testUser, "", "", "")
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	ctx := context.TODO()
	authenticatedCtx := auth.SetTokenInContext(ctx, jwt)

	adminCtx := context.TODO()
	adminCtx = auth.SetIsAdminContext(adminCtx, true)
	authenticatedAdminCtx := auth.SetTokenInContext(adminCtx, jwt)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    want
		wantErr bool
		setupFn func(dbapi.KafkaList)
	}{
		{
			name: "success: list with default values for admin context",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedAdminCtx,
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: dbapi.KafkaList{
					&dbapi.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name",
						Status:        "accepted",
						Owner:         testUser,
						Meta: api.Meta{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							DeletedAt: gorm.DeletedAt{Valid: true},
						},
					},
					&dbapi.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name2",
						Status:        "accepted",
						Owner:         testUser,
						Meta: api.Meta{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							DeletedAt: gorm.DeletedAt{Valid: true},
						},
					},
				},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  2,
					Total: 2,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "kafka_requests"`).WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaList)
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "success: list with default values",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: dbapi.KafkaList{
					&dbapi.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name",
						Status:        "accepted",
						Owner:         testUser,
						Meta: api.Meta{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							DeletedAt: gorm.DeletedAt{Valid: true},
						},
					},
					&dbapi.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name2",
						Status:        "accepted",
						Owner:         testUser,
						Meta: api.Meta{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							DeletedAt: gorm.DeletedAt{Valid: true},
						},
					},
				},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  2,
					Total: 2,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "kafka_requests"`).WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaList)
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "success: list with specified size",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 1,
				},
			},
			want: want{
				kafkaList: dbapi.KafkaList{
					&dbapi.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name",
						Status:        "accepted",
						Owner:         testUser,
						Meta: api.Meta{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							DeletedAt: gorm.DeletedAt{Valid: true},
						},
					},
				},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  1,
					Total: 5,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": "5"}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "kafka_requests"`).WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaList)

				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "success: return empty list if no kafka requests available for user",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: dbapi.KafkaList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "kafka_requests"`).WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaList)

				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "fail: user credentials not available in context",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: context.TODO(),
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList:  nil,
				pagingMeta: nil,
			},
			wantErr: true,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery("SELECT count(1) FROM \"kafka_requests\"").WithReply(totalCountResponse)

				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaList)

				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &services.ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: dbapi.KafkaList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: true,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn(tt.want.kafkaList)
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}

			result, pagingMeta, err := k.List(tt.args.ctx, tt.args.listArgs)

			// check errors
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// compare wanted vs actual pagingMeta result
			Expect(pagingMeta).To(Equal(tt.want.pagingMeta))

			// compare wanted vs actual results
			if len(result) != len(tt.want.kafkaList) {
				t.Errorf("kafka.Service.List(): total number of results: got = %d want = %d", len(result), len(tt.want.kafkaList))
			}

			for i, got := range result {
				Expect(got).To(Equal(tt.want.kafkaList[i]))
			}
		})
	}
}

func Test_kafkaService_ListByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		status constants2.KafkaStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*dbapi.KafkaRequest
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail when database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "success",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			want: []*dbapi.KafkaRequest{buildKafkaRequest(nil)},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE status IN ($1)`).
					WithArgs("").
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			got, err := k.ListByStatus(tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_kafkaService_UpdateStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		id     string
		status constants2.KafkaStatus
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		wantExecuted bool
		setupFn      func()
	}{
		{
			name:         "fail when database returns an error",
			wantExecuted: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "refuse execution because cluster in deprovisioning state",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr:      true,
			wantExecuted: false,
			setupFn: func() {
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.Status = constants2.KafkaRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants2.KafkaRequestStatusPreparing,
			},
		},
		{
			name: "success when because cluster in deprovisioning state but status to update is deleted ",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr:      false,
			wantExecuted: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"=$1`)
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.Status = constants2.KafkaRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants2.KafkaRequestStatusDeleting,
			},
		},
		{
			name:         "success",
			wantExecuted: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.Status = constants2.KafkaRequestStatusPreparing.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"=$1`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id: testID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			executed, err := k.UpdateStatus(tt.args.id, tt.args.status)
			if executed != tt.wantExecuted {
				t.Error("kafkaService.UpdateStatus() error = should have refused execution but didn't")
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_kafkaService_Update(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail when database returns an error",
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "success",
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Update(tt.args.kafkaRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_kafkaService_Updates(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail when database returns an error",
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "success",
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Updates(tt.args.kafkaRequest, map[string]interface{}{
				"id":    "idsds",
				"owner": "",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_kafkaService_DeprovisionKafkaForUsers(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		users []string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "should receive error when update fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithError(fmt.Errorf("some update error"))
			},
			args: args{users: []string{"user"}},
		},
		{
			name: "should not receive error when update succeed",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			args:    args{users: []string{"user"}},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			gomega.RegisterTestingT(t)
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeprovisionKafkaForUsers(tt.args.users)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_kafkaService_DeprovisionExpiredKafkas(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	type args struct {
		kafkaAgeInMins int
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail when database update throws an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithError(fmt.Errorf("an update error"))
			},
		},
		{
			name: "success when database does not throw an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"=$1,"updated_at"=$2 WHERE instance_type = $3 AND created_at  <=  $4 AND status NOT IN ($5,$6)`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig:       config.NewKafkaConfig(),
			}
			err := k.DeprovisionExpiredKafkas(tt.args.kafkaAgeInMins)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaService_CountByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status []constants2.KafkaStatus
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		want      []KafkaStatusCount
		setupFunc func()
	}{
		{
			name:   "should return the counts of Kafkas in different status",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []constants2.KafkaStatus{constants2.KafkaRequestStatusAccepted, constants2.KafkaRequestStatusReady, constants2.KafkaRequestStatusProvisioning},
			},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"status": "accepted",
						"count":  2,
					},
					{
						"status": "ready",
						"count":  1,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT status as Status, count(1) as Count FROM "kafka_requests" WHERE status in ($1,$2,$3)`).
					WithArgs(constants2.KafkaRequestStatusAccepted.String(), constants2.KafkaRequestStatusReady.String(), constants2.KafkaRequestStatusProvisioning.String()).
					WithReply(counters)
			},
			want: []KafkaStatusCount{{
				Status: constants2.KafkaRequestStatusAccepted,
				Count:  2,
			}, {
				Status: constants2.KafkaRequestStatusReady,
				Count:  1,
			}, {
				Status: constants2.KafkaRequestStatusProvisioning,
				Count:  0,
			}},
		},
		{
			name:   "should return error",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []constants2.KafkaStatus{constants2.KafkaRequestStatusAccepted, constants2.KafkaRequestStatusReady},
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
			},
			want: nil,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			status, err := k.CountByStatus(tt.args.status)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for CountByStatus: %v", err)
			}
			Expect(status).To(Equal(tt.want))
		})
	}
}

func TestKafkaService_CountByRegionAndInstanceType(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		setupFunc func()
	}{
		{
			name:    "should return the counts of Kafkas per region and instance type",
			fields:  fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"region":        "us-east-1",
						"instance_type": "standard",
						"cluster_id":    testClusterID,
						"Count":         1,
					},
					{
						"region":        "eu-west-1",
						"instance_type": "eval",
						"cluster_id":    testClusterID,
						"Count":         1,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT FROM "kafka_requests" WHERE "kafka_requests"."deleted_at" IS NULL`).
					WithReply(counters)
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			_, err := k.CountByRegionAndInstanceType()
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for CountByRegionAndInstanceType: %v", err)
			}
		})
	}
}

func TestKafkaService_ChangeKafkaCNAMErecords(t *testing.T) {
	type fields struct {
		awsClient aws.Client
	}

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
		action       KafkaRoutesAction
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should create CNAMEs for kafka",
			fields: fields{
				awsClient: &aws.ClientMock{
					ChangeResourceRecordSetsFunc: func(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
						if len(recordChangeBatch.Changes) != 1 {
							return nil, goerrors.Errorf("number of record changes should be 1")
						}
						if *recordChangeBatch.Changes[0].Action != "CREATE" {
							return nil, goerrors.Errorf("the action of the record change is not CREATE")
						}
						return nil, nil
					},
					ListHostedZonesByNameInputFunc: func(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "test-kafka-id",
					},
					Name:   "test-kafka-cname",
					Routes: []byte("[{\"domain\": \"test-kafka-id.example.com\", \"router\": \"test-kafka-id.rhcloud.com\"}]"),
					Region: testKafkaRequestRegion,
				},
				action: KafkaRoutesActionCreate,
			},
		},
		{
			name: "should delete CNAMEs for kafka",
			fields: fields{
				awsClient: &aws.ClientMock{
					ChangeResourceRecordSetsFunc: func(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
						if len(recordChangeBatch.Changes) != 1 {
							return nil, goerrors.Errorf("number of record changes should be 1")
						}
						if *recordChangeBatch.Changes[0].Action != "DELETE" {
							return nil, goerrors.Errorf("the action of the record change is not DELETE")
						}
						return nil, nil
					},
					ListHostedZonesByNameInputFunc: func(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "test-kafka-id",
					},
					Name:   "test-kafka-cname",
					Routes: []byte("[{\"domain\": \"test-kafka-id.example.com\", \"router\": \"test-kafka-id.rhcloud.com\"}]"),
					Region: testKafkaRequestRegion,
				},
				action: KafkaRoutesActionDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kafkaService := &kafkaService{
				awsClientFactory: aws.NewMockClientFactory(tt.fields.awsClient),
				awsConfig: &config.AWSConfig{
					Route53AccessKey:       "test-route-53-key",
					Route53SecretAccessKey: "test-route-53-secret-key",
				},
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "rhcloud.com",
				},
			}

			_, err := kafkaService.ChangeKafkaCNAMErecords(tt.args.kafkaRequest, tt.args.action)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error for ChangeKafkaCNAMErecords %v", err)
			}
		})
	}

}

func TestKafkaService_ListComponentVersions(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		want      []KafkaComponentVersions
		setupFunc func()
	}{
		{
			name:    "should return the component versions for Kafka",
			fields:  fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			wantErr: false,
			setupFunc: func() {
				versions := []map[string]interface{}{
					{
						"id":                        "1",
						"cluster_id":                "cluster1",
						"desired_strimzi_version":   "1.0.1",
						"actual_strimzi_version":    "1.0.0",
						"strimzi_upgrading":         true,
						"desired_kafka_version":     "2.0.1",
						"actual_kafka_version":      "2.0.0",
						"kafka_upgrading":           false,
						"desired_kafka_ibp_version": "2.0",
						"actual_kafka_ibp_version":  "2.0",
						"kafka_ibp_upgrading":       false,
					},
					{
						"id":                        "2",
						"cluster_id":                "cluster2",
						"desired_strimzi_version":   "1.0.1",
						"actual_strimzi_version":    "1.0.0",
						"strimzi_upgrading":         false,
						"desired_kafka_version":     "2.0.1",
						"actual_kafka_version":      "2.0.0",
						"kafka_upgrading":           false,
						"desired_kafka_ibp_version": "2.2",
						"actual_kafka_ibp_version":  "2.1",
						"kafka_ibp_upgrading":       true,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT "id","cluster_id","desired_strimzi_version","actual_strimzi_version","strimzi_upgrading","desired_kafka_version","actual_kafka_version","kafka_upgrading","desired_kafka_ibp_version","actual_kafka_ibp_version","kafka_ibp_upgrading"`).
					WithReply(versions)
			},
			want: []KafkaComponentVersions{{
				ID:                     "1",
				ClusterID:              "cluster1",
				DesiredStrimziVersion:  "1.0.1",
				ActualStrimziVersion:   "1.0.0",
				StrimziUpgrading:       true,
				DesiredKafkaVersion:    "2.0.1",
				ActualKafkaVersion:     "2.0.0",
				KafkaUpgrading:         false,
				DesiredKafkaIBPVersion: "2.0",
				ActualKafkaIBPVersion:  "2.0",
				KafkaIBPUpgrading:      false,
			}, {
				ID:                     "2",
				ClusterID:              "cluster2",
				DesiredStrimziVersion:  "1.0.1",
				ActualStrimziVersion:   "1.0.0",
				StrimziUpgrading:       false,
				DesiredKafkaVersion:    "2.0.1",
				ActualKafkaVersion:     "2.0.0",
				KafkaUpgrading:         false,
				DesiredKafkaIBPVersion: "2.2",
				ActualKafkaIBPVersion:  "2.1",
				KafkaIBPUpgrading:      true,
			}},
		},
		{
			name:    "should return error",
			fields:  fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
			},
			want: nil,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			result, err := k.ListComponentVersions()
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for ListComponentVersions: %v", err)
			}
			Expect(result).To(Equal(tt.want))
		})
	}
}
