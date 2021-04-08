package services

import (
	"errors"
	"fmt"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"reflect"
	"testing"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	dbConverters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
)

var (
	testRegion   = "us-west-1"
	testProvider = "aws"
	testDNS      = "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"
	testMultiAZ  = true
	testStatus   = api.ClusterProvisioned
)

// build a test cluster
func buildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
	}
	if modifyFn != nil {
		modifyFn(cluster)
	}
	return cluster
}

func Test_Cluster_Create(t *testing.T) {
	awsConfig := &config.AWSConfig{
		AccountID:       "dummy",
		AccessKey:       "dummy",
		SecretAccessKey: "dummy",
	}
	wantedCluster, _ := v1.NewCluster().Build()

	type fields struct {
		connectionFactory *db.ConnectionFactory
		ocmClient         ocm.Client
		awsConfig         *config.AWSConfig
		clusterBuilder    ocm.ClusterBuilder
	}
	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *v1.Cluster
	}{
		{
			name: "successful cluster creation from cluster request job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(Cluster *v1.Cluster) (*v1.Cluster, error) {
						newCluster, _ := v1.NewCluster().Build()
						return newCluster, nil
					},
				},
				awsConfig: awsConfig,
				clusterBuilder: &ocm.ClusterBuilderMock{
					NewOCMClusterFromClusterFunc: func(cluster *api.Cluster) (*v1.Cluster, error) {
						newCluster, _ := v1.NewCluster().Build()
						return newCluster, nil
					},
				},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: false,
			want:    wantedCluster,
		},
		{
			name: "successful cluster creation without cluster request job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(Cluster *v1.Cluster) (*v1.Cluster, error) {
						newCluster, _ := v1.NewCluster().Build()
						return newCluster, nil
					},
				},
				awsConfig: awsConfig,
				clusterBuilder: &ocm.ClusterBuilderMock{
					NewOCMClusterFromClusterFunc: func(cluster *api.Cluster) (*v1.Cluster, error) {
						newCluster, _ := v1.NewCluster().Build()
						return newCluster, nil
					},
				},
			},
			args: args{
				cluster: buildCluster(func(cluster *api.Cluster) {
					cluster.ID = ""
				}),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("INSERT").WithReply(nil)
				mocket.Catcher.NewMock().WithExecException()
			},
			wantErr: false,
			want:    wantedCluster,
		},
		{
			name: "NewOCMClusterFromCluster failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterBuilder: &ocm.ClusterBuilderMock{
					NewOCMClusterFromClusterFunc: func(cluster *api.Cluster) (*v1.Cluster, error) {
						return nil, errors.New("test NewOCMClusterFromCluster cluster creation failure")
					},
				},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			wantErr: true,
		},
		{
			name: "CreateCluster failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(Cluster *v1.Cluster) (*v1.Cluster, error) {
						return nil, errors.New("CreateCluster failure")
					},
				},
				awsConfig: awsConfig,
				clusterBuilder: &ocm.ClusterBuilderMock{
					NewOCMClusterFromClusterFunc: func(cluster *api.Cluster) (*v1.Cluster, error) {
						newCluster, _ := v1.NewCluster().Build()
						return newCluster, nil
					},
				},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: true,
		},
		{
			name: "Database error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(Cluster *v1.Cluster) (*v1.Cluster, error) {
						return nil, errors.New("CreateCluster failure due to db connection issue")
					},
				},
				awsConfig: awsConfig,
				clusterBuilder: &ocm.ClusterBuilderMock{
					NewOCMClusterFromClusterFunc: func(cluster *api.Cluster) (*v1.Cluster, error) {
						return nil, errors.New("NewOCMClusterFromCluster failure due to db connection issue")
					},
				},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				ocmClient:         tt.fields.ocmClient,
				awsConfig:         tt.fields.awsConfig,
				clusterBuilder:    tt.fields.clusterBuilder,
			}

			got, err := c.Create(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_GetClusterDNS(t *testing.T) {
	mockClusterDNS := testDNS

	type fields struct {
		connectionFactory *db.ConnectionFactory
		ocmClient         ocm.Client
		awsConfig         *config.AWSConfig
		clusterBuilder    ocm.ClusterBuilder
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    string
	}{
		{
			name: "successful retrieval of clusterDNS",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return mockClusterDNS, nil
					},
				},
			},
			args: args{
				clusterID: testClusterID,
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithReply(nil)
			},
			wantErr: false,
			want:    mockClusterDNS,
		},
		{
			name: "error when passing empty clusterID",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return "", errors.New("ClusterID cannot be empty")
					},
				},
			},
			args: args{
				clusterID: "",
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

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				ocmClient:         tt.fields.ocmClient,
				awsConfig:         tt.fields.awsConfig,
				clusterBuilder:    tt.fields.clusterBuilder,
			}

			got, err := c.GetClusterDNS(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClusterDNS() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if got == "" && !tt.wantErr {
				t.Errorf("GetClusterDNS() error - expecting non-empty cluster DNS here, got '%v'", got)
			}
		})
	}
}

func Test_Cluster_FindClusterByID(t *testing.T) {
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
		want    *api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: "non-existentID",
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "error when id is empty (undefined)",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: "",
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: testClusterID,
			},
			wantErr: true,
			want:    nil,
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
				id: testClusterID,
			},
			want: &api.Cluster{ClusterID: testClusterID},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"cluster_id": testClusterID}}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(mockedResponse)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindClusterByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindClusterByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindClusterByID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_FindCluster(t *testing.T) {
	clusterDetails := FindClusterCriteria{
		Provider: testProvider,
		Region:   testRegion,
		MultiAZ:  testMultiAZ,
		Status:   testStatus,
	}

	nonExistentClusterDetails := FindClusterCriteria{
		Provider: "nonExistentProvider",
		Region:   testRegion,
		MultiAZ:  testMultiAZ,
		Status:   testStatus,
	}

	type fields struct {
		connectionFactory *db.ConnectionFactory
		ocmClient         ocm.Client
		awsConfig         *config.AWSConfig
		clusterBuilder    ocm.ClusterBuilder
	}
	type args struct {
		criteria FindClusterCriteria
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "return nil if no cluster is found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: nonExistentClusterDetails,
			},
			want:    nil,
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(nil)
			},
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterDetails,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful retrieval of a cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterDetails,
			},
			want: buildCluster(nil),
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(dbConverters.ConvertCluster(buildCluster(nil)))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				ocmClient:         tt.fields.ocmClient,
				awsConfig:         tt.fields.awsConfig,
				clusterBuilder:    tt.fields.clusterBuilder,
			}

			got, err := c.FindCluster(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindCluster() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindCluster() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_ListByStatus(t *testing.T) {
	var nonEmptyClusterList = []api.Cluster{
		{
			CloudProvider: testProvider,
			MultiAZ:       testMultiAZ,
			Region:        testRegion,
			Status:        testStatus,
		},
	}
	emptyClusterList := []api.Cluster{}

	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when status is undefined",
			args: args{
				status: "",
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "success: return empty list of clusters with specified status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			want:    emptyClusterList,
			wantErr: false,
			setupFn: func() {
				response, err := dbConverters.ConvertClusterList(emptyClusterList)
				if err != nil {
					t.Errorf("Test_ListByStatus() failed to convert clusterList: %s", err.Error())
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(response)
			},
		},
		{
			name: "success: return non-empty list of clusters with specified status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			want:    nonEmptyClusterList,
			wantErr: false,
			setupFn: func() {
				response, err := dbConverters.ConvertClusterList(nonEmptyClusterList)
				if err != nil {
					t.Errorf("Test_ListByStatus() failed to convert clusterList: %s", err.Error())
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(response)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListByStatus(tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListByStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		cluster api.Cluster
		status  api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when status is undefined",
			args: args{
				status: "",
			},
			wantErr: true,
		},
		{
			name: "error when id is undefined",
			args: args{
				cluster: api.Cluster{},
				status:  testStatus,
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "successful status update by id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status:  testStatus,
				cluster: api.Cluster{Meta: api.Meta{ID: testID}},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("WHERE (id =").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("UPDATE").WithReply(nil)
			},
		},
		{
			name: "successful status update by ClusterId",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status:  testStatus,
				cluster: api.Cluster{ClusterID: testID},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("WHERE (cluster_id =").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("UPDATE").WithReply(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.UpdateStatus(tt.args.cluster, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_RegisterClusterJob(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterRequest api.Cluster
		status         api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "success registering a new job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: "",
				clusterRequest: api.Cluster{
					CloudProvider: "",
					ClusterID:     "",
					ExternalID:    "",
					MultiAZ:       false,
					Region:        "",
					BYOC:          false,
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`INSERT  INTO "clusters"`).WithReply(nil)
				mocket.Catcher.NewMock().WithExecException() // Fail on anything else
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}

			err := k.RegisterClusterJob(&tt.args.clusterRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterClusterJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_ScaleUpComputeNodes(t *testing.T) {
	testNodeIncrement := 3
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when cluster id is undefined",
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
		{
			name: "error when scale up compute nodes ocm function fails",
			args: args{
				clusterID: "test",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleUpComputeNodesFunc: func(clusterID string, increment int) (*v1.Cluster, error) {
						return nil, errors.New("test ScaleUpComputeNodes failure")
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				ocmClient: tt.fields.ocmClient,
			}
			_, err := k.ScaleUpComputeNodes(tt.args.clusterID, testNodeIncrement)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScaleUpComputeNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_ScaleDownComputeNodes(t *testing.T) {
	testNodeDecrement := 3
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when cluster id is undefined",
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
		{
			name: "error when scale down ocm function fails",
			args: args{
				clusterID: "test",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleDownComputeNodesFunc: func(clusterID string, decrement int) (*v1.Cluster, error) {
						return nil, errors.New("test ScaleDownComputeNodes failure")
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				ocmClient: tt.fields.ocmClient,
			}
			_, err := k.ScaleDownComputeNodes(tt.args.clusterID, testNodeDecrement)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScaleDownComputeNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClusterService_ListGroupByProviderAndRegion(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		providers         []string
		regions           []string
		status            []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []*ResGroupCPRegion
		setupFn func()
	}{
		{
			name: "ListGroupByProviderAndRegion success",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providers:         []string{"aws"},
				regions:           []string{"us-east-1"},
				status:            api.StatusForValidCluster,
			},
			want:    []*ResGroupCPRegion{},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(nil)
			},
		},
		{
			name: "ListGroupByProviderAndRegion failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providers:         []string{"aws"},
				regions:           []string{"us-east-1"},
				status:            api.StatusForValidCluster,
			},
			want:    nil,
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListGroupByProviderAndRegion(tt.fields.providers, tt.fields.regions, tt.fields.status)
			if err != nil && !tt.wantErr {
				t.Errorf("ListGroupByProviderAndRegion err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListGroupByProviderAndRegion got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_AddIdentityProviderID(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		id                 string
		identityProviderId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id:                 "12345",
				identityProviderId: "foobar",
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithError(fmt.Errorf("some error"))
			},
		},
		{
			name: "successfully adds an identity provider ID",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id:                 "12345",
				identityProviderId: "foobar",
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("WHERE (id =").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("UPDATE").WithReply(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.AddIdentityProviderID(tt.args.id, tt.args.identityProviderId)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_DeleteByClusterId(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithError(fmt.Errorf("database error"))
			},
		},
		{
			name: "successful soft delete cluster by id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterID: "123",
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("WHERE (cluster_id =").WithReply(nil)
				mocket.Catcher.NewMock().WithQuery("UPDATE").WithReply(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeleteByClusterID(tt.args.clusterID)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_Cluster_FindNonEmptyClusterById(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterId: "non-existentID",
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterId: testClusterID,
			},
			wantErr: true,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithError(fmt.Errorf("some database error"))
			},
		},
		{
			name: "successful find the cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterId: testClusterID,
			},
			want: &api.Cluster{ClusterID: testClusterID},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"cluster_id": testClusterID}}
				query := `SELECT * FROM "clusters"  WHERE "clusters"."deleted_at" IS NULL AND (("clusters"."cluster_id" = test-cluster-id) AND (cluster_id IN (SELECT "kafka_requests"."cluster_id" FROM kafka_requests WHERE "kafka_requests"."status" != 'failed' AND "kafka_requests"."deleted_at" IS NULL AND "kafka_requests"."cluster_id" != ''))) ORDER BY "clusters"."id" ASC LIMIT 1`
				mocket.Catcher.Reset().NewMock().WithQuery(query).WithReply(mockedResponse)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindNonEmptyClusterById(tt.args.clusterId)
			gomega.Expect(got).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_ListAllClusterIds(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	emptyResult := []api.Cluster{}
	var clusters []api.Cluster
	clusters = append(clusters, api.Cluster{ClusterID: "test01"})

	tests := []struct {
		name    string
		fields  fields
		want    []api.Cluster
		setupFn func()
		want1   *apiErrors.ServiceError
	}{
		{
			name: "Empty cluster Ids",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			want:  emptyResult,
			want1: nil,
		},
		{
			name: "List All cluster id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				var res []api.Cluster
				c := api.Cluster{
					ClusterID: "test01",
				}
				res = append(res, c)
				covRest, _ := dbConverters.ConvertClusterList(res)
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(covRest)
			},
			want:  clusters,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, got1 := c.ListAllClusterIds()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListAllClusterIds() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ListAllClusterIds() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_clusterService_FindKafkaInstanceCount(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterID []string
	}
	var testRes []*ResKafkaInstanceCount
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*ResKafkaInstanceCount
		wantErr bool
		setupFn func()
	}{
		{
			name: "Instance count equals to 0",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				[]string{"test01"},
			},
			want:    []*ResKafkaInstanceCount{}, //zero valued
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset()
			},
		},
		{
			name: "Instance count with exception",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				[]string{},
			},
			want:    testRes,
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
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindKafkaInstanceCount(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindKafkaInstanceCount() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindKafkaInstanceCount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clusterService_FindAllClusters(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	clusterReady := FindClusterCriteria{
		Provider: "test-provider",
		Region:   "us-east",
		MultiAZ:  true,
		Status:   api.ClusterReady,
	}
	emptyResult := []*api.Cluster{}
	var clusters []*api.Cluster
	clusters = append(clusters, &api.Cluster{ClusterID: "test01", Status: api.ClusterReady})

	type args struct {
		criteria FindClusterCriteria
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*api.Cluster
		setupFn func()
		wantErr bool
	}{
		{
			name: "Find all cluster with empty result",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterReady,
			},
			want:    emptyResult,
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset()
			},
		},
		{
			name: "List All cluster id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				var res []api.Cluster
				c := api.Cluster{
					ClusterID: "test01",
					Status:    api.ClusterReady,
				}
				res = append(res, c)
				covRest, _ := dbConverters.ConvertClusterList(res)
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(covRest)
			},
			args: args{
				criteria: clusterReady,
			},
			want:    clusters,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindAllClusters(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAllClusters() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindAllClusters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clusterService_UpdateMultiClusterStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterIds []string
		status     api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: false,
		},
		{
			name: "error when ids is undefined",
			args: args{
				clusterIds: []string{},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: true,
		},
		{
			name: "error when status is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			if err := c.UpdateMultiClusterStatus(tt.args.clusterIds, tt.args.status); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMultiClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
