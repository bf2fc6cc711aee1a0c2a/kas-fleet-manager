package services

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/route53"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/dinosaurs/types"
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
)

const (
	JwtKeyFile         = "test/support/jwt_private_key.pem"
	JwtCAFile          = "test/support/jwt_ca.pem"
	MaxClusterCapacity = 1000
)

var (
	testDinosaurRequestRegion   = "us-east-1"
	testDinosaurRequestProvider = "aws"
	testDinosaurRequestName     = "test-cluster"
	testClusterID               = "test-cluster-id"
	testID                      = "test"
	testUser                    = "test-user"
	dinosaurRequestTableName    = "dinosaur_requests"
)

// build a test dinosaur request
func buildDinosaurRequest(modifyFn func(dinosaurRequest *dbapi.DinosaurRequest)) *dbapi.DinosaurRequest {
	dinosaurRequest := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID:        testID,
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Region:        testDinosaurRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testDinosaurRequestProvider,
		Name:          testDinosaurRequestName,
		MultiAZ:       false,
		Owner:         testUser,
	}
	if modifyFn != nil {
		modifyFn(dinosaurRequest)
	}
	return dinosaurRequest
}

// This test should act as a "golden" test to describe the general testing approach taken in the service, for people
// onboarding into development of the service.
func Test_dinosaurService_Get(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case dinosaurService
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to dinosaurService.PrepareDinosaurRequest
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
		want *dbapi.DinosaurRequest
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
			want: buildDinosaurRequest(nil),
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE id = $1 AND owner = $2`).
					WithArgs(testID, testUser).
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(nil)))
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
			// we're testing the dinosaurService struct, so use the 'fields' to create one
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
			}
			// we're testing the dinosaurService.Get function so use the 'args' to provide arguments to the function
			got, err := k.Get(tt.args.ctx, tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// in our test case we used 'want' to define the output api.DinosaurRequest that we expect to be returned, we
			// can use reflect.DeepEqual to compare the actual struct with the expected struct
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dinosaurService_GetById(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case dinosaurService
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to dinosaurService.PrepareDinosaurRequest
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
		want *dbapi.DinosaurRequest
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
			want: buildDinosaurRequest(nil),
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(nil)))
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
			// we're testing the dinosaurService struct, so use the 'fields' to create one
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
			}
			// we're testing the dinosaurService.Get function so use the 'args' to provide arguments to the function
			got, err := k.GetById(tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// in our test case we used 'want' to define the output api.DinosaurRequest that we expect to be returned, we
			// can use reflect.DeepEqual to compare the actual struct with the expected struct
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dinosaurService_HasAvailableCapacity(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		dinosaurConfig    *config.DinosaurConfig
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
				dinosaurConfig: &config.DinosaurConfig{
					DinosaurCapacity: config.DinosaurCapacityConfig{
						MaxCapacity: MaxClusterCapacity,
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply([]map[string]interface{}{{"a": fmt.Sprintf("%d", MaxClusterCapacity)}})
			},
			wantErr: false,
		},
		{
			name:        "capacity available",
			hasCapacity: true,
			fields: fields{
				dinosaurConfig: &config.DinosaurConfig{
					DinosaurCapacity: config.DinosaurCapacityConfig{
						MaxCapacity: MaxClusterCapacity,
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply([]map[string]interface{}{{"a": "999"}})
				// This is necessary otherwise if the query is wrong, the caller will receive a `0` result al `count`
				mocket.Catcher.NewMock().WithExecException().WithQueryException() // With this, an expected query will throw an exception
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				dinosaurConfig:    tt.fields.dinosaurConfig,
			}

			if hasCapacity, err := k.HasAvailableCapacity(); (err != nil) != tt.wantErr {
				t.Errorf("HasAvailableCapacity() error = %v, wantErr = %v", err, tt.wantErr)
			} else if hasCapacity != tt.hasCapacity {
				t.Errorf("HasAvailableCapacity() hasCapacity = %v, wanted = %v", hasCapacity, tt.hasCapacity)
			}
		})
	}
}

func Test_dinosaurService_PrepareDinosaurRequest(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		keycloakService   services.KeycloakService
		dinosaurConfig    *config.DinosaurConfig
	}
	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}

	longDinosaurName := "long-dinosaur-name-which-will-be-truncated-since-route-host-names-are-limited-to-63-characters"

	tests := []struct {
		name                    string
		fields                  fields
		args                    args
		setupFn                 func()
		wantErr                 bool
		wantBootstrapServerHost string
	}{
		{
			name: "successful dinosaur request preparation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					RegisterDinosaurClientInSSOFunc: func(dinosaurNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							DinosaurRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
					CreateServiceAccountInternalFunc: func(request services.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests"`)
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
				keycloakService: &services.KeycloakServiceMock{
					RegisterDinosaurClientInSSOFunc: func(dinosaurNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							DinosaurRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
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
				keycloakService: &services.KeycloakServiceMock{
					RegisterDinosaurClientInSSOFunc: func(dinosaurNamespace string, orgId string) (string, *errors.ServiceError) {
						return "secret", nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							DinosaurRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
						}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.Name = longDinosaurName
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr:                 false,
			wantBootstrapServerHost: fmt.Sprintf("%s-%s.clusterDNS", TruncateString(longDinosaurName, truncatedNameLen), testID),
		},
		{
			name: "failed SSO client creation",
			fields: fields{
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					RegisterDinosaurClientInSSOFunc: func(dinosaurNamespace string, orgId string) (string, *errors.ServiceError) {
						return "", errors.FailedToCreateSSOClient("failed to create the sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							DinosaurRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnDinosaur: true,
						}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
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
				keycloakService: &services.KeycloakServiceMock{
					RegisterDinosaurClientInSSOFunc: func(dinosaurNamespace string, orgId string) (string, *errors.ServiceError) {
						return "dsd", nil
					},
					CreateServiceAccountInternalFunc: func(request services.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.FailedToCreateSSOClient("failed to create the sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							DinosaurRealm: &keycloak.KeycloakRealmConfig{
								ClientID: "test",
							},
							EnableAuthenticationOnDinosaur: true,
						}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				keycloakService:   tt.fields.keycloakService,
				dinosaurConfig:    tt.fields.dinosaurConfig,
				awsConfig:         config.NewAWSConfig(),
			}

			if err := k.PrepareDinosaurRequest(tt.args.dinosaurRequest); (err != nil) != tt.wantErr {
				t.Errorf("PrepareDinosaurRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantBootstrapServerHost != "" && tt.args.dinosaurRequest.BootstrapServerHost != tt.wantBootstrapServerHost {
				t.Errorf("BootstrapServerHost error. Actual = %v, wantBootstrapServerHost = %v", tt.args.dinosaurRequest.BootstrapServerHost, tt.wantBootstrapServerHost)
			}

			if !tt.wantErr && tt.args.dinosaurRequest.Namespace == "" {
				t.Errorf("PrepareDinosaurRequest() dinosaurRequest.Namespace = \"\", want = %v", fmt.Sprintf("dinosaur-%s", tt.args.dinosaurRequest.ID))
			}
		})
	}
}

func Test_dinosaurService_RegisterDinosaurDeprovisionJob(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		quotaService      QuotaService
	}
	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
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
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
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
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			wantErr:    true,
			wantErrMsg: "DINOSAURS-MGMT-9",
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "dinosaur_requests"`).WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.RegisterDinosaurDeprovisionJob(context.TODO(), tt.args.dinosaurRequest.ID)
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

func Test_dinosaurService_Delete(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		keycloakService   services.KeycloakService
		dinosaurConfig    *config.DinosaurConfig
	}
	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "successfully deletes a Dinosaur request when it has not been assigned to an OSD cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &services.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(dinosaurClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnDinosaur: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successfully deletes a Dinosaur request and cleans up all of its dependencies",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &services.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(dinosaurClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnDinosaur: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "deleted_at"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "fail to delete dinosaur request: error when deleting sso client",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &services.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(dinosaurClusterName string) *errors.ServiceError {
						return errors.FailedToDeleteSSOClient("failed to delete sso client")
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnDinosaur: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			wantErr: true,
		},
		{
			name: "fail to delete dinosaur request: error when canary service account",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				keycloakService: &services.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(dinosaurClusterName string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnDinosaur: true,
						}
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			wantErr: true,
		},
		{
			name: "fail to delete dinosaur request: error when deleting CNAME records",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *errors.ServiceError) {
						return "", errors.GeneralError("failed to get cluster dns")
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(dinosaurClusterName string) *errors.ServiceError {
						return nil
					},
					DeleteServiceAccountInternalFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return &keycloak.KeycloakConfig{
							EnableAuthenticationOnDinosaur: true,
						}
					},
				},
				dinosaurConfig: &config.DinosaurConfig{
					EnableDinosaurExternalCertificate: true,
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = testID
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				keycloakService:   tt.fields.keycloakService,
				dinosaurConfig:    tt.fields.dinosaurConfig,
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Delete(tt.args.dinosaurRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_dinosaurService_RegisterDinosaurJob(t *testing.T) {

	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
		quotaService      QuotaService
		dinosaurConfig    config.DinosaurConfig
	}

	type errorCheck struct {
		wantErr  bool
		code     errors.ServiceErrorCode
		httpCode int
	}

	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}

	defaultDinosaurConf := config.DinosaurConfig{
		DinosaurCapacity: config.DinosaurCapacityConfig{
			MaxCapacity: MaxClusterCapacity,
		},
		Quota: config.NewDinosaurQuotaConfig(),
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		error   errorCheck
	}{
		{
			name: "registering dinosaur job succeeds",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService:    nil,
				dinosaurConfig:    defaultDinosaurConf,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					dinosaurRequest.ID = ""
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
				mocket.Catcher.NewMock().WithQuery("INSERT")
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr: false,
			},
		},
		{
			name: "registering dinosaur job eval disabled",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterService:    nil,
				dinosaurConfig: config.DinosaurConfig{
					DinosaurCapacity: config.DinosaurCapacityConfig{
						MaxCapacity: MaxClusterCapacity,
					},
					Quota: &config.DinosaurQuotaConfig{
						Type:                   api.QuotaManagementListQuotaType.String(),
						AllowEvaluatorInstance: false,
					},
				},
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						// No RHOSAK quota assigned
						return instanceType != types.STANDARD, nil
					},
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					// we need to empty to ID otherwise an UPDATE will be performed instead of an insert
					dinosaurRequest.ID = ""
				}),
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorForbidden,
				httpCode: http.StatusForbidden,
			},
		},
		{
			name: "registering dinosaur too many instances",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dinosaurConfig:    defaultDinosaurConf,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": fmt.Sprintf("%d", MaxClusterCapacity)}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyDinosaurInstancesReached,
				httpCode: http.StatusTooManyRequests,
			},
		},
		{
			name: "registering dinosaur too many eval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dinosaurConfig:    defaultDinosaurConf,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						// No RHOSAK quota assigned
						return instanceType != types.STANDARD, nil
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE "dinosaur_requests"."deleted_at" IS NULL`).
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.
					NewMock().
					WithQuery(`SELECT count(1) FROM "dinosaur_requests" WHERE instance_type = $1 AND owner = $2 AND (organisation_id = $3)`).
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			error: errorCheck{
				wantErr:  true,
				code:     errors.ErrorTooManyDinosaurInstancesReached,
				httpCode: http.StatusTooManyRequests,
			},
		},
		{
			name: "registering dinosaur job fails: postgres error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dinosaurConfig:    defaultDinosaurConf,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						return true, nil
					},
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "fake-subscription-id", nil
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
					dinosaurRequest.ID = ""
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
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
			name: "registering dinosaur job fails: quota error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dinosaurConfig:    defaultDinosaurConf,
				clusterService:    nil,
				quotaService: &QuotaServiceMock{
					CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
						return false, errors.InsufficientQuotaError("insufficient quota error")
					},
				},
			},
			args: args{
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT count").WithReply([]map[string]interface{}{{"count": "200"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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

			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				dinosaurConfig:    &tt.fields.dinosaurConfig,
				awsConfig:         config.NewAWSConfig(),
				quotaServiceFactory: &QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quotaType api.QuotaType) (QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}

			err := k.RegisterDinosaurJob(tt.args.dinosaurRequest)

			if (err != nil) != tt.error.wantErr {
				t.Errorf("RegisterDinosaurJob() error = %v, wantErr = %v", err, tt.error.wantErr)
			}

			if tt.error.wantErr {
				if err.Code != tt.error.code {
					t.Errorf("RegisterDinosaurJob() received error code %v, expected error %v", err.Code, tt.error.code)
				}
				if err.HttpCode != tt.error.httpCode {
					t.Errorf("RegisterDinosaurJob() received http code %v, expected %v", err.HttpCode, tt.error.httpCode)
				}
			}
		})
	}
}

func Test_dinosaurService_List(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		ctx      context.Context
		listArgs *services.ListArguments
	}

	type want struct {
		dinosaurList dbapi.DinosaurList
		pagingMeta   *api.PagingMeta
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
		setupFn func(dbapi.DinosaurList)
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
				dinosaurList: dbapi.DinosaurList{
					&dbapi.DinosaurRequest{
						Region:        testDinosaurRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testDinosaurRequestProvider,
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
					&dbapi.DinosaurRequest{
						Region:        testDinosaurRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testDinosaurRequestProvider,
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
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(dinosaurList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply(totalCountResponse)

				// actual query to return list of dinosaur requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, dinosaurRequestTableName)
				response := converters.ConvertDinosaurRequestList(dinosaurList)
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
				dinosaurList: dbapi.DinosaurList{
					&dbapi.DinosaurRequest{
						Region:        testDinosaurRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testDinosaurRequestProvider,
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
					&dbapi.DinosaurRequest{
						Region:        testDinosaurRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testDinosaurRequestProvider,
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
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(dinosaurList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply(totalCountResponse)

				// actual query to return list of dinosaur requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, dinosaurRequestTableName)
				response := converters.ConvertDinosaurRequestList(dinosaurList)
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
				dinosaurList: dbapi.DinosaurList{
					&dbapi.DinosaurRequest{
						Region:        testDinosaurRequestRegion,
						ClusterID:     testClusterID,
						CloudProvider: testDinosaurRequestProvider,
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
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": "5"}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply(totalCountResponse)

				// actual query to return list of dinosaur requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, dinosaurRequestTableName)
				response := converters.ConvertDinosaurRequestList(dinosaurList)

				mocket.Catcher.NewMock().WithQuery(query).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "success: return empty list if no dinosaur requests available for user",
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
				dinosaurList: dbapi.DinosaurList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: false,
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset()

				// total count query
				totalCountResponse := []map[string]interface{}{{"count": len(dinosaurList)}}
				mocket.Catcher.NewMock().WithQuery(`SELECT count(1) FROM "dinosaur_requests"`).WithReply(totalCountResponse)

				// actual query to return list of dinosaur requests based on filters
				query := fmt.Sprintf(`SELECT * FROM "%s"`, dinosaurRequestTableName)
				response := converters.ConvertDinosaurRequestList(dinosaurList)

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
				dinosaurList: nil,
				pagingMeta:   nil,
			},
			wantErr: true,
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset()

				totalCountResponse := []map[string]interface{}{{"count": len(dinosaurList)}}
				mocket.Catcher.NewMock().WithQuery("SELECT count(1) FROM \"dinosaur_requests\"").WithReply(totalCountResponse)

				query := fmt.Sprintf(`SELECT * FROM "%s"`, dinosaurRequestTableName)
				response := converters.ConvertDinosaurRequestList(dinosaurList)

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
				dinosaurList: dbapi.DinosaurList{},
				pagingMeta: &api.PagingMeta{
					Page:  1,
					Size:  0,
					Total: 0,
				},
			},
			wantErr: true,
			setupFn: func(dinosaurList dbapi.DinosaurList) {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn(tt.want.dinosaurList)
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}

			result, pagingMeta, err := k.List(tt.args.ctx, tt.args.listArgs)

			// check errors
			if (err != nil) != tt.wantErr {
				t.Errorf("dinosaurService.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// compare wanted vs actual pagingMeta result
			if !reflect.DeepEqual(pagingMeta, tt.want.pagingMeta) {
				t.Errorf("dinosaur.Service.List(): Paging meta returned is not correct:\n\tgot: %+v\n\twant: %+v", pagingMeta, tt.want.pagingMeta)
			}

			// compare wanted vs actual results
			if len(result) != len(tt.want.dinosaurList) {
				t.Errorf("dinosaur.Service.List(): total number of results: got = %d want = %d", len(result), len(tt.want.dinosaurList))
			}

			for i, got := range result {
				if !reflect.DeepEqual(got, tt.want.dinosaurList[i]) {
					t.Errorf("dinosaurService.List():\ngot = %+v\nwant = %+v", got, tt.want.dinosaurList[i])
				}
			}
		})
	}
}

func Test_dinosaurService_ListByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		status constants2.DinosaurStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*dbapi.DinosaurRequest
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
			want: []*dbapi.DinosaurRequest{buildDinosaurRequest(nil)},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE status IN ($1)`).
					WithArgs("").
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			got, err := k.ListByStatus(tt.args.status)
			// check errors
			if (err != nil) != tt.wantErr {
				t.Errorf("dinosaurService.ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListByStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dinosaurService_UpdateStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		id     string
		status constants2.DinosaurStatus
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
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
						dinosaurRequest.Status = constants2.DinosaurRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants2.DinosaurRequestStatusPreparing,
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "status"=$1`)
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
						dinosaurRequest.Status = constants2.DinosaurRequestStatusDeprovision.String()
					})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id:     testID,
				status: constants2.DinosaurRequestStatusDeleting,
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
					WithQuery(`SELECT * FROM "dinosaur_requests" WHERE id = $1`).
					WithArgs(testID).
					WithReply(converters.ConvertDinosaurRequest(buildDinosaurRequest(func(dinosaurRequest *dbapi.DinosaurRequest) {
						dinosaurRequest.Status = constants2.DinosaurRequestStatusPreparing.String()
					})))
				mocket.Catcher.NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "status"=$1`)
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
			k := dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			executed, err := k.UpdateStatus(tt.args.id, tt.args.status)
			if executed != tt.wantExecuted {
				t.Error("dinosaurService.UpdateStatus() error = should have refused execution but didn't")
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("dinosaurService.UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_dinosaurService_Update(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
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
				dinosaurRequest: buildDinosaurRequest(nil),
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
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Update(tt.args.dinosaurRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("dinosaurService.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_dinosaurService_Updates(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		clusterService    ClusterService
	}
	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
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
				dinosaurRequest: buildDinosaurRequest(nil),
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
				dinosaurRequest: buildDinosaurRequest(nil),
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				clusterService:    tt.fields.clusterService,
				dinosaurConfig:    config.NewDinosaurConfig(),
				awsConfig:         config.NewAWSConfig(),
			}
			err := k.Updates(tt.args.dinosaurRequest, map[string]interface{}{
				"id":    "idsds",
				"owner": "",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("dinosaurService.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_dinosaurService_DeprovisionDinosaurForUsers(t *testing.T) {
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "status"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			gomega.RegisterTestingT(t)
			k := dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeprovisionDinosaurForUsers(tt.args.users)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_dinosaurService_DeprovisionExpiredDinosaurs(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	type args struct {
		dinosaurAgeInMins int
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "dinosaur_requests" SET "status"=$1,"updated_at"=$2 WHERE instance_type = $3 AND created_at  <=  $4 AND status NOT IN ($5,$6)`)
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
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
				dinosaurConfig:    config.NewDinosaurConfig(),
			}
			err := k.DeprovisionExpiredDinosaurs(tt.args.dinosaurAgeInMins)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestDinosaurService_CountByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status []constants2.DinosaurStatus
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		want      []DinosaurStatusCount
		setupFunc func()
	}{
		{
			name:   "should return the counts of Dinosaurs in different status",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []constants2.DinosaurStatus{constants2.DinosaurRequestStatusAccepted, constants2.DinosaurRequestStatusReady, constants2.DinosaurRequestStatusProvisioning},
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
					WithQuery(`SELECT status as Status, count(1) as Count FROM "dinosaur_requests" WHERE status in ($1,$2,$3)`).
					WithArgs(constants2.DinosaurRequestStatusAccepted.String(), constants2.DinosaurRequestStatusReady.String(), constants2.DinosaurRequestStatusProvisioning.String()).
					WithReply(counters)
			},
			want: []DinosaurStatusCount{{
				Status: constants2.DinosaurRequestStatusAccepted,
				Count:  2,
			}, {
				Status: constants2.DinosaurRequestStatusReady,
				Count:  1,
			}, {
				Status: constants2.DinosaurRequestStatusProvisioning,
				Count:  0,
			}},
		},
		{
			name:   "should return error",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []constants2.DinosaurStatus{constants2.DinosaurRequestStatusAccepted, constants2.DinosaurRequestStatusReady},
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
			}
			status, err := k.CountByStatus(tt.args.status)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for CountByStatus: %v", err)
			}
			if !reflect.DeepEqual(status, tt.want) {
				t.Errorf("CountByStatus want = %v, got = %v", tt.want, status)
			}
		})
	}
}

func TestDinosaurService_ChangeDinosaurCNAMErecords(t *testing.T) {
	type fields struct {
		awsClient aws.Client
	}

	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
		action          DinosaurRoutesAction
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should create CNAMEs for dinosaur",
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
				dinosaurRequest: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						ID: "test-dinosaur-id",
					},
					Name:   "test-dinosaur-cname",
					Routes: []byte("[{\"domain\": \"test-dinosaur-id.example.com\", \"router\": \"test-dinosaur-id.rhcloud.com\"}]"),
					Region: "us-east-1",
				},
				action: DinosaurRoutesActionCreate,
			},
		},
		{
			name: "should delete CNAMEs for dinosaur",
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
				dinosaurRequest: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						ID: "test-dinosaur-id",
					},
					Name:   "test-dinosaur-cname",
					Routes: []byte("[{\"domain\": \"test-dinosaur-id.example.com\", \"router\": \"test-dinosaur-id.rhcloud.com\"}]"),
					Region: "us-east-1",
				},
				action: DinosaurRoutesActionDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dinosaurService := &dinosaurService{
				awsClientFactory: aws.NewMockClientFactory(tt.fields.awsClient),
				awsConfig: &config.AWSConfig{
					Route53AccessKey:       "test-route-53-key",
					Route53SecretAccessKey: "test-route-53-secret-key",
				},
				dinosaurConfig: &config.DinosaurConfig{
					DinosaurDomainName: "rhcloud.com",
				},
			}

			_, err := dinosaurService.ChangeDinosaurCNAMErecords(tt.args.dinosaurRequest, tt.args.action)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error for ChangeDinosaurCNAMErecords %v", err)
			}
		})
	}

}

func TestDinosaurService_ListComponentVersions(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		want      []DinosaurComponentVersions
		setupFunc func()
	}{
		{
			name:    "should return the component versions for Dinosaur",
			fields:  fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			wantErr: false,
			setupFunc: func() {
				versions := []map[string]interface{}{
					{
						"id":                       "1",
						"cluster_id":               "cluster1",
						"desired_strimzi_version":  "1.0.1",
						"actual_strimzi_version":   "1.0.0",
						"strimzi_upgrading":        true,
						"desired_dinosaur_version": "2.0.1",
						"actual_dinosaur_version":  "2.0.0",
						"dinosaur_upgrading":       false,
					},
					{
						"id":                       "2",
						"cluster_id":               "cluster2",
						"desired_strimzi_version":  "1.0.1",
						"actual_strimzi_version":   "1.0.0",
						"strimzi_upgrading":        false,
						"desired_dinosaur_version": "2.0.1",
						"actual_dinosaur_version":  "2.0.0",
						"dinosaur_upgrading":       false,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT "id","cluster_id","desired_strimzi_version","actual_strimzi_version","strimzi_upgrading","desired_dinosaur_version","actual_dinosaur_version","dinosaur_upgrading"`).
					WithReply(versions)
			},
			want: []DinosaurComponentVersions{{
				ID:                     "1",
				ClusterID:              "cluster1",
				DesiredStrimziVersion:  "1.0.1",
				ActualStrimziVersion:   "1.0.0",
				StrimziUpgrading:       true,
				DesiredDinosaurVersion: "2.0.1",
				ActualDinosaurVersion:  "2.0.0",
				DinosaurUpgrading:      false,
			}, {
				ID:                     "2",
				ClusterID:              "cluster2",
				DesiredStrimziVersion:  "1.0.1",
				ActualStrimziVersion:   "1.0.0",
				StrimziUpgrading:       false,
				DesiredDinosaurVersion: "2.0.1",
				ActualDinosaurVersion:  "2.0.0",
				DinosaurUpgrading:      false,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			k := &dinosaurService{
				connectionFactory: tt.fields.connectionFactory,
			}
			result, err := k.ListComponentVersions()
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for ListComponentVersions: %v", err)
			}
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("ListComponentVersions want = %v, got = %v", tt.want, result)
			}
		})
	}
}
