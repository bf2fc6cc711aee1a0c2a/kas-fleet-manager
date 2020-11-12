package services

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	dbConverters "gitlab.cee.redhat.com/service/managed-services-api/pkg/db/converters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
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
			got, err := k.Get(tt.args.id)
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

func Test_kafkaService_Create(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
		clusterService    ClusterService
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
					CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
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
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
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
		{
			name: "failed clusterDNS retrieval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorBadRequest, "")
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
		{
			name: "validate BootstrapServerHost truncate",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					CreateFunc: func(syncsetBuilder *v1.SyncsetBuilder, syncsetId string, clusterId string) (*v1.Syncset, *errors.ServiceError) {
						syncset, _ := cmv1.NewSyncset().ID(testSyncsetID).Build()
						return syncset, nil
					},
				},
				clusterService: &ClusterServiceMock{
					GetClusterDNSFunc: func(string) (string, *errors.ServiceError) {
						return "clusterDNS", nil
					},
				},
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

func Test_kafkaService_Delete(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		syncsetService    SyncsetService
	}
	type args struct {
		id string
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
			name: "error when syncset deletion fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					DeleteFunc: func(syncsetId string, clusterId string) *errors.ServiceError {
						return errors.GeneralError("error deleting syncset")
					},
				},
			},
			args: args{
				id: testID,
			},
			wantErr: true,
		},
		{
			name: "successful output",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				syncsetService: &SyncsetServiceMock{
					DeleteFunc: func(syncsetId string, clusterId string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				id: testID,
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertKafkaRequest(&api.KafkaRequest{
					Meta: api.Meta{
						ID: testID,
					},
					Region:        clusterservicetest.MockClusterRegion,
					ClusterID:     clusterservicetest.MockClusterID,
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					MultiAZ:       false,
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
			}
			err := k.Delete(tt.args.id)
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
				clusterService:    nil,
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
				clusterService:    nil,
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
				clusterService:    tt.fields.clusterService,
			}

			if err := k.RegisterKafkaJob(tt.args.kafkaRequest); (err != nil) != tt.wantErr {
				t.Errorf("RegisterKafkaJob() error = %v, wantErr = %v", err, tt.wantErr)
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
				ctx: auth.SetUsernameContext(context.TODO(), testUser),
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
				ctx: auth.SetUsernameContext(context.TODO(), testUser),
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
				ctx: auth.SetUsernameContext(context.TODO(), testUser),
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
		status KafkaStatus
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
		status KafkaStatus
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
			}
			err := k.UpdateStatus(tt.args.id, tt.args.status)
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
			}
			err := k.Update(tt.args.kafkaRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("kafkaService.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
