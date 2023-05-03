package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafkatlscertmgmt"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	managedkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/onsi/gomega"
	goerrors "github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func buildDataplaneClusterConfigWithAutoscalingOn() *config.DataplaneClusterConfig {
	dataplane := config.NewDataplaneClusterConfig()
	dataplane.DataPlaneClusterScalingType = config.AutoScaling
	dataplane.ClusterConfig = config.NewClusterConfig(config.ClusterList{})
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

func buildProviderConfiguration(regionName string, standardLimit, developerLimit int, noLimit bool) *config.ProviderConfig {

	instanceTypeLimits := config.InstanceTypeMap{
		"standard": config.InstanceTypeConfig{
			Limit: &standardLimit,
		},
		"developer": config.InstanceTypeConfig{
			Limit: &developerLimit,
		},
	}

	if noLimit {
		instanceTypeLimits = config.InstanceTypeMap{
			"standard":  config.InstanceTypeConfig{},
			"developer": config.InstanceTypeConfig{},
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

var testSupportedKafkaBillingModelsStandard = []config.KafkaBillingModel{
	config.KafkaBillingModel{
		ID:          "standard",
		AMSResource: "rhosak",
		AMSProduct:  "RHOSAK",
		AMSBillingModels: []string{
			"standard",
		},
	},
	config.KafkaBillingModel{
		ID:          "marketplace",
		AMSResource: "rhosak",
		AMSProduct:  "RHOSAK",
		AMSBillingModels: []string{
			"marketplace",
			"marketplace-rhm",
			"marketplace-aws",
		},
	},
	config.KafkaBillingModel{
		ID:          "eval",
		AMSResource: "rhosak",
		AMSProduct:  "RHOSAKEval",
		AMSBillingModels: []string{
			"standard",
		},
	},
}

var testSupportedKafkaBillingModelsDeveloper = []config.KafkaBillingModel{
	config.KafkaBillingModel{
		ID:          "standard",
		AMSResource: "rhosak",
		AMSProduct:  "RHOSAKTrial",
		AMSBillingModels: []string{
			"standard",
		},
	},
}

var supportedKafkaSizeStandard = []config.KafkaInstanceSize{
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
		DeprecatedQuotaType:         "rhosak",
		CapacityConsumed:            1,
		MaxMessageSize:              "1Mi",
		MinInSyncReplicas:           2,
		ReplicationFactor:           3,
	},
}

var supportedKafkaSizeDeveloper = []config.KafkaInstanceSize{
	{
		Id:                          "x1",
		IngressThroughputPerSec:     "60Mi",
		EgressThroughputPerSec:      "60Mi",
		TotalMaxConnections:         2000,
		MaxDataRetentionSize:        "200Gi",
		MaxPartitions:               2000,
		MaxDataRetentionPeriod:      "P14D",
		MaxConnectionAttemptsPerSec: 200,
		QuotaConsumed:               2,
		DeprecatedQuotaType:         "rhosak",
		CapacityConsumed:            2,
		MaxMessageSize:              "1Mi",
		MinInSyncReplicas:           1,
		ReplicationFactor:           1,
		LifespanSeconds:             &[]int{172800}[0],
	},
}

var kafkaSupportedInstanceTypesConfig = config.KafkaSupportedInstanceTypesConfig{
	Configuration: config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			{
				Id:                     "standard",
				DisplayName:            "Standard",
				SupportedBillingModels: testSupportedKafkaBillingModelsStandard,
				Sizes:                  supportedKafkaSizeStandard,
			},
			{
				Id:                     "developer",
				DisplayName:            "Trial",
				SupportedBillingModels: testSupportedKafkaBillingModelsDeveloper,
				Sizes:                  supportedKafkaSizeDeveloper,
			},
		},
	},
}

var defaultKafkaConf = config.KafkaConfig{
	Quota:                  config.NewKafkaQuotaConfig(),
	SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
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
			name: "error when kafka id is undefined",
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	// we loop through each test case defined in the list above and start a new test invocation, using the testing
	// t.Run function
	for _, testcase := range tests {
		tt := testcase

		// tt now contains our test case, we can use the 'fields' to construct the struct that we want to test and the
		// 'args' to pass to the function we want to test
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_GetByID(t *testing.T) {
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
			name: "error when kafka id is undefined",
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.GetByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_PrepareKafkaRequest(t *testing.T) {
	type fields struct {
		connectionFactory                    *db.ConnectionFactory
		clusterService                       ClusterService
		keycloakService                      sso.KeycloakService
		kafkaConfig                          *config.KafkaConfig
		kafkaTLSCertificateManagementService kafkatlscertmgmt.KafkaTLSCertificateManagementService
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
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				}),
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
						return "", errors.GeneralError("test")
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Name = longKafkaName
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
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
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							KafkaRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnKafka: true,
						}
					},
					CreateServiceAccountInternalFunc: func(request sso.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.FailedToCreateSSOClient("failed to create the sso client")
					},
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: true,
		},
		{
			name: "failed to create canary service account",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
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
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: true,
		},
		{
			name: "should not return an error when automatic certificate management has not been started yet i.e the certificate refs are not present for the given kafka",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: nil, // should never be called
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc:                    nil,
					CreateServiceAccountInternalFunc: nil,
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = ""
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = ""
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				}),
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
		},
		{
			name: "should not return an error when automatic certificate management is enabled but the Kafka certificate doesn't exist",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: nil, // should never be called
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc:                    nil,
					CreateServiceAccountInternalFunc: nil,
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					GetCertificateFunc: func(ctx context.Context, request kafkatlscertmgmt.GetCertificateRequest) (kafkatlscertmgmt.Certificate, error) {
						return kafkatlscertmgmt.Certificate{}, fs.ErrNotExist
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
					kafkaRequest.KafkasRoutesBaseDomainName = "some-base-domain"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "some-key-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "some-crt-ref"
				}),
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
		},
		{
			name: "should return an error when automatic certificate management is enabled but there is an error which is different than not exist error returned when retrieving the certificate",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: nil, // should never be called
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc:                    nil,
					CreateServiceAccountInternalFunc: nil,
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					GetCertificateFunc: func(ctx context.Context, request kafkatlscertmgmt.GetCertificateRequest) (kafkatlscertmgmt.Certificate, error) {
						return kafkatlscertmgmt.Certificate{}, fmt.Errorf("some error")
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
					kafkaRequest.KafkasRoutesBaseDomainName = "some-base-domain"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "some-key-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "some-crt-ref"
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				connectionFactory:                    tt.fields.connectionFactory,
				clusterService:                       tt.fields.clusterService,
				keycloakService:                      tt.fields.keycloakService,
				kafkaConfig:                          tt.fields.kafkaConfig,
				awsConfig:                            config.NewAWSConfig(),
				kafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "error when id is an empty string",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
				kafkaRequest.ID = ""
			}),
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

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
					t.Errorf("bad error message received: '%s'. Expecting to contain %s", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}

func Test_kafkaService_Delete(t *testing.T) {
	type fields struct {
		connectionFactory                    *db.ConnectionFactory
		clusterService                       ClusterService
		keycloakService                      sso.KeycloakService
		kafkaConfig                          *config.KafkaConfig
		kafkaTLSCertificateManagementService kafkatlscertmgmt.KafkaTLSCertificateManagementService
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
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: nil, // should never be called
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafkatlscertmgmt.CertificateRevocationReason) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
					kafkaRequest.ClusterID = ""
					kafkaRequest.KafkasRoutesBaseDomainName = "bade domain"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "crt-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "key-ref"
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
			name: "fail to delete kafka request: error when canary service account",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
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
		{
			name: "successfully deletes a Kafka request when canary service account is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{
							Code: errors.ErrorServiceAccountNotFound,
						}
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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successfully deletes a Kafka request without revoking the shared certificate", // a certificate is considerd shared if kafkaConfig.KafkaDomainName == kafkaRequest.KafkasRoutesBaseDomainName
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{
							Code: errors.ErrorServiceAccountNotFound,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "kafka-base-domain.org",
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: nil, // should never be called
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
					kafkaRequest.CanaryServiceAccountClientID = "canary-id"
					kafkaRequest.KafkasRoutesBaseDomainName = "kafka-base-domain.org"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "crt-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "key-ref"
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successfully deletes a Kafka request when kafka has certificates info generated and the Kafka is assigned to a data plane cluster i.e cluster_id set",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{
							Code: errors.ErrorServiceAccountNotFound,
						}
					},
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafkatlscertmgmt.CertificateRevocationReason) error {
						return nil
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
					kafkaRequest.ClusterID = "some-cluster-id"
					kafkaRequest.CanaryServiceAccountClientID = "canary-id"
					kafkaRequest.KafkasRoutesBaseDomainName = "bade domain"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "crt-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "key-ref"
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "returns an error when certificate revocation fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{
							Code: errors.ErrorServiceAccountNotFound,
						}
					},
				},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					RevokeCertificateFunc: func(ctx context.Context, domain string, reason kafkatlscertmgmt.CertificateRevocationReason) error {
						return fmt.Errorf("cert revocation failed")
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ID = testID
					kafkaRequest.CanaryServiceAccountClientID = "canary-id"
					kafkaRequest.KafkasRoutesBaseDomainName = "bade domain"
					kafkaRequest.KafkasRoutesBaseDomainTLSCrtRef = "crt-ref"
					kafkaRequest.KafkasRoutesBaseDomainTLSKeyRef = "key-ref"
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory:                    tt.fields.connectionFactory,
				clusterService:                       tt.fields.clusterService,
				keycloakService:                      tt.fields.keycloakService,
				kafkaConfig:                          tt.fields.kafkaConfig,
				awsConfig:                            config.NewAWSConfig(),
				kafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
			}
			err := k.Delete(tt.args.kafkaRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
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
	nowTime := time.Now()

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func(connectionFactory *db.ConnectionFactory)
		error   errorCheck
		// verifyKafkaUpdatedContentsFunc contains a set of test assertions
		// checking the kafka request result contents. If null then it is not
		// run
		verifyKafkaUpdatedContentsFunc func(g *gomega.WithT, kafkaRequest *dbapi.KafkaRequest)
	}{
		{
			name: "registering kafka job succeeds",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				kafkaConfig:            defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				connectionFactory.DB.NowFunc = func() time.Time {
					return nowTime
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr: false,
			},
			verifyKafkaUpdatedContentsFunc: func(g *gomega.WithT, kafkaRequest *dbapi.KafkaRequest) {
				g.Expect(kafkaRequest.ExpiresAt.Valid).NotTo((gomega.BeTrue()))
				g.Expect(kafkaRequest.ExpiresAt.Time).To(gomega.BeZero())
				g.Expect(kafkaRequest.CreatedAt).To(gomega.Equal(nowTime))
				g.Expect(kafkaRequest.UpdatedAt).To(gomega.Equal(nowTime))
			},
		},
		{
			name: "registering kafka job succeeds with developer",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				kafkaConfig:            defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(username, externalId string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.DEVELOPER.String()
					kafkaRequest.SizeId = "x1"
					kafkaRequest.Owner = testUser
					kafkaRequest.OrganisationId = "org-id"
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				connectionFactory.DB.NowFunc = func() time.Time {
					return nowTime
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "developer", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.ID = ""
						kafkaRequest.InstanceType = types.DEVELOPER.String()
						kafkaRequest.SizeId = "x1"
						kafkaRequest.Owner = testUser
						kafkaRequest.OrganisationId = "org-id"
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQuery(``)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()

			},
			error: errorCheck{
				wantErr: false,
			},
			verifyKafkaUpdatedContentsFunc: func(g *gomega.WithT, kafkaRequest *dbapi.KafkaRequest) {
				g.Expect(kafkaRequest.ExpiresAt.Valid).To((gomega.BeTrue()))
				g.Expect(kafkaRequest.ExpiresAt.Time).NotTo(gomega.BeZero())
				calculatedExpiresAt := nowTime.Add(time.Duration(*supportedKafkaSizeDeveloper[0].LifespanSeconds) * time.Second)
				g.Expect(kafkaRequest.ExpiresAt.Time).To(gomega.Equal(calculatedExpiresAt))
				g.Expect(kafkaRequest.CreatedAt).To(gomega.Equal(nowTime))
				g.Expect(kafkaRequest.UpdatedAt).To(gomega.Equal(nowTime))
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
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				providerConfig: buildProviderConfiguration(testKafkaRequestRegion, 0, 0, true),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
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
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
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
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "unsuccessful registering kafka job with limit set to zero for developer instance",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, 0, false),
				kafkaConfig:            defaultKafkaConf,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return false, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.InstanceType = types.DEVELOPER.String()
					kafkaRequest.SizeId = "x1"
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "should register kafka job successfully when developer instances count for the user is less than max-allowed-developer-instances",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                         api.QuotaManagementListQuotaType.String(),
						AllowDeveloperInstance:       true,
						MaxAllowedDeveloperInstances: 1,
					},
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.DEVELOPER.String()
					kafkaRequest.SizeId = "x1"
					kafkaRequest.Owner = testUser
					kafkaRequest.OrganisationId = "org-id"
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				totalCountResponse := []map[string]interface{}{{"count": 0}}

				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2 AND organisation_id = $3 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), testUser, "org-id").
					WithReply(totalCountResponse)
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "developer", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.ID = ""
						kafkaRequest.InstanceType = types.DEVELOPER.String()
						kafkaRequest.SizeId = "x1"
						kafkaRequest.Owner = testUser
						kafkaRequest.OrganisationId = "org-id"
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "should fail to register kafka job when developer kafka instances count for the user has reached the max-allowed-developer-instances limit",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                         api.QuotaManagementListQuotaType.String(),
						AllowDeveloperInstance:       true,
						MaxAllowedDeveloperInstances: 2,
					},
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.DEVELOPER.String()
					kafkaRequest.SizeId = "x1"
					kafkaRequest.Owner = testUser
					kafkaRequest.OrganisationId = "org-id"
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "developer", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.ID = ""
						kafkaRequest.ClusterID = ""
						kafkaRequest.InstanceType = types.DEVELOPER.String()
						kafkaRequest.SizeId = "x1"
						kafkaRequest.Owner = testUser
						kafkaRequest.OrganisationId = "org-id"
					})))

				totalCountResponse := []map[string]interface{}{{"count": 2}}

				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2 AND organisation_id = $3 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), testUser, "org-id").
					WithReply(totalCountResponse)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "registering kafka job unsuccessful when wrong plan is selected",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterService:         nil,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, MaxClusterCapacity, MaxClusterCapacity, false),
				kafkaConfig: config.KafkaConfig{
					Quota: &config.KafkaQuotaConfig{
						Type:                   api.QuotaManagementListQuotaType.String(),
						AllowDeveloperInstance: false,
					},
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						// No RHOSAK quota assigned
						return instanceType != types.STANDARD, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.DEVELOPER.String()
					kafkaRequest.SizeId = "x2"
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorInstancePlanNotSupported,
				httpCode: http.StatusBadRequest,
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset()
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
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
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
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "registering kafka successfully when we are on dynamic scaling mode, region limits have not been reached but there is no cluster found to assign the Kafka to",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: buildDataplaneClusterConfigWithAutoscalingOn(),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, 2, 0, false),
				kafkaConfig:            defaultKafkaConf,
				clusterService:         nil,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = ""
					kafkaRequest.InstanceType = types.STANDARD.String()
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "registering kafka fails when we are on dynamic scaling mode and region limits have been reached",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: buildDataplaneClusterConfigWithAutoscalingOn(),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, 1, 0, false),
				kafkaConfig:            defaultKafkaConf,
				clusterService:         nil,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
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
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
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
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
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
			setupFn: func(connectionFactory *db.ConnectionFactory) {
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
		{
			name: "should fail when cluster for enterprise kafka request cannot be found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to find cluster")
					},
				},
				kafkaConfig: defaultKafkaConf,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, 0, 0, true),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = "some-cluster-id"
					kafkaRequest.InstanceType = types.STANDARD.String()
					kafkaRequest.DesiredKafkaBillingModel = constants.BillingModelEnterprise.String()
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`).WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "successfully registers an enterprise kafka",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaConfig:       defaultKafkaConf,
				clusterPlmtStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID: "some-cluster-id",
						}, nil
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(kafka *dbapi.KafkaRequest) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration(testKafkaRequestRegion, 0, 0, true),
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					kafkaRequest.ID = ""
					kafkaRequest.ClusterID = "some-cluster-id"
					kafkaRequest.InstanceType = types.STANDARD.String()
					kafkaRequest.DesiredKafkaBillingModel = constants.BillingModelEnterprise.String()
				}),
			},
			setupFn: func(connectionFactory *db.ConnectionFactory) {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs("us-east-1", "aws", "standard", constants.BillingModelEnterprise.String()).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
						kafkaRequest.InstanceType = types.STANDARD.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`INSERT INTO "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			error: errorCheck{
				wantErr: false,
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn(tt.fields.connectionFactory)
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

			if tt.verifyKafkaUpdatedContentsFunc != nil {
				tt.verifyKafkaUpdatedContentsFunc(g, tt.args.kafkaRequest)
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
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(owner string, organisationID string, instanceType types.KafkaInstanceType, billingModel config.KafkaBillingModel) (bool, *errors.ServiceError) {
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
	for _, testcase := range tests {
		tt := testcase

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

			_, err := k.AssignInstanceType(tt.args.kafkaRequest.Owner, tt.args.kafkaRequest.OrganisationId)

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
				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests"`).WithReply(totalCountResponse)

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
				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests"`).WithReply(totalCountResponse)

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
				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests"`).WithReply(totalCountResponse)

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
				mocket.Catcher.NewMock().WithQuery(`SELECT count(*) FROM "kafka_requests"`).WithReply(totalCountResponse)

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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(pagingMeta).To(gomega.Equal(tt.want.pagingMeta))

			// compare wanted vs actual results
			if len(result) != len(tt.want.kafkaList) {
				t.Errorf("kafka.Service.List(): total number of results: got = %d want = %d", len(result), len(tt.want.kafkaList))
			}

			for i, got := range result {
				g.Expect(got).To(gomega.Equal(tt.want.kafkaList[i]))
			}
		})
	}
}

func Test_kafkaService_ListAll(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    dbapi.KafkaList
		wantErr bool
		setupFn func(dbapi.KafkaList)
	}{
		{
			name: "success list all kafkas",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			want: dbapi.KafkaList{
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
			wantErr: false,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset()

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
			want:    dbapi.KafkaList{},
			wantErr: true,
			setupFn: func(kafkaList dbapi.KafkaList) {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn(tt.want)
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}

			result, err := k.ListAll()

			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			g.Expect(len(result)).To(gomega.Equal(len(tt.want)))

			for i, got := range result {
				g.Expect(got).To(gomega.Equal(tt.want[i]))
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
		status constants.KafkaStatus
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(got).To(gomega.Equal(tt.want))
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
		status constants.KafkaStatus
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
						kafkaRequest.Status = constants.KafkaRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants.KafkaRequestStatusPreparing,
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
						kafkaRequest.Status = constants.KafkaRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants.KafkaRequestStatusDeleting,
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
						kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"=$1`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id: testID,
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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
	for _, testcase := range tests {
		tt := testcase

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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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
	for _, testcase := range tests {
		tt := testcase

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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeprovisionKafkaForUsers(tt.args.users)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_kafkaService_DeprovisionExpiredKafkas(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		// testDBNowFunc is used by the DB to obtain the current time.
		// The current time is used by the DB to set the created_at and updated_at fields.
		// We provide customizable method to return the time to make tests return arbitrary
		// created_at and updated_at on DB side
		testDBNowFunc func() time.Time
	}

	const instanceType = "type1"
	const instanceSize = "size4"
	expiredTime := sql.NullTime{Time: time.Now().Add(-(2 * time.Hour)), Valid: true}
	unexpiredTime := sql.NullTime{Time: expiredTime.Time.Add(48 * time.Hour), Valid: true}

	nowTime := time.Now()
	dbTime := nowTime.Add(300 * time.Microsecond)

	tests := []struct {
		name    string
		fields  fields
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
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "kafka_requests" WHERE status NOT IN ($2,$3) AND expires_at IS NOT NULL`).WithReply([]map[string]interface{}{{"id": "kafkainstance1", "instance_type": instanceType, "size_id": instanceSize}})
				mocket.Catcher.NewMock().WithQuery(`UPDATE "kafka_requests" SET "status"=$1,"updated_at"=$2 WHERE id IN ($3)`).WithError(fmt.Errorf("an update error"))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "when a kafka instance has expired it marks it as deprovisioned successfully",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				testDBNowFunc: func() time.Time {
					return dbTime
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE status NOT IN ($1,$2) AND expires_at IS NOT NULL`).
					WithReply([]map[string]interface{}{{"id": "kafkainstance1", "instance_type": instanceType, "size_id": instanceSize, "expires_at": &expiredTime}})
				mocket.Catcher.NewMock().
					WithArgs(constants.KafkaRequestStatusDeprovision.String(), dbTime, "kafkainstance1").
					WithQuery(`UPDATE "kafka_requests" SET "status"=$1,"updated_at"=$2 WHERE id IN ($3)`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "when a kafka instance has no expiration set it succeeds and it does not mark it as deprovisioned",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				testDBNowFunc: func() time.Time {
					return dbTime
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE status NOT IN ($1,$2) AND expires_at IS NOT NULL`).
					WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().
					WithQuery(`UPDATE "kafka_requests"`).WithExecException()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "when a kafka instance has expiration set but it is not expired it does not mark it as deprovisioned",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				testDBNowFunc: func() time.Time {
					return dbTime
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE status NOT IN ($1,$2) AND expires_at IS NOT NULL`).
					WithReply([]map[string]interface{}{{"id": "kafkainstance1", "instance_type": instanceType, "size_id": instanceSize, "expires_at": &unexpiredTime}})
				mocket.Catcher.NewMock().
					WithQuery(`UPDATE "kafka_requests"`).WithExecException()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			if tt.fields.testDBNowFunc != nil {
				tt.fields.connectionFactory.DB.NowFunc = tt.fields.testDBNowFunc
			}
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig:       config.NewKafkaConfig(),
			}
			k.kafkaConfig.SupportedInstanceTypes.Configuration = config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					{
						Id: instanceType,
						Sizes: []config.KafkaInstanceSize{
							{Id: "size1"},
							{Id: instanceSize},
						},
					},
				},
			}
			err := k.DeprovisionExpiredKafkas()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_KafkaService_CountByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status []constants.KafkaStatus
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
				status: []constants.KafkaStatus{constants.KafkaRequestStatusAccepted, constants.KafkaRequestStatusReady, constants.KafkaRequestStatusProvisioning},
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
					WithArgs(constants.KafkaRequestStatusAccepted.String(), constants.KafkaRequestStatusReady.String(), constants.KafkaRequestStatusProvisioning.String()).
					WithReply(counters)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: []KafkaStatusCount{{
				Status: constants.KafkaRequestStatusAccepted,
				Count:  2,
			}, {
				Status: constants.KafkaRequestStatusReady,
				Count:  1,
			}, {
				Status: constants.KafkaRequestStatusProvisioning,
				Count:  0,
			}},
		},
		{
			name:   "should return error",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []constants.KafkaStatus{constants.KafkaRequestStatusAccepted, constants.KafkaRequestStatusReady},
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(status).To(gomega.Equal(tt.want))
		})
	}
}

func Test_KafkaService_ChangeKafkaCNAMErecords(t *testing.T) {
	type fields struct {
		awsClient aws.AWSClient
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
				awsClient: &aws.AWSClientMock{
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
					Name:          "test-kafka-cname",
					Routes:        []byte("[{\"domain\": \"test-kafka-id.example.com\", \"router\": \"test-kafka-id.rhcloud.com\"}]"),
					Region:        testKafkaRequestRegion,
					CloudProvider: cloudproviders.AWS.String(),
				},
				action: KafkaRoutesActionCreate,
			},
		},
		{
			name: "should delete CNAMEs for kafka",
			fields: fields{
				awsClient: &aws.AWSClientMock{
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
					Name:          "test-kafka-cname",
					Routes:        []byte("[{\"domain\": \"test-kafka-id.example.com\", \"router\": \"test-kafka-id.rhcloud.com\"}]"),
					Region:        testKafkaRequestRegion,
					CloudProvider: cloudproviders.AWS.String(),
				},
				action: KafkaRoutesActionDelete,
			},
		},
		{
			name: "should return error if it fails to get routes",
			fields: fields{
				awsClient: &aws.AWSClientMock{
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
					Region: testKafkaRequestRegion,
				},
				action: KafkaRoutesActionCreate,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			awsConfig := &config.AWSConfig{}
			awsConfig.Route53.AccessKey = "test-route-53-key"
			awsConfig.Route53.SecretAccessKey = "test-route-53-secret-key"

			kafkaService := &kafkaService{
				awsClientFactory: aws.NewMockClientFactory(tt.fields.awsClient),
				awsConfig:        awsConfig,
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

func Test_KafkaService_ListComponentVersions(t *testing.T) {
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(result).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_GetAvailableSizesInRegion(t *testing.T) {
	type fields struct {
		connectionFactory        *db.ConnectionFactory
		kafkaConfig              *config.KafkaConfig
		dataplaneClusterConfig   *config.DataplaneClusterConfig
		providerConfig           *config.ProviderConfig
		clusterPlacementStrategy ClusterPlacementStrategy
	}
	type args struct {
		criteria *FindClusterCriteria
	}

	defaultCluster := buildManualCluster(1, api.AllInstanceTypeSupport.String(), testKafkaRequestRegion)
	defaultDataplaneClusterConfig := []config.ManualCluster{defaultCluster}

	dynamicScalingEnabledDataplaneClusterConfig := config.NewDataplaneClusterConfig()
	dynamicScalingEnabledDataplaneClusterConfig.DataPlaneClusterScalingType = config.AutoScaling
	testCriteria := &FindClusterCriteria{
		Provider:              defaultCluster.CloudProvider,
		Region:                defaultCluster.Region,
		MultiAZ:               defaultCluster.MultiAZ,
		SupportedInstanceType: "standard",
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		result      []string
		wantErr     bool
		expectedErr *errors.ServiceError
		setupFn     func()
	}{
		{
			name: "when dynamic scaling is disabled, should return all available sizes in region if capacity and limit have not been reached",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				kafkaConfig:            &defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration("us-east-1", 1000, 1000, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mocks.BuildCluster(nil), nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			result:  []string{"x1"},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "when dynamic scaling is enabled, should return all available sizes in region if limit has not been reached even when there is no cluster capacity",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: api.StandardTypeSupport.String(),
									Sizes: []config.KafkaInstanceSize{
										config.KafkaInstanceSize{
											Id:               "x1",
											CapacityConsumed: 1,
										},
										config.KafkaInstanceSize{
											Id:               "x2",
											CapacityConsumed: 2,
										},
									},
								},
							},
						},
					},
				},
				dataplaneClusterConfig: dynamicScalingEnabledDataplaneClusterConfig,
				providerConfig:         buildProviderConfiguration("us-east-1", 1000, 1000, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			result:  []string{"x1", "x2"},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "when dynamic scaling is enabled, should return nil if region limit has been reached",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				kafkaConfig:            &defaultKafkaConf,
				dataplaneClusterConfig: dynamicScalingEnabledDataplaneClusterConfig,
				providerConfig:         buildProviderConfiguration("us-east-1", 1, 1, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mocks.BuildCluster(nil), nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			result:  nil,
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithReply([]map[string]interface{}{
						{
							"region":         testCriteria.Region,
							"cloud_provider": testCriteria.Provider,
							"multi_az":       testCriteria.MultiAZ,
							"name":           testKafkaRequestName,
							"size_id":        "x1",
							"instance_type":  testCriteria.SupportedInstanceType,
						},
					})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should return nil if no cluster capacity is left",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				kafkaConfig:            &defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration("us-east-1", 1000, 1000, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			result:  nil,
			wantErr: false,
		},
		{
			name: "should return nil if region limit has been reached",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				kafkaConfig:            &defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration("us-east-1", 1, 1, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return mocks.BuildCluster(nil), nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithReply([]map[string]interface{}{
						{
							"region":         testCriteria.Region,
							"cloud_provider": testCriteria.Provider,
							"multi_az":       testCriteria.MultiAZ,
							"name":           testKafkaRequestName,
							"size_id":        "x1",
							"instance_type":  testCriteria.SupportedInstanceType,
						},
					})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			result:  nil,
			wantErr: false,
		},
		{
			name:    "should return an error if criteria was not defined",
			fields:  fields{},
			args:    args{},
			result:  nil,
			wantErr: true,
			expectedErr: &errors.ServiceError{
				Code: errors.ErrorGeneral,
			},
		},
		{
			name: "should return an error if instance type in criteria is not supported",
			fields: fields{
				kafkaConfig: &defaultKafkaConf,
			},
			args: args{
				criteria: &FindClusterCriteria{
					Provider:              defaultCluster.CloudProvider,
					Region:                defaultCluster.Region,
					MultiAZ:               defaultCluster.MultiAZ,
					SupportedInstanceType: "unsupported",
				},
			},
			result:  nil,
			wantErr: true,
			expectedErr: &errors.ServiceError{
				Code: errors.ErrorInstanceTypeNotSupported,
			},
		},
		{
			name: "should return an error if db query failed",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				kafkaConfig:            &defaultKafkaConf,
				dataplaneClusterConfig: buildDataplaneClusterConfig(defaultDataplaneClusterConfig),
				providerConfig:         buildProviderConfiguration("us-east-1", 1000, 1000, false),
				clusterPlacementStrategy: &ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				criteria: testCriteria,
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE region = $1 AND cloud_provider = $2 AND instance_type = $3 AND actual_kafka_billing_model != $4`).
					WithArgs(testCriteria.Region, testCriteria.Provider, testCriteria.SupportedInstanceType, constants.BillingModelEnterprise.String()).
					WithQueryException()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			result:  nil,
			wantErr: true,
			expectedErr: &errors.ServiceError{
				Code: errors.ErrorGeneral,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				connectionFactory:        tt.fields.connectionFactory,
				kafkaConfig:              tt.fields.kafkaConfig,
				dataplaneClusterConfig:   tt.fields.dataplaneClusterConfig,
				providerConfig:           tt.fields.providerConfig,
				clusterPlacementStrategy: tt.fields.clusterPlacementStrategy,
			}

			got, err := k.GetAvailableSizesInRegion(tt.args.criteria)

			g.Expect(tt.result).To(gomega.Equal(got))
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			if tt.wantErr && tt.expectedErr != nil {
				g.Expect(tt.expectedErr.Code).To(gomega.Equal(err.Code))
			}
		})
	}
}

func Test_kafkaService_GetManagedKafkaByClusterID(t *testing.T) {
	type fields struct {
		connectionFactory                    *db.ConnectionFactory
		keycloakService                      sso.KeycloakService
		clusterService                       ClusterService
		kafkaConfig                          *config.KafkaConfig
		kafkaTLSCertificateManagementService kafkatlscertmgmt.KafkaTLSCertificateManagementService
	}
	type args struct {
		clusterID string
	}
	kafkaRequestList := dbapi.KafkaList{
		&dbapi.KafkaRequest{
			ClusterID:    testClusterID,
			InstanceType: "developer",
			SizeId:       "x1",
		},
	}
	managedkafkaCRWithoutCerts, _ := buildManagedKafkaCR(
		&dbapi.KafkaRequest{
			ClusterID:    testClusterID,
			InstanceType: "developer",
			SizeId:       "x1",
		},
		&config.KafkaConfig{
			EnableKafkaCNAMERegistration: true,
			SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
		},
		&sso.KeycloakServiceMock{
			GetConfigFunc: func() *keycloak.KeycloakConfig {
				return &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
				}
			},
			GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
				return &keycloak.KeycloakRealmConfig{}
			},
		}, kafkatlscertmgmt.Certificate{}, false)

	managedkafkaCRWithCert, _ := buildManagedKafkaCR(
		&dbapi.KafkaRequest{
			ClusterID:    testClusterID,
			InstanceType: "developer",
			SizeId:       "x1",
		},
		&config.KafkaConfig{
			EnableKafkaCNAMERegistration: true,
			SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
		},
		&sso.KeycloakServiceMock{
			GetConfigFunc: func() *keycloak.KeycloakConfig {
				return &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
				}
			},
			GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
				return &keycloak.KeycloakRealmConfig{}
			},
		}, kafkatlscertmgmt.Certificate{TLSCert: "crt-cert", TLSKey: "key-cert"}, true)

	managedkafkaCRWithPausedReconciliation, _ := buildManagedKafkaCR(
		&dbapi.KafkaRequest{
			ClusterID:    testClusterID,
			InstanceType: "developer",
			SizeId:       "x1",
		},
		&config.KafkaConfig{
			EnableKafkaCNAMERegistration: true,
			SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
		},
		&sso.KeycloakServiceMock{
			GetConfigFunc: func() *keycloak.KeycloakConfig {
				return &keycloak.KeycloakConfig{
					EnableAuthenticationOnKafka: true,
				}
			},
			GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
				return &keycloak.KeycloakRealmConfig{}
			},
		}, kafkatlscertmgmt.Certificate{}, true)

	managedkafkaCRWithPausedReconciliation.Annotations[managedkafka.ManagedKafkaBf2PauseReconciliationAnnotationKey] = "true"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []managedkafka.ManagedKafka
		wantErr bool
		setupFn func()
	}{
		{
			name: "should return the kafka by cluster id when external certificate is disabled",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsKafkaExternalCertificateEnabledFunc: func() bool {
						return false
					},
					GetCertificateFunc: nil, // assign to nil as it should never be called
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloak.KeycloakRealmConfig{}
					},
				},
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
					SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: nil, // setting to nil as it should never be called
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: false,
			want:    []managedkafka.ManagedKafka{*managedkafkaCRWithoutCerts},
			setupFn: func() {
				mocket.Catcher.Reset()
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaRequestList)
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should return the kafka by cluster id when external certificate is enabled",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsKafkaExternalCertificateEnabledFunc: func() bool {
						return true
					},
					GetCertificateFunc: func(ctx context.Context, request kafkatlscertmgmt.GetCertificateRequest) (kafkatlscertmgmt.Certificate, error) {
						return kafkatlscertmgmt.Certificate{
							TLSCert: "crt-cert",
							TLSKey:  "key-cert",
						}, nil
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloak.KeycloakRealmConfig{}
					},
				},
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
					SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{ClusterID: clusterID}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: false,
			want:    []managedkafka.ManagedKafka{*managedkafkaCRWithCert},
			setupFn: func() {
				mocket.Catcher.Reset()
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaRequestList)
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should pause reconciliation of the ManagedKafka in the data plane when fetching certificates fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsKafkaExternalCertificateEnabledFunc: func() bool {
						return true
					},
					GetCertificateFunc: func(ctx context.Context, request kafkatlscertmgmt.GetCertificateRequest) (kafkatlscertmgmt.Certificate, error) {
						return kafkatlscertmgmt.Certificate{}, fmt.Errorf("some error")
					},
				},
				keycloakService: &sso.KeycloakServiceMock{
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloak.KeycloakRealmConfig{}
					},
				},
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
					SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{ClusterID: clusterID}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: false,
			want:    []managedkafka.ManagedKafka{*managedkafkaCRWithPausedReconciliation},
			setupFn: func() {
				mocket.Catcher.Reset()
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response := converters.ConvertKafkaRequestList(kafkaRequestList)
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		tt.setupFn()
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{
				connectionFactory:                    tt.fields.connectionFactory,
				keycloakService:                      tt.fields.keycloakService,
				kafkaConfig:                          tt.fields.kafkaConfig,
				kafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
				clusterService:                       tt.fields.clusterService,
			}
			got, err := k.GetManagedKafkaByClusterID(tt.args.clusterID)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_kafkaService_GenerateReservedManagedKafkasByClusterID(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		kafkaConfig            *config.KafkaConfig
		clusterService         ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		clusterID string
	}

	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"

	testDeveloperInstanceType, err := kafkaSupportedInstanceTypesConfig.Configuration.GetKafkaInstanceTypeByID("developer")
	if err != nil {
		panic("unexpected test error")
	}
	testDeveloperX1InstanceSize, err := testDeveloperInstanceType.GetKafkaInstanceSizeByID("x1")
	if err != nil {
		panic("unexpected test error")
	}
	testStandardInstanceType, err := kafkaSupportedInstanceTypesConfig.Configuration.GetKafkaInstanceTypeByID("standard")
	if err != nil {
		panic("unexpected test error")
	}
	testStandardX1InstanceSize, err := testStandardInstanceType.GetKafkaInstanceSizeByID("x1")
	if err != nil {
		panic("unexpected test error")
	}

	testAvailableStrimziVersions := []api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
			},
		},
	}
	marshaledTestAvailableStrimziVersions, err := json.Marshal(testAvailableStrimziVersions)
	if err != nil {
		panic("unexpected test error")
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []managedkafka.ManagedKafka
		wantErr bool
	}{
		{
			name: "returns generated managed kafkas",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
					SupportedInstanceTypes:       &kafkaSupportedInstanceTypesConfig,
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					NodePrewarmingConfig: config.NodePrewarmingConfig{
						Configuration: map[string]config.InstanceTypeNodePrewarmingConfig{
							"developer": {
								NumReservedInstances: 1,
							},
							"standard": {
								NumReservedInstances: 2,
							},
						},
					},
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							ClusterID:                clusterID,
							Status:                   api.ClusterReady,
							SupportedInstanceType:    "developer,standard",
							AvailableStrimziVersions: marshaledTestAvailableStrimziVersions,
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: false,
			want: []managedkafka.ManagedKafka{
				managedkafka.ManagedKafka{
					Id: "reserved-kafka-developer-1",
					TypeMeta: metav1.TypeMeta{
						Kind:       "ManagedKafka",
						APIVersion: "managedkafka.bf2.org/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "reserved-kafka-developer-1",
						Namespace: "reserved-kafka-developer-1",
						Annotations: map[string]string{
							"bf2.org/id":          "reserved-kafka-developer-1",
							"bf2.org/placementId": "reserved-kafka-developer-1",
						},
						Labels: map[string]string{
							"bf2.org/kafkaInstanceProfileQuotaConsumed":    strconv.Itoa(testDeveloperX1InstanceSize.QuotaConsumed),
							"bf2.org/kafkaInstanceProfileType":             "developer",
							managedkafka.ManagedKafkaBf2DeploymentLabelKey: managedkafka.ManagedKafkaBf2DeploymentLabelValueReserved,
						},
					},
					Spec: managedkafka.ManagedKafkaSpec{
						Capacity: managedkafka.Capacity{
							IngressPerSec:               testDeveloperX1InstanceSize.IngressThroughputPerSec.String(),
							EgressPerSec:                testDeveloperX1InstanceSize.EgressThroughputPerSec.String(),
							TotalMaxConnections:         testDeveloperX1InstanceSize.TotalMaxConnections,
							MaxDataRetentionSize:        testDeveloperX1InstanceSize.MaxDataRetentionSize.String(),
							MaxPartitions:               testDeveloperX1InstanceSize.MaxPartitions,
							MaxDataRetentionPeriod:      testDeveloperX1InstanceSize.MaxDataRetentionPeriod,
							MaxConnectionAttemptsPerSec: testDeveloperX1InstanceSize.MaxConnectionAttemptsPerSec,
						},
						Endpoint: managedkafka.EndpointSpec{
							BootstrapServerHost: fmt.Sprintf("%s-dummyhost", "reserved-kafka-developer-1"),
						},
						Versions: managedkafka.VersionsSpec{
							Strimzi:  testAvailableStrimziVersions[0].Version,
							Kafka:    testAvailableStrimziVersions[0].KafkaVersions[0].Version,
							KafkaIBP: testAvailableStrimziVersions[0].GetLatestKafkaIBPVersion().Version,
						},
						Deleted: false,
						Owners:  []string{},
					},
					Status: managedkafka.ManagedKafkaStatus{},
				},
				managedkafka.ManagedKafka{
					Id: "reserved-kafka-standard-1",
					TypeMeta: metav1.TypeMeta{
						Kind:       "ManagedKafka",
						APIVersion: "managedkafka.bf2.org/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "reserved-kafka-standard-1",
						Namespace: "reserved-kafka-standard-1",
						Annotations: map[string]string{
							"bf2.org/id":          "reserved-kafka-standard-1",
							"bf2.org/placementId": "reserved-kafka-standard-1",
						},
						Labels: map[string]string{
							"bf2.org/kafkaInstanceProfileQuotaConsumed":    strconv.Itoa(testStandardX1InstanceSize.QuotaConsumed),
							"bf2.org/kafkaInstanceProfileType":             "standard",
							managedkafka.ManagedKafkaBf2DeploymentLabelKey: managedkafka.ManagedKafkaBf2DeploymentLabelValueReserved,
						},
					},
					Spec: managedkafka.ManagedKafkaSpec{
						Capacity: managedkafka.Capacity{
							IngressPerSec:               testStandardX1InstanceSize.IngressThroughputPerSec.String(),
							EgressPerSec:                testStandardX1InstanceSize.EgressThroughputPerSec.String(),
							TotalMaxConnections:         testStandardX1InstanceSize.TotalMaxConnections,
							MaxDataRetentionSize:        testStandardX1InstanceSize.MaxDataRetentionSize.String(),
							MaxPartitions:               testStandardX1InstanceSize.MaxPartitions,
							MaxDataRetentionPeriod:      testStandardX1InstanceSize.MaxDataRetentionPeriod,
							MaxConnectionAttemptsPerSec: testStandardX1InstanceSize.MaxConnectionAttemptsPerSec,
						},
						Endpoint: managedkafka.EndpointSpec{
							BootstrapServerHost: fmt.Sprintf("%s-dummyhost", "reserved-kafka-standard-1"),
						},
						Versions: managedkafka.VersionsSpec{
							Strimzi:  testAvailableStrimziVersions[0].Version,
							Kafka:    testAvailableStrimziVersions[0].KafkaVersions[0].Version,
							KafkaIBP: testAvailableStrimziVersions[0].GetLatestKafkaIBPVersion().Version,
						},
						Deleted: false,
						Owners:  []string{},
					},
					Status: managedkafka.ManagedKafkaStatus{},
				},
				managedkafka.ManagedKafka{
					Id: "reserved-kafka-standard-2",
					TypeMeta: metav1.TypeMeta{
						Kind:       "ManagedKafka",
						APIVersion: "managedkafka.bf2.org/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "reserved-kafka-standard-2",
						Namespace: "reserved-kafka-standard-2",
						Annotations: map[string]string{
							"bf2.org/id":          "reserved-kafka-standard-2",
							"bf2.org/placementId": "reserved-kafka-standard-2",
						},
						Labels: map[string]string{
							"bf2.org/kafkaInstanceProfileQuotaConsumed":    strconv.Itoa(testStandardX1InstanceSize.QuotaConsumed),
							"bf2.org/kafkaInstanceProfileType":             "standard",
							managedkafka.ManagedKafkaBf2DeploymentLabelKey: managedkafka.ManagedKafkaBf2DeploymentLabelValueReserved,
						},
					},
					Spec: managedkafka.ManagedKafkaSpec{
						Capacity: managedkafka.Capacity{
							IngressPerSec:               testStandardX1InstanceSize.IngressThroughputPerSec.String(),
							EgressPerSec:                testStandardX1InstanceSize.EgressThroughputPerSec.String(),
							TotalMaxConnections:         testStandardX1InstanceSize.TotalMaxConnections,
							MaxDataRetentionSize:        testStandardX1InstanceSize.MaxDataRetentionSize.String(),
							MaxPartitions:               testStandardX1InstanceSize.MaxPartitions,
							MaxDataRetentionPeriod:      testStandardX1InstanceSize.MaxDataRetentionPeriod,
							MaxConnectionAttemptsPerSec: testStandardX1InstanceSize.MaxConnectionAttemptsPerSec,
						},
						Endpoint: managedkafka.EndpointSpec{
							BootstrapServerHost: fmt.Sprintf("%s-dummyhost", "reserved-kafka-standard-2"),
						},
						Versions: managedkafka.VersionsSpec{
							Strimzi:  testAvailableStrimziVersions[0].Version,
							Kafka:    testAvailableStrimziVersions[0].KafkaVersions[0].Version,
							KafkaIBP: testAvailableStrimziVersions[0].GetLatestKafkaIBPVersion().Version,
						},
						Deleted: false,
						Owners:  []string{},
					},
					Status: managedkafka.ManagedKafkaStatus{},
				},
			},
		},
		{
			name: "returns an empty list when pre warming is not enabled for any instance type",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					NodePrewarmingConfig: config.NodePrewarmingConfig{
						Configuration: map[string]config.InstanceTypeNodePrewarmingConfig{},
					},
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							ClusterID:                clusterID,
							Status:                   api.ClusterReady,
							SupportedInstanceType:    "developer,standard",
							AvailableStrimziVersions: marshaledTestAvailableStrimziVersions,
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			want: []managedkafka.ManagedKafka{},
		},
		{
			name: "an error is returned when the provided ClusterID does not exist",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
		},
		{
			name: "an error is returned when trying to get the provided cluster returns an error",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("test error")
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
		},
		{
			name: "an empty list is returned when the provided ClusterID is not in ready status",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							ClusterID: clusterID,
							Status:    api.ClusterWaitingForKasFleetShardOperator,
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			want: []managedkafka.ManagedKafka{},
		},
		{
			name: "an error is returned when there are no ready strimzi versions",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
							{
								Version: strimziOperatorVersion,
								Ready:   false,
								KafkaVersions: []api.KafkaVersion{
									{
										Version: "2.7.0",
									},
								},
								KafkaIBPVersions: []api.KafkaIBPVersion{
									{
										Version: "2.7",
									},
								},
							},
						})
						if err != nil {
							t.Fatal("failed to convert available strimzi versions to json")
						}
						return &api.Cluster{
							ClusterID:                clusterID,
							Status:                   api.ClusterReady,
							AvailableStrimziVersions: availableStrimziVersions,
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
		},
		{
			name: "an error is returned when there are no kafka versions in the latest ready strimzi version",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					NodePrewarmingConfig: config.NodePrewarmingConfig{
						Configuration: map[string]config.InstanceTypeNodePrewarmingConfig{
							"developer": {
								NumReservedInstances: 1,
							},
							"standard": {
								NumReservedInstances: 1,
							},
						},
					},
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
							{
								Version:       strimziOperatorVersion,
								Ready:         true,
								KafkaVersions: []api.KafkaVersion{},
								KafkaIBPVersions: []api.KafkaIBPVersion{
									{
										Version: "2.7",
									},
								},
							},
						})
						if err != nil {
							t.Fatal("failed to convert available strimzi versions to json")
						}
						return &api.Cluster{
							ClusterID:                clusterID,
							Status:                   api.ClusterReady,
							AvailableStrimziVersions: availableStrimziVersions,
							SupportedInstanceType:    "developer",
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
		},
		{
			name: "an error is returned when there are no kafka ibp versions in the latest ready strimzi version",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					NodePrewarmingConfig: config.NodePrewarmingConfig{
						Configuration: map[string]config.InstanceTypeNodePrewarmingConfig{
							"developer": {
								NumReservedInstances: 1,
							},
							"standard": {
								NumReservedInstances: 1,
							},
						},
					},
				},
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
							{
								Version: strimziOperatorVersion,
								Ready:   true,
								KafkaVersions: []api.KafkaVersion{
									api.KafkaVersion{
										Version: "2.7.0",
									},
								},
								KafkaIBPVersions: []api.KafkaIBPVersion{},
							},
						})
						if err != nil {
							t.Fatal("failed to convert available strimzi versions to json")
						}
						return &api.Cluster{
							ClusterID:                clusterID,
							Status:                   api.ClusterReady,
							AvailableStrimziVersions: availableStrimziVersions,
							SupportedInstanceType:    "developer",
						}, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			k := &kafkaService{
				connectionFactory:      tt.fields.connectionFactory,
				clusterService:         tt.fields.clusterService,
				kafkaConfig:            tt.fields.kafkaConfig,
				dataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
			}
			got, err := k.GenerateReservedManagedKafkasByClusterID(tt.args.clusterID)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).Should(gomega.HaveLen(len(tt.want)))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_VerifyAndUpdateKafkaAdmin(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		authService       authorization.Authorization
	}
	type args struct {
		ctx          context.Context
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
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *errors.ServiceError
		setupFunc func()
	}{
		{
			name: "should return nil if it can Verify And Update Kafka Admin ",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				authService:       authorization.NewMockAuthorization(),
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:                "cluster-id",
							AvailableStrimziVersions: availableStrimziVersions,
						}, nil
					},
				},
			},
			args: args{
				ctx: auth.SetIsAdminContext(context.TODO(), true),
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					MaxDataRetentionSize:   "100",
				},
			},
			want: nil,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should return error if user is not authenticated",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				authService:       authorization.NewMockAuthorization(),
				clusterService: &ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:                "cluster-id",
							AvailableStrimziVersions: availableStrimziVersions,
						}, nil
					},
					IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
						return true, nil
					},
					CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
						return true, nil
					},
				},
			},
			args: args{
				ctx: auth.SetIsAdminContext(context.TODO(), false),
			},
			want: errors.New(errors.ErrorUnauthenticated, "user not authenticated"),
			setupFunc: func() {
				mocket.Catcher.Reset()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		tt.setupFunc()
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				authService:       tt.fields.authService,
			}
			g.Expect(k.VerifyAndUpdateKafkaAdmin(tt.args.ctx, tt.args.kafkaRequest)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_GetCNAMERecordStatus(t *testing.T) {
	type fields struct {
		awsConfig        *config.AWSConfig
		awsClientFactory aws.ClientFactory
	}

	CNAME_Id := "CNAME_Id"
	CNAME_Status := "CNAME_Status"

	awsConfig := &config.AWSConfig{}
	awsConfig.Route53.AccessKey = "Route53AccessKey"
	awsConfig.Route53.SecretAccessKey = "Route53SecretAccessKey"

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *CNameRecordStatus
		wantErr bool
	}{
		{
			name: "should get the CNAME record Status",
			fields: fields{
				awsConfig: awsConfig,
				awsClientFactory: aws.NewMockClientFactory(&aws.AWSClientMock{
					GetChangeFunc: func(changeId string) (*route53.GetChangeOutput, error) {
						return &route53.GetChangeOutput{
							ChangeInfo: &route53.ChangeInfo{
								Id:     &CNAME_Id,
								Status: &CNAME_Status,
							},
						}, nil
					},
				}),
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Region:        "us-east-1",
					CloudProvider: cloudproviders.AWS.String(),
				},
			},
			want: &CNameRecordStatus{
				Id:     &CNAME_Id,
				Status: &CNAME_Status,
			},
			wantErr: false,
		},
		{
			name: "should return error when it fails to get CNAME status",
			fields: fields{
				awsConfig: awsConfig,
				awsClientFactory: aws.NewMockClientFactory(&aws.AWSClientMock{
					GetChangeFunc: func(changeId string) (*route53.GetChangeOutput, error) {
						return nil, errors.GeneralError("unable to CNAME record status")
					},
				}),
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Region: "us-east-1",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{
				awsConfig:        tt.fields.awsConfig,
				awsClientFactory: tt.fields.awsClientFactory,
			}
			got, err := k.GetCNAMERecordStatus(tt.args.kafkaRequest)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_NewKafkaService(t *testing.T) {
	type args struct {
		connectionFactory                    *db.ConnectionFactory
		clusterService                       ClusterService
		keycloakService                      sso.KafkaKeycloakService
		kafkaConfig                          *config.KafkaConfig
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		awsConfig                            *config.AWSConfig
		quotaServiceFactory                  QuotaServiceFactory
		awsClientFactory                     aws.ClientFactory
		authorizationService                 authorization.Authorization
		providerConfig                       *config.ProviderConfig
		clusterPlacementStrategy             ClusterPlacementStrategy
		kafkaTLSCertificateManagementService kafkatlscertmgmt.KafkaTLSCertificateManagementService
	}
	tests := []struct {
		name string
		args args
		want *kafkaService
	}{
		{
			name: "should return the kafka service",
			args: args{
				connectionFactory:                    &db.ConnectionFactory{},
				clusterService:                       &ClusterServiceMock{},
				keycloakService:                      &sso.KeycloakServiceMock{},
				kafkaConfig:                          &config.KafkaConfig{},
				dataplaneClusterConfig:               &config.DataplaneClusterConfig{},
				awsConfig:                            &config.AWSConfig{},
				quotaServiceFactory:                  &QuotaServiceFactoryMock{},
				awsClientFactory:                     &aws.MockClientFactory{},
				providerConfig:                       &config.ProviderConfig{},
				clusterPlacementStrategy:             &ClusterPlacementStrategyMock{},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{},
			},
			want: &kafkaService{
				connectionFactory:                    &db.ConnectionFactory{},
				clusterService:                       &ClusterServiceMock{},
				keycloakService:                      &sso.KeycloakServiceMock{},
				kafkaConfig:                          &config.KafkaConfig{},
				dataplaneClusterConfig:               &config.DataplaneClusterConfig{},
				awsConfig:                            &config.AWSConfig{},
				quotaServiceFactory:                  &QuotaServiceFactoryMock{},
				awsClientFactory:                     &aws.MockClientFactory{},
				providerConfig:                       &config.ProviderConfig{},
				clusterPlacementStrategy:             &ClusterPlacementStrategyMock{},
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{},
			},
		},
	}

	for _, testcase := range tests {
		g := gomega.NewWithT(t)
		tt := testcase
		g.Expect(NewKafkaService(
			tt.args.connectionFactory,
			tt.args.clusterService,
			tt.args.keycloakService,
			tt.args.kafkaConfig,
			tt.args.dataplaneClusterConfig,
			tt.args.awsConfig,
			tt.args.quotaServiceFactory,
			tt.args.awsClientFactory,
			tt.args.authorizationService,
			tt.args.providerConfig,
			tt.args.clusterPlacementStrategy,
			tt.args.kafkaTLSCertificateManagementService)).To(gomega.Equal(tt.want))
	}
}

func Test_kafkaService_ListKafkasWithRoutesNotCreated(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    []*dbapi.KafkaRequest
		wantErr *errors.ServiceError
		setupFn func()
	}{
		{
			name: "should return the kafka by cluster id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			want: []*dbapi.KafkaRequest{buildKafkaRequest(nil)},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "kafka_requests"`).
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		tt.setupFn()
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
			}
			g.Expect(k.ListKafkasWithRoutesNotCreated()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_AssignBootstrapServerHost(t *testing.T) {
	type fields struct {
		clusterService ClusterService
		kafkaConfig    *config.KafkaConfig
	}
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return error if failed clusterDNS retrieval",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorBadRequest, "")
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
			name: "should use create bootstrap server host in kafka domain if kafkaconfig specifies",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: false,
		},
		{
			name: "should successfully assign BootstrapServerHost",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
					FindClusterByIDFunc: nil, // it should never be called
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: false,
		},
		{
			name: "should return not retrieve cluster DNS and successfully assign the kafka bootstrap server host if EnableKafkaCNAMERegistration is enabled",
			fields: fields{
				clusterService: nil, // setting to nil as it should never be called
				kafkaConfig: &config.KafkaConfig{
					EnableKafkaCNAMERegistration: true,
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{
				clusterService: test.fields.clusterService,
				kafkaConfig:    test.fields.kafkaConfig,
			}

			//set the bootstrap server host as empty, mimicking what really happens
			test.args.kafkaRequest.BootstrapServerHost = ""

			err := k.AssignBootstrapServerHost(test.args.kafkaRequest)
			g.Expect(err != nil).To(gomega.Equal(test.wantErr))

			if !test.wantErr {
				g.Expect(test.args.kafkaRequest.BootstrapServerHost).ToNot(gomega.BeEmpty())
			}
		})
	}
}

func Test_kafkaService_getRoute53RegionFromKafkaRequest(t *testing.T) {

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Route53 region is correctly returned for Kafka instances in AWS",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Region:        "anotherregion",
					CloudProvider: cloudproviders.AWS.String(),
				},
			},
			want:    aws.DefaultAWSRoute53Region,
			wantErr: false,
		},
		{
			name: "Route53 region is correctly returned for Kafka instances in GCP",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Region:        "anotherregiontwo",
					CloudProvider: cloudproviders.GCP.String(),
				},
			},
			want:    aws.DefaultGCPRoute53Region,
			wantErr: false,
		},
		{
			name: "An error is returned if the Kafka instance has an unknown cloud provider",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Region:        "us-east-1",
					CloudProvider: "anunknowncloudprovider",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &kafkaService{}
			res, err := k.getRoute53RegionFromKafkaRequest(tt.args.kafkaRequest)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_kafkaService_ManagedKafkasRoutesTLSCertificate(t *testing.T) {
	g := gomega.NewWithT(t)
	type fields struct {
		kafkaConfig                          *config.KafkaConfig
		connectionFactory                    *db.ConnectionFactory
		kafkaTLSCertificateManagementService kafkatlscertmgmt.KafkaTLSCertificateManagementService
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
			name: "do not reconcile the certificate if the Kafka is in a deleting state",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory:                    db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: nil,
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkasRoutesBaseDomainName: "",
					Meta: api.Meta{
						ID: "some-id",
					},
					Status: constants.KafkaRequestStatusDeleting.String(),
				},
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException() // an update shouldn't be performed in the database.
			},
			wantErr: false,
		},
		{
			name: "do not reconcile the certificate if the Kafka is in a deprovision state",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory:                    db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: nil,
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkasRoutesBaseDomainName: "",
					Meta: api.Meta{
						ID: "some-id",
					},
					Status: constants.KafkaOperationDeprovision.String(),
				},
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException() // an update shouldn't be performed in the database.
			},
			wantErr: false,
		},
		{
			name: "do not reconcile the certificate if the Kafka is in a suspended state",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory:                    db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: nil,
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkasRoutesBaseDomainName: "",
					Meta: api.Meta{
						ID: "some-id",
					},
					Status: constants.KafkaRequestStatusSuspended.String(),
				},
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException() // an update shouldn't be performed in the database.
			},
			wantErr: false,
		},
		{
			name: "use the kafka domain name from the configuration when generating the certificate if automatic certificate management not enabled",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						g.Expect(domain).To(gomega.Equal("some-kafka-domain.bf2.dev"))
						return kafkatlscertmgmt.CertificateManagementOutput{}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkasRoutesBaseDomainName: "",
					Meta: api.Meta{
						ID: "some-id",
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
		},
		{
			name: "use the kafka domain name from the configuration when generating the certificate if automatic certificate management is enabled and kafka already prepared",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain-ready.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						g.Expect(domain).To(gomega.Equal("some-kafka-domain-ready.bf2.dev"))
						return kafkatlscertmgmt.CertificateManagementOutput{}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkasRoutesBaseDomainName: "",
					Meta: api.Meta{
						ID: "some-id",
					},
					Status: "ready", // a Kafka with ready status is already prepared
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
		},
		{
			name: "concatenate the kafka id and the kafka domain name from the configuration and use the resulting string as the domain when managing the certificate if automatic certificate management is enabled and standard kafka not already prepared",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						g.Expect(domain).To(gomega.Equal("123.some-kafka-domain.bf2.dev"))
						return kafkatlscertmgmt.CertificateManagementOutput{}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "123",
					},
					KafkasRoutesBaseDomainName: "",
					InstanceType:               types.STANDARD.String(),
					Status:                     "preparing", // a Kafka with preparing status is not already prepared i.e it may not have certificate information populated in the database so it is safe to treat it as a new kafka
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
		},
		{
			name: "concatenate the 'trial' and the kafka domain name from the configuration and use the resulting string as the domain when managing the certificate if automatic certificate management is enabled and developer kafka not already prepared",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						g.Expect(domain).To(gomega.Equal("trial.some-kafka-domain.bf2.dev"))
						return kafkatlscertmgmt.CertificateManagementOutput{}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "123",
					},
					KafkasRoutesBaseDomainName: "",
					InstanceType:               types.DEVELOPER.String(),
					Status:                     "preparing", // a Kafka with preparing status is not already prepared i.e it may not have certificate information populated in the database so it is safe to treat it as a new kafka
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
		},
		{
			name: "return an error when certificate management fails",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						return kafkatlscertmgmt.CertificateManagementOutput{}, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "456",
					},
					KafkasRoutesBaseDomainName: "",
					Status:                     "preparing", // a Kafka with preparing status is not already prepared i.e it may not have certificate information populated in the database so it is safe to treat it as a new kafka
				},
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException() // an update shouldn't be performed in the database.
			},
			wantErr: true,
		},
		{
			name: "return an error when database update fails",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						return kafkatlscertmgmt.CertificateManagementOutput{
							TLSCertRef: "some-cert-ref",
							TLSKeyRef:  "some-key-ref",
						}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "456",
					},
					KafkasRoutesBaseDomainName: "",
					Status:                     "preparing", // a Kafka with preparing status is not already prepared i.e it may not have certificate information populated in the database so it is safe to treat it as a new kafka
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "kafka_requests"`).WithExecException()
			},
			wantErr: true,
		},
		{
			name: "do not perform a database update if certificate references have not changed",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "some-kafka-domain.bf2.dev",
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
				kafkaTLSCertificateManagementService: &kafkatlscertmgmt.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafkatlscertmgmt.CertificateManagementOutput, error) {
						return kafkatlscertmgmt.CertificateManagementOutput{
							TLSCertRef: "some-cert-ref",
							TLSKeyRef:  "some-key-ref",
						}, nil
					},
				},
			},
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Meta: api.Meta{
						ID: "456",
					},
					KafkasRoutesBaseDomainName:      "some-kafka-domain.bf2.dev",
					KafkasRoutesBaseDomainTLSKeyRef: "some-key-ref",
					KafkasRoutesBaseDomainTLSCrtRef: "some-cert-ref",
					Status:                          "ready",
				},
			},
			setupFn: func() {
				mocket.Catcher.NewMock().WithExecException() // an update shouldn't be performed in the database.
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			testcase.setupFn()
			k := &kafkaService{
				kafkaConfig:                          testcase.fields.kafkaConfig,
				connectionFactory:                    testcase.fields.connectionFactory,
				kafkaTLSCertificateManagementService: testcase.fields.kafkaTLSCertificateManagementService,
			}
			err := k.ManagedKafkasRoutesTLSCertificate(testcase.args.kafkaRequest)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
