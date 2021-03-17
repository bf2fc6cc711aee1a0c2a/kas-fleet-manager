package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	dbConverters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/converters"
	mocket "github.com/selvatico/go-mocket"
)

var (
	testKafkaConnectorName = "test-connector"
	testConnectorId        = "test-connector-id"
	testConnectorTypeId    = "test-connector-type-id"
)

// build a test connector request
func buildConnector(modifyFn func(connector *api.Connector)) *api.Connector {
	resource := &api.Connector{
		Meta: api.Meta{
			ID: testConnectorId,
		},
		Region:          testKafkaRequestRegion,
		ClusterID:       testClusterID,
		ConnectorTypeId: testConnectorTypeId,
		KafkaID:         testID,
		CloudProvider:   testKafkaRequestProvider,
		Name:            testKafkaConnectorName,
		MultiAZ:         false,
		Owner:           testUser,
	}
	if modifyFn != nil {
		modifyFn(resource)
	}
	return resource
}

func Test_connectorsService_Get(t *testing.T) {
	// fields are the variables on the struct that we're testing, in this case connectorsService
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	// args are the variables that will be provided to the function we're testing, in this case it's just the id we
	// pass to connectorsService.Create
	type args struct {
		ctx context.Context
		kid string
		id  string
		tid string
	}

	// Create authenticated context
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
	authenticatedCtx := auth.SetTokenInContext(context.TODO(), jwt)

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
		want *api.Connector
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
				kid: "",
				id:  "",
				tid: "",
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
				kid: testID,
				id:  testConnectorId,
				tid: testConnectorTypeId,
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
				kid: testID,
				id:  testConnectorId,
				tid: testConnectorTypeId,
			},
			want: buildConnector(nil),
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertConnectors(buildConnector(nil)))
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
			// we're testing the connectorsService struct, so use the 'fields' to create one
			k := &connectorsService{
				connectionFactory: tt.fields.connectionFactory,
			}
			// we're testing the connectorsService.Get function so use the 'args' to provide arguments to the function
			got, err := k.Get(tt.args.ctx, tt.args.id, tt.args.tid)
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

func Test_connectorsService_Create(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		connector *api.Connector
	}
	tests := []struct {
		name                    string
		fields                  fields
		args                    args
		setupFn                 func()
		wantErr                 bool
		wantBootstrapServerHost string
	}{
		{
			name: "successful connector creation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				connector: buildConnector(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery("INSERT").WithReply(nil).
					WithQuery("SELECT").WithReply(dbConverters.ConvertConnectors(buildConnector(nil)))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			k := &connectorsService{
				connectionFactory: tt.fields.connectionFactory,
			}

			if err := k.Create(context.Background(), tt.args.connector); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
