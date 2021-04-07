package services

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusterservicetest"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	constants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	dbConverters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
)

const (
	JwtKeyFile = "test/support/jwt_private_key.pem"
	JwtCAFile  = "test/support/jwt_ca.pem"
)

var (
	testKafkaRequestRegion   = "us-east-1"
	testKafkaRequestProvider = "aws"
	testKafkaRequestName     = "test-cluster"
	testClusterID            = "test-cluster-id"
	testSyncsetID            = "test-syncset-id"
	testID                   = "test"
	testUser                 = "test-user"
	kafkaRequestTableName    = "kafka_requests"
)

// build a test kafka request
func buildKafkaRequest(modifyFn func(kafkaRequest *api.KafkaRequest)) *api.KafkaRequest {
	kafkaRequest := &api.KafkaRequest{
		Meta: api.Meta{
			ID: testID,
		},
		Region:        testKafkaRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testKafkaRequestProvider,
		Name:          testKafkaRequestName,
		MultiAZ:       false,
		Owner:         testUser,
	}
	if modifyFn != nil {
		modifyFn(kafkaRequest)
	}
	return kafkaRequest
}

// This test should act as a "golden" test to describe the general testing approach taken in the service, for people
// onboarding into development of the service.
func Test_kafkaService_Get(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case kafkaService
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to kafkaService.Create
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
		want *api.KafkaRequest
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
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(buildKafkaRequest(nil)))
			},
		},
	}
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
				syncsetService:    tt.fields.syncsetService,
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
			// can use reflect.DeepEqual to compare the actual struct with the expected struct
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kafkaService_GetById(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case kafkaService
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to kafkaService.Create
	type args struct {
		id string
	}

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
		want *api.KafkaRequest
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
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(buildKafkaRequest(nil)))
			},
		},
	}
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
				syncsetService:    tt.fields.syncsetService,
			}
			// we're testing the kafkaService.Get function so use the 'args' to provide arguments to the function
			got, err := k.GetById(tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// in our test case we used 'want' to define the output api.KafkaRequest that we expect to be returned, we
			// can use reflect.DeepEqual to compare the actual struct with the expected struct
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kafkaService_HasAvailableCapacity(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		kafkaConfig       *config.KafkaConfig
	}

	tests := []struct {
		name        string
		fields      fields
		setupFn     func()
		wantErr     bool
		hasCapacity bool
	}{
		{
			name:        "capacity exhausted",
			hasCapacity: false,
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaCapacity: config.KafkaCapacityConfig{
						MaxCapacity: 1000,
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{{"a": "1000"}})
			},
			wantErr: false,
		},
		{
			name:        "capacity available",
			hasCapacity: true,
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					KafkaCapacity: config.KafkaCapacityConfig{
						MaxCapacity: 1000,
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{{"a": "999"}})
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
				kafkaConfig:       tt.fields.kafkaConfig,
			}

			if hasCapacity, err := k.HasAvailableCapacity(); (err != nil) != tt.wantErr {
				t.Errorf("HasAvailableCapacity() error = %v, wantErr = %v", err, tt.wantErr)
			} else if hasCapacity != tt.hasCapacity {
				t.Errorf("HasAvailableCapacity() hasCapacity = %v, wanted = %v", hasCapacity, tt.hasCapacity)
			}
		})
	}
}

func Test_kafkaService_Create(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
		clusterService    ClusterService
		keycloakService   KeycloakService
		kafkaConfig       *config.KafkaConfig
	}
	type args struct {
		kafkaRequest *api.KafkaRequest
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
			name: "successful syncset creation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId string, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							KafkaRealm: &config.KeycloakRealmConfig{
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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: false,
		},
		{
			name: "failed syncset creation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId string, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
						return nil, errors.New(errors.ErrorBadRequest, "")
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							KafkaRealm: &config.KeycloakRealmConfig{
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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: true,
		},
		{
			name: "failed clusterDNS retrieval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId string, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorBadRequest, "")
					},
				},
				keycloakService: &KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							KafkaRealm: &config.KeycloakRealmConfig{
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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: true,
		},
		{
			name: "validate BootstrapServerHost truncate",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId string, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							KafkaRealm: &config.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.Name = longKafkaName
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr:                 false,
			wantBootstrapServerHost: fmt.Sprintf("%s-%s.clusterDNS", truncateString(longKafkaName, truncatedNameLen), testID),
		},
		{
			name: "should not create sync set if EnableKasFleetshardSync is true",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *cmv1.SyncsetBuilder, syncsetId string, clusterId string) (*cmv1.Syncset, *errors.ServiceError) {
						panic("this should not be called")
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &KeycloakServiceMock{
					RegisterKafkaClientInSSOFunc: func(kafkaNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							KafkaRealm: &config.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{
					EnableKasFleetshardSync: true,
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
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
				syncsetService:    tt.fields.syncsetService,
				clusterService:    tt.fields.clusterService,
				keycloakService:   tt.fields.keycloakService,
				kafkaConfig:       tt.fields.kafkaConfig,
				awsConfig:         config.NewAWSConfig(),
			}

			if err := k.Create(tt.args.kafkaRequest); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantBootstrapServerHost != "" && tt.args.kafkaRequest.BootstrapServerHost != tt.wantBootstrapServerHost {
				t.Errorf("BootstrapServerHost error. Actual = %v, wantBootstrapServerHost = %v", tt.args.kafkaRequest.BootstrapServerHost, tt.wantBootstrapServerHost)
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
		kafkaRequest *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
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
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
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
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
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
		})
	}
}

func Test_kafkaService_Delete(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
		keycloakService   KeycloakService
		kafkaConfig       *config.KafkaConfig
		quotaService      QuotaService
	}
	type args struct {
		kafkaRequest *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when syncset deletion fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					DeleteFunc: func(syncsetId string, clusterId string) (int, *errors.ServiceError) {
						return http.StatusInternalServerError, errors.GeneralError("error deleting syncset")
					},
				},
				keycloakService: &KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			wantErr: true,
		},
		{
			name: "successful delete the kafka instance when synset delete successful",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					DeleteFunc: func(syncsetId string, clusterId string) (int, *errors.ServiceError) {
						return http.StatusNoContent, nil
					},
				},
				keycloakService: &KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(dbConverters.ConvertKafkaRequest(&api.KafkaRequest{
					Meta: api.Meta{
						ID: testID,
					},
					Status: constants.KafkaRequestStatusAccepted.String(),
				}))
			},
		},
		{
			name: "successful delete the kafka instance when synset not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					DeleteFunc: func(syncsetId string, clusterId string) (int, *errors.ServiceError) {
						return http.StatusNotFound, errors.GeneralError("syncset not found")
					},
				},
				keycloakService: &KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return &config.KeycloakConfig{
							EnableAuthenticationOnKafka: true,
						}
					},
				},
				kafkaConfig: &config.KafkaConfig{},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(dbConverters.ConvertKafkaRequest(&api.KafkaRequest{
					Meta: api.Meta{
						ID: testID,
					},
					Region:        clusterservicetest.MockClusterRegion,
					ClusterID:     clusterservicetest.MockClusterID,
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					MultiAZ:       false,
					Status:        constants.KafkaRequestStatusPreparing.String(),
				}))
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
				syncsetService:    tt.fields.syncsetService,
				keycloakService:   tt.fields.keycloakService,
				kafkaConfig:       tt.fields.kafkaConfig,
				awsConfig:         config.NewAWSConfig(),
				quotaService:      tt.fields.quotaService,
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
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
		clusterService    ClusterService
		quotaService      QuotaService
	}

	type errorCheck struct {
		wantErr  bool
		code     errors.ServiceErrorCode
		httpCode int
	}

	type args struct {
		kafkaRequest *api.KafkaRequest
	}
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
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					ReserveQuotaFunc: func(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError) {
						return true, "", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
				mocket.Catcher.NewMock().WithQuery("INSERT").WithReply(nil)
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "registering kafka too many instances",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					ReserveQuotaFunc: func(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError) {
						return true, "", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "1000"}})
				mocket.Catcher.NewMock().WithQuery("INSERT").WithReply(nil)
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyKafkaInstancesReached,
				httpCode: http.StatusTooManyRequests,
			},
		},
		{
			name: "registering kafka job fails: postgres error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					ReserveQuotaFunc: func(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError) {
						return true, "", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
				mocket.Catcher.NewMock().WithQuery("INSERT").WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorGeneral,
				httpCode: http.StatusInternalServerError,
			},
		},
		{
			name: "registering kafka job fails: quota error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					ReserveQuotaFunc: func(productID string, clusterID string, kafkaID string, owner string, reserve bool, availability string) (bool, string, *errors.ServiceError) {
						return false, "", nil
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
				mocket.Catcher.NewMock().WithQuery("INSERT").WithExecException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorGeneral,
				httpCode: http.StatusInternalServerError,
			},
		},
	}

	kafkaConf := config.KafkaConfig{
		KafkaCapacity: config.KafkaCapacityConfig{
			MaxCapacity: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				syncsetService:    tt.fields.syncsetService,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       &kafkaConf,
				awsConfig:         config.NewAWSConfig(),
				quotaService:      tt.fields.quotaService,
			}

			err := k.RegisterKafkaJob(tt.args.kafkaRequest)

			if (err != nil) != tt.error.wantErr {
				t.Errorf("RegisterKafkaJob() error = %v, wantErr = %v", err, tt.error.wantErr)
			}

			if tt.error.wantErr && err.Code != tt.error.code {
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

func Test_kafkaService_List(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	type args struct {
		ctx      context.Context
		listArgs *ListArguments
	}

	type want struct {
		kafkaList  api.KafkaList
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

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    want
		wantErr bool
		setupFn func(api.KafkaList)
	}{
		{
			name: "success: list with default values",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: api.KafkaList{
					&api.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name",
						Status:        "accepted",
						Owner:         testUser,
					},
					&api.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name2",
						Status:        "accepted",
						Owner:         testUser,
					},
				},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  2,
					Total: 2,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList api.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery("count").WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response, err := dbConverters.ConvertKafkaRequestList(kafkaList)
				if err != nil {
					t.Errorf("List() failed to convert KafkaRequestList: %s", err.Error())
				}
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
			},
		},
		{
			name: "success: list with specified size",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &ListArguments{
					Page: 1,
					Size: 1,
				},
			},
			want: want{
				kafkaList: api.KafkaList{
					&api.KafkaRequest{
						Region:        testKafkaRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testKafkaRequestProvider,
						MultiAZ:       false,
						Name:          "dummy-cluster-name",
						Status:        "accepted",
						Owner:         testUser,
					},
				},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  1,
					Total: 5,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList api.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": "5"}}
				mocket.Catcher.NewMock().WithQuery("count").WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response, err := dbConverters.ConvertKafkaRequestList(kafkaList)
				if err != nil {
					t.Errorf("List() failed to convert KafkaRequestList: %s", err.Error())
				}
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
			},
		},
		{
			name: "success: return empty list if no kafka requests available for user",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: api.KafkaList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: false,
			setupFn: func(kafkaList api.KafkaList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery("count").WithReply(totalCountResponse)

				// actual query to return list of kafka requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response, err := dbConverters.ConvertKafkaRequestList(kafkaList)
				if err != nil {
					t.Errorf("List() failed to convert KafkaRequestList: %s", err.Error())
				}
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
			},
		},
		{
			name: "fail: user credentials not available in context",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				ctx: context.TODO(),
				listArgs: &ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList:  nil,
				pagingMeta: nil,
			},
			wantErr: true,
			setupFn: func(kafkaList api.KafkaList) {
				mocket.Catcher.Reset()

				totalCountResponse := []map[string]interface{}{{"count": len(kafkaList)}}
				mocket.Catcher.NewMock().WithQuery("count").WithReply(totalCountResponse)

				query := fmt.Sprintf(`SELECT * FROM "%s"`, kafkaRequestTableName)
				response, err := dbConverters.ConvertKafkaRequestList(kafkaList)
				if err != nil {
					t.Errorf("List() failed to convert KafkaRequestList: %s", err.Error())
				}
				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
			},
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				ctx: authenticatedCtx,
				listArgs: &ListArguments{
					Page: 1,
					Size: 100,
				},
			},
			want: want{
				kafkaList: api.KafkaList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: true,
			setupFn: func(kafkaList api.KafkaList) {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn(tt.want.kafkaList)
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				syncsetService:    tt.fields.syncsetService,
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
			if !reflect.DeepEqual(pagingMeta, tt.want.pagingMeta) {
				t.Errorf("kafka.Service.List(): Paging meta returned is not correct:\n\tgot: %+v\n\twant: %+v", pagingMeta, tt.want.pagingMeta)
			}

			// compare wanted vs actual results
			if len(result) != len(tt.want.kafkaList) {
				t.Errorf("kafka.Service.List(): total number of results: got = %d want = %d", len(result), len(tt.want.kafkaList))
			}

			for i, got := range result {
				if !reflect.DeepEqual(got, tt.want.kafkaList[i]) {
					t.Errorf("kafkaService.List():\ngot = %+v\nwant = %+v", got, tt.want.kafkaList[i])
				}
			}
		})
	}
}

func Test_kafkaService_ListByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
		clusterService    ClusterService
	}
	type args struct {
		status constants.KafkaStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*api.KafkaRequest
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
			want: []*api.KafkaRequest{},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := &kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				syncsetService:    tt.fields.syncsetService,
				clusterService:    tt.fields.clusterService,
				kafkaConfig:       config.NewKafkaConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			got, err := k.ListByStatus(tt.args.status)
			// check errors
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListByStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_kafkaService_UpdateStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
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
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusDeprovision.String()
				})))
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
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusDeprovision.String()
				})))
			},
			args: args{
				id:     testID,
				status: constants.KafkaRequestStatusDeleted,
			},
		},
		{
			name:         "success",
			wantExecuted: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(buildKafkaRequest(func(kafkaRequest *api.KafkaRequest) {
					kafkaRequest.Status = constants.KafkaRequestStatusPreparing.String()
				})))
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
				syncsetService:    tt.fields.syncsetService,
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
		syncsetService    SyncsetService
		clusterService    ClusterService
	}
	type args struct {
		kafkaRequest *api.KafkaRequest
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
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := kafkaService{
				connectionFactory: tt.fields.connectionFactory,
				syncsetService:    tt.fields.syncsetService,
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
				mocket.Catcher.Reset().NewMock().WithQuery(fmt.Sprintf(`UPDATE "kafka_requests" SET "status" = %s`, constants.KafkaRequestStatusDeprovision)).WithReply(nil)
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
				mocket.Catcher.Reset().NewMock().WithQuery(fmt.Sprintf(`UPDATE "kafka_requests" SET "status" = %s`, constants.KafkaRequestStatusDeprovision)).WithReply(nil)
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
			}
			err := k.DeprovisionExpiredKafkas(tt.args.kafkaAgeInMins)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
