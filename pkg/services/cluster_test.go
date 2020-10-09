package services

import (
	"errors"
	"reflect"
	"testing"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mocket "github.com/selvatico/go-mocket"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

var (
	testRegion   = "us-west-1"
	testProvider = "aws"
	testDNS      = "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"
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
