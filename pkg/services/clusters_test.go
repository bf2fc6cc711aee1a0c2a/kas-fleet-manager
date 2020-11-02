package services

import (
	"errors"
	"reflect"
	"testing"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	dbConverters "gitlab.cee.redhat.com/service/managed-services-api/pkg/db/converters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

var (
	testRegion        = "us-west-1"
	testProvider      = "aws"
	testDNS           = "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"
	testCloudProvider = "aws"
	testMultiAZ       = true
	testStatus        = api.ClusterProvisioned
	clustersTableName = "clusters"
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

// build a test machine pool
func buildTestMachinePool(replicas int) *clustersmgmtv1.MachinePool {
	pool, err := clustersmgmtv1.NewMachinePool().
		ID(DefaultMachinePoolID).
		Replicas(replicas).
		InstanceType(DefaultInstanceType).
		Build()
	if err != nil {
		return nil
	}
	return pool
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
			name: "successful cluster creation",
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
		want    api.Cluster
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
				id: testClusterID,
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
				id: testClusterID,
			},
			want: api.Cluster{},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(nil)
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
		api.Cluster{
			CloudProvider: testProvider,
			MultiAZ:       testMultiAZ,
			Region:        testRegion,
			Status:        testStatus,
		},
	}
	emptyClusterList := []api.Cluster{}

	type fields struct {
		connectionFactory *db.ConnectionFactory
		ocmClient         ocm.Client
		awsConfig         *config.AWSConfig
		clusterBuilder    ocm.ClusterBuilder
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
					t.Errorf("Test_ListByStatus() failed to convert ClusterList: %s", err.Error())
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
					t.Errorf("Test_ListByStatus() failed to convert ClusterList: %s", err.Error())
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
		id     string
		status api.ClusterStatus
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
				id:     "",
				status: testStatus,
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
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful status update",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
				id:     testID,
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply(nil)
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
			err := k.UpdateStatus(tt.args.id, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_ScaleUpMachinePool(t *testing.T) {
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
			name: "error when scale up machine pool ocm function fails",
			args: args{
				clusterID: "test",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleUpMachinePoolFunc: func(clusterID string, poolID string) (*v1.MachinePool, error) {
						return nil, errors.New("test ScaleUpMachinePool failure")
					},
					MachinePoolExistsFunc: func(clusterID string, poolID string) (bool, error) {
						return true, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when create machine pool ocm function fails",
			args: args{
				clusterID: "test",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					MachinePoolExistsFunc: func(clusterID string, poolID string) (bool, error) {
						return false, nil
					},
					CreateMachinePoolFunc: func(clusterID string, poolID string, instanceType string, replicas int) (*v1.MachinePool, error) {
						return nil, errors.New("test CreateMachinePool failure")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when machine pool exists ocm function fails",
			args: args{
				clusterID: "test",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					MachinePoolExistsFunc: func(clusterID string, poolID string) (bool, error) {
						return false, errors.New("test MachinePoolExists failure")
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
			_, err := k.ScaleUpMachinePool(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScaleUpMachinePool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_ScaleDownMachinePool(t *testing.T) {
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
					ScaleDownMachinePoolFunc: func(clusterID string, poolID string) (*v1.MachinePool, error) {
						return nil, errors.New("test ScaleDownMachinePool failure")
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
			_, err := k.ScaleDownMachinePool(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScaleDownMachinePool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
