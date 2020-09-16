package services

import (
	"reflect"
	"testing"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	dbConverters "gitlab.cee.redhat.com/service/managed-services-api/pkg/db/converters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

var (
	testKafkaRequestRegion   = "eu-west-1"
	testKafkaRequestProvider = "aws"
	testKafkaRequestName     = "test-cluster"
	testClusterID            = "test-cluster-id"
	testSyncsetID            = "test-syncset-id"
)

// build a test kafka request
func buildKafkaRequest(modifyFn func(kafkaRequest *api.KafkaRequest)) *api.KafkaRequest {
	kafkaRequest := &api.KafkaRequest{
		Region:        testKafkaRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testKafkaRequestProvider,
		Name:          testKafkaRequestName,
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
				id: "test",
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
				id: "test",
			},
			want: &api.KafkaRequest{
				Meta: api.Meta{
					ID: "test",
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(&api.KafkaRequest{
					Meta: api.Meta{
						ID: "test",
					},
					Region:        clusterservicetest.MockClusterRegion,
					ClusterID:     clusterservicetest.MockClusterID,
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					MultiAZ:       false,
				}))
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
			got, err := k.Get(tt.args.id)
			// in our test case we used 'wantErr' to define if we expect and error to be returned from the function or
			// not, now we test that expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOCMClusterFromCluster() error = %v, wantErr %v", err, tt.wantErr)
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

func Test_kafkaService_Create(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	type args struct {
		kafkaRequest *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
	}{
		{
			name: "successful syncset creation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
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
		{
			name: "failed syncset creation",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
						return nil, errors.New(errors.ErrorBadRequest, "")
					},
				},
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
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
				syncsetService:    tt.fields.syncsetService,
			}

			if err := k.Create(tt.args.kafkaRequest); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func Test_kafkaService_RegisterKafkaJob(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	type args struct {
		kafkaRequest *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
	}{
		{
			name: "registering kafka job succeeds",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("INSERT").WithReply(nil)
			},
			wantErr: false,
		},
		{
			name: "registering kafka job fails: postgres error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService:    nil,
			},
			args: args{
				kafkaRequest: buildKafkaRequest(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("INSERT").WithExecException()
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
				syncsetService:    tt.fields.syncsetService,
			}

			if err := k.RegisterKafkaJob(tt.args.kafkaRequest); (err != nil) != tt.wantErr {
				t.Errorf("RegisterKafkaJob() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
