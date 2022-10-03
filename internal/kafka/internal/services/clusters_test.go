package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	instanceTypesMocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"

	kafkaTypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"gorm.io/gorm"
)

var (
	testRegion   = "us-west-1"
	testProvider = "aws"
	testDNS      = "apps.mk-btq2d1h8d3b1.b3k3.s1.devshift.org"
	testMultiAZ  = true
	testStatus   = api.ClusterProvisioned
)

func checkClusterFields(this *api.Cluster, that *api.Cluster) bool {
	if this == that {
		return true
	}
	if this.ClusterID != that.ClusterID || this.ExternalID != that.ExternalID || this.Region != that.Region || this.MultiAZ != that.MultiAZ || this.ProviderType != that.ProviderType || this.Status != that.Status || this.CloudProvider != that.CloudProvider {
		return false
	}
	return true
}

func Test_Cluster_Create(t *testing.T) {
	testClusterInternalId := "test-cluster-id"
	testClusterExternalId := "test-cluster-external-id"
	wantedCluster := &api.Cluster{
		CloudProvider: testProvider,
		ClusterID:     testClusterInternalId,
		ExternalID:    testClusterExternalId,
		MultiAZ:       testMultiAZ,
		Region:        testRegion,
		Status:        api.ClusterProvisioning,
		ProviderType:  api.ClusterProviderOCM,
		ProviderSpec:  nil,
		ClusterSpec:   nil,
	}

	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
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
		want    *api.Cluster
	}{
		{
			name: "successful cluster creation from cluster request job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: testClusterInternalId,
								ExternalID: testClusterExternalId,
								Status:     api.ClusterProvisioning,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: mocks.BuildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`INSERT INTO "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want:    wantedCluster,
		},
		{
			name: "CreateCluster failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return nil, errors.New("CreateCluster failure")
						},
					}, nil
				}},
			},
			args: args{
				cluster: mocks.BuildCluster(nil),
			},
			wantErr: true,
		},
		{
			name: "Database error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: testClusterInternalId,
								ExternalID: testClusterExternalId,
								Status:     api.ClusterProvisioning,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: mocks.BuildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("INSERT").WithQueryException()
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: mocks.BuildCluster(nil),
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

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.Create(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !checkClusterFields(got, tt.want) {
				t.Errorf("Create() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_GetClusterDNS(t *testing.T) {
	mockClusterDNS := testDNS

	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
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
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
						return mockClusterDNS, nil
					}}, nil
				}},
			},
			args: args{
				clusterID: testClusterID,
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          "testid",
						"cluster_id":  "testid",
						"cluster_dns": "",
					},
				}
				mocket.Catcher.Reset().
					NewMock().WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(res)
			},
			wantErr: false,
			want:    mockClusterDNS,
		},
		{
			name: "error when passing empty clusterID",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
						return mockClusterDNS, nil
					}}, nil
				}},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cluster DNS cannot be obtained from OCM",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
						return "", errors.New("failed to get cluster DNS from OCM")
					}}, nil
				}},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				clusterID: "",
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

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
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
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
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
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindClusterByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindClusterByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
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

	externalID := "34a47b83-e09a-42cd-9332-08dd77b84d53"

	externalClusterIDcriteria := FindClusterCriteria{
		ExternalID: externalID,
	}

	type fields struct {
		connectionFactory *db.ConnectionFactory
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
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
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
			want: mocks.BuildCluster(nil),
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertCluster(mocks.BuildCluster(nil)))
			},
		},
		{
			name: "successful retrieval of a cluster when passing external ID",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: externalClusterIDcriteria,
			},
			want: mocks.BuildCluster(func(cluster *api.Cluster) {
				cluster.ExternalID = externalID
			}),
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertCluster(
					mocks.BuildCluster(func(cluster *api.Cluster) {
						cluster.ExternalID = externalID
					}),
				))
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

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}

			got, err := c.FindCluster(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindCluster() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
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
			Meta: api.Meta{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				DeletedAt: gorm.DeletedAt{Valid: true},
			},
		},
	}

	var emptyClusterList []api.Cluster

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
				response := converters.ConvertClusterList(emptyClusterList)
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
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
				response := converters.ConvertClusterList(nonEmptyClusterList)
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE status = $1`).
					WithArgs(api.ClusterProvisioned.String()).
					WithReply(response)
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
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListByStatus(tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_Update(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		cluster api.Cluster
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
			name: "error when id is undefined",
			args: args{
				cluster: api.Cluster{},
			},
			wantErr: true,
		},
		{
			name: "error when database update returns an error",
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
				cluster: api.Cluster{Meta: api.Meta{ID: testID}},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "id"=$1,"updated_at"=$2 WHERE "clusters"."deleted_at" IS NULL AND "id" = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.Update(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "status"=$1,"updated_at"=$2 WHERE id = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successful status update by ClusterID",
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "status"=$1,"updated_at"=$2 WHERE cluster_id = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

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
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`INSERT INTO "clusters"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

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

func Test_ListEnterpriseClustersOfAnOrganization(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount(testUser, "", "", "13640203")
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	ctx := context.TODO()
	authenticatedCtx := auth.SetTokenInContext(ctx, jwt)

	clusterID := "npnbplmrku0bgnzj82h5uszyxwbetdwd"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*api.Cluster
		wantErr bool
		setupFn func([]*api.Cluster)
	}{
		{
			name: "should successfully list enterprise clusters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				ctx: authenticatedCtx,
			},
			wantErr: false,
			want: []*api.Cluster{
				{
					ClusterID:                     clusterID,
					Status:                        api.ClusterReady,
					MultiAZ:                       true,
					AccessKafkasViaPrivateNetwork: true,
					CloudProvider:                 "some-provider",
					Region:                        "some-region",
				},
			},
			setupFn: func(clusters []*api.Cluster) {
				mocket.Catcher.Reset()

				response := []map[string]interface{}{
					{
						"cluster_id":                        clusterID,
						"status":                            api.ClusterReady,
						"access_kafkas_via_private_network": true,
						"multi_az":                          true,
						"cloud_provider":                    "some-provider",
						"region":                            "some-region",
					},
				}

				query := `SELECT id, cluster_id, status, dynamic_capacity_info, access_kafkas_via_private_network, cloud_provider, region, multi_az FROM "clusters" WHERE (organization_id = $1 AND cluster_type = $2) AND "clusters"."deleted_at" IS NULL`

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
			},
			want:    nil,
			wantErr: true,
			setupFn: func(clusters []*api.Cluster) {
				mocket.Catcher.Reset()

				response := []map[string]interface{}{}

				query := `SELECT id, cluster_id, status, dynamic_capacity_info, access_kafkas_via_private_network, cloud_provider, region, multi_az FROM "clusters" WHERE (organization_id = $1 AND cluster_type = $2) AND "clusters"."deleted_at" IS NULL`

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
			},
			want:    nil,
			wantErr: true,
			setupFn: func(clusters []*api.Cluster) {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn(tt.want)
			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}

			result, err := c.ListEnterpriseClustersOfAnOrganization(tt.args.ctx)

			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			g.Expect(len(result)).To(gomega.Equal(len(tt.want)))

			for i, got := range result {
				g.Expect(got).To(gomega.Equal(tt.want[i]))
			}
		})
	}
}

func Test_clusterService_ListGroupByProviderAndRegion(t *testing.T) {
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
			want: []*ResGroupCPRegion{
				{
					Provider: "aws",
					Region:   "east-1",
					Count:    1,
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`FROM "clusters"`).WithReply([]map[string]interface{}{
					{
						"Provider": "aws",
						"Region":   "east-1",
						"Count":    1,
					},
				})
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListGroupByProviderAndRegion(tt.fields.providers, tt.fields.regions, tt.fields.status)
			if err != nil && !tt.wantErr {
				t.Errorf("ListGroupByProviderAndRegion err = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
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
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
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
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeleteByClusterID(tt.args.clusterID)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_Cluster_FindNonEmptyClusterByID(t *testing.T) {
	now := time.Now()

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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
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
			want: &api.Cluster{ClusterID: testClusterID, Meta: api.Meta{CreatedAt: now, UpdatedAt: now, DeletedAt: gorm.DeletedAt{Valid: true}}},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"cluster_id": testClusterID, "created_at": now, "updated_at": now, "deleted_at": gorm.DeletedAt{Valid: true}.Time}}
				query := `SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1 AND cluster_id IN (SELECT "cluster_id" FROM "kafka_requests" WHERE (status != $2 AND cluster_id = $3) AND "kafka_requests"."deleted_at" IS NULL) AND "clusters"."deleted_at" IS NULL ORDER BY "clusters"."id" LIMIT 1%`
				mocket.Catcher.Reset().NewMock().WithQuery(query).WithReply(mockedResponse)
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
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindNonEmptyClusterByID(tt.args.clusterId)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_ListNonEnterpriseClusterIDs(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	var clusters []api.Cluster
	clusters = append(clusters, api.Cluster{ClusterID: "test01"})

	tests := []struct {
		name    string
		fields  fields
		want    []api.Cluster
		setupFn func()
		wantErr *apiErrors.ServiceError
	}{
		{
			name: "Empty cluster Ids",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT "cluster_id" FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "List All cluster id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT "cluster_id" FROM "clusters" WHERE cluster_id != ''`).WithReply([]map[string]interface{}{
					{
						"cluster_id": "test01",
					},
				})
			},
			want:    clusters,
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.ListNonEnterpriseClusterIDs()
			if err != nil && err != tt.wantErr {
				t.Errorf("ListNonEnterpriseClusterIDs() err = %v, want %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
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
	var testRes []ResKafkaInstanceCount
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []ResKafkaInstanceCount
		wantErr bool
		setupFn func()
	}{
		{
			name: "Instance count equals to 2",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				[]string{},
			},
			want: []ResKafkaInstanceCount{
				{
					ClusterID: "test01",
					Count:     2,
				},
				{
					ClusterID: "test02",
					Count:     0,
				},
			},
			wantErr: false,
			setupFn: func() {
				counters := []map[string]interface{}{
					{
						"clusterid": "test01",
						"count":     2,
					},
				}
				mocket.Catcher.Reset().NewMock().WithQuery(`GROUP BY "cluster_id"`).WithReply(counters)
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
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
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
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindKafkaInstanceCount(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindKafkaInstanceCount() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			for i, res := range got {
				g.Expect(res).To(gomega.Equal(tt.want[i]))
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

	externalID := "34a47b83-e09a-42cd-9332-08dd77b84d53"

	externalClusterIDcriteria := FindClusterCriteria{
		ExternalID: externalID,
	}

	var clusters []*api.Cluster
	clusters = append(clusters, &api.Cluster{ClusterID: "test01", Status: api.ClusterReady, Meta: api.Meta{
		DeletedAt: gorm.DeletedAt{
			Valid: true,
		},
	}})

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
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "Find all clusters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertClusters(clusters))
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			args: args{
				criteria: clusterReady,
			},
			want:    clusters,
			wantErr: false,
		},
		{
			name: "successful retrieval of all clusters when passing external ID to the find criteria",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: externalClusterIDcriteria,
			},
			want: []*api.Cluster{mocks.BuildCluster(func(cluster *api.Cluster) {
				cluster.ExternalID = externalID
			})},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertClusters([]*api.Cluster{
					mocks.BuildCluster(func(cluster *api.Cluster) {
						cluster.ExternalID = externalID
					}),
				}))
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
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindAllClusters(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAllClusters() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			for i, res := range got {
				g.Expect(*res).To(gomega.Equal(*tt.want[i]))
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
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
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
	for _, testcase := range tests {
		tt := testcase

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

func Test_clusterService_CountByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status []api.ClusterStatus
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		want      []ClusterStatusCount
		setupFunc func()
	}{
		{
			name:   "should return the counts of clusters in different status",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []api.ClusterStatus{api.ClusterAccepted, api.ClusterReady, api.ClusterProvisioning},
			},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"status": "cluster_accepted",
						"count":  2,
					},
					{
						"status": "ready",
						"count":  1,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT status as Status, count(1) as Count FROM "clusters" WHERE status in ($1,$2,$3)`).
					WithArgs(api.ClusterAccepted.String(), api.ClusterReady.String(), api.ClusterProvisioning.String()).
					WithReply(counters)
			},
			want: []ClusterStatusCount{{
				Status: api.ClusterAccepted,
				Count:  2,
			}, {
				Status: api.ClusterReady,
				Count:  1,
			}, {
				Status: api.ClusterProvisioning,
				Count:  0,
			}},
		},
		{
			name:   "should return error",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []api.ClusterStatus{api.ClusterAccepted, api.ClusterReady},
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
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
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			status, err := c.CountByStatus(tt.args.status)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for CountByStatus: %v", err)
			}
			g.Expect(status).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_CheckClusterStatus(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "successfully check cluster status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CheckClusterStatusFunc: func(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: clusterId,
								ExternalID: clusterExternalId,
								Status:     api.ClusterProvisioned,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterId,
					},
					ExternalID: clusterExternalId,
					ClusterID:  clusterId,
					Status:     clusterStatus,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "id"=$1`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: &api.Cluster{
				Meta:       api.Meta{ID: clusterId},
				ClusterID:  clusterId,
				ExternalID: clusterExternalId,
				Status:     api.ClusterProvisioned,
			},
		},
		{
			name: "error when failed to check cluster status",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CheckClusterStatusFunc: func(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
							return nil, errors.Errorf("failed to get cluster status")
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterId,
					},
					ExternalID: clusterExternalId,
					ClusterID:  clusterId,
					Status:     clusterStatus,
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterId,
					},
					ExternalID: clusterExternalId,
					ClusterID:  clusterId,
					Status:     clusterStatus,
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.CheckClusterStatus(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckClusterStatus() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_RemoveClusterFromProvider(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ExternalID: clusterExternalId,
		ClusterID:  clusterId,
		Status:     clusterStatus,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    bool
	}{
		{
			name: "successfully delete cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						DeleteFunc: func(spec *types.ClusterSpec) (bool, error) {
							return true, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "error when failed to delete cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						DeleteFunc: func(spec *types.ClusterSpec) (bool, error) {
							return false, errors.Errorf("failed to delete cluster")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.Delete(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_ConfigureAndSaveIdentityProvider(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster          *api.Cluster
		identityProvider types.IdentityProviderInfo
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ExternalID: clusterExternalId,
		ClusterID:  clusterId,
		Status:     clusterStatus,
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
			name: "successfully configured identity provider",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						AddIdentityProviderFunc: func(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
							return &types.IdentityProviderInfo{
								OpenID: &types.OpenIDIdentityProviderInfo{
									ID: "test-id",
								},
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterId,
					},
					ExternalID: clusterExternalId,
					ClusterID:  clusterId,
					Status:     clusterStatus,
				},
				identityProvider: types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
					Name:         "test-name",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					Issuer:       "test-issuer",
				}},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: &api.Cluster{
				Meta: api.Meta{
					ID: clusterId,
				},
				ExternalID:         clusterExternalId,
				ClusterID:          clusterId,
				Status:             clusterStatus,
				IdentityProviderID: "test-id",
			},
		},
		{
			name: "error when failed to add identity provider",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						AddIdentityProviderFunc: func(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
							return nil, errors.Errorf("failed to add identity provider")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
				identityProvider: types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
					Name:         "test-name",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					Issuer:       "test-issuer",
				}},
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.ConfigureAndSaveIdentityProvider(tt.args.cluster, tt.args.identityProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigureAndSaveIdentityProvider() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_ApplyResources(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster   *api.Cluster
		resources types.ResourceSet
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ExternalID: clusterExternalId,
		ClusterID:  clusterId,
		Status:     clusterStatus,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
	}{
		{
			name: "successfully applied resources",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						ApplyResourcesFunc: func(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
							return nil, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
				resources: types.ResourceSet{
					Name:      "test-resources",
					Resources: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "error when failed to apply resources",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						ApplyResourcesFunc: func(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
							return nil, errors.Errorf("failed to apply resources")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: cluster,
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

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			err := c.ApplyResources(tt.args.cluster, tt.args.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyResources() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_clusterService_InstallStrimzi(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
		addonID string
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ExternalID:   clusterExternalId,
		ClusterID:    clusterId,
		Status:       clusterStatus,
		ProviderType: api.ClusterProviderOCM,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    bool
	}{
		{
			name: "successfully install strimzi",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{InstallStrimziFunc: func(clusterSpec *types.ClusterSpec) (bool, error) {
							return true, nil
						}}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-id",
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "error when failed to install strimzi",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							InstallStrimziFunc: func(clusterSpec *types.ClusterSpec) (bool, error) {
								return false, errors.Errorf("failed to install addon")
							}}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-addon-id",
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-id",
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.InstallStrimzi(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallStrimzi() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_ClusterLogging(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
		addonID string
	}

	clusterId := "test-internal-id"
	clusterExternalId := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ExternalID:   clusterExternalId,
		ClusterID:    clusterId,
		Status:       clusterStatus,
		ProviderType: api.ClusterProviderOCM,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    bool
	}{
		{
			name: "successfully install cluster logging",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							InstallClusterLoggingFunc: func(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
								return true, nil
							}}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-id",
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "error when failed to install cluster logging",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							InstallClusterLoggingFunc: func(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
								return false, errors.Errorf("failed to install addon")
							},
						}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-addon-id",
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-id",
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.InstallClusterLogging(tt.args.cluster, []types.Parameter{})
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallClusterLogging() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_GetExternalID(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
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
			name: "When cluster exists and external ID exists it is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"external_id": "test-cluster-id"}}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    "test-cluster-id",
		},
		{
			name: "When cluster exists and external ID does not exit the empty string is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"external_id": ""}}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    "",
		},
		{
			name: "When cluster does not exist an error is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
		},
		{
			name: "When provided clusterID is empty an error is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.GetExternalID(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExternalID() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_clusterService_GetClientID(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		clusterId string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		setupFn func()
	}{
		{
			name: "Should return the clusters client ID",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterId: testClusterID,
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          "testid",
						"cluster_id":  testClusterID,
						"cluster_dns": "",
						"client_id":   "ClientID",
					},
				}
				mocket.Catcher.Reset().
					NewMock().WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(res)
			},
			wantErr: false,
			want:    "ClientID",
		},
		{
			name: "An empty string is returned if the provided cluster is not found",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterId: "fake-cluster-id",
			},
			want:    "",
			wantErr: false,
			setupFn: func() {},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		if tt.setupFn != nil {
			tt.setupFn()
		}
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}
			got, err := c.GetClientID(tt.args.clusterId)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_CheckStrimziVersionReady(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		providerFactory   clusters.ProviderFactory
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
	type args struct {
		cluster        *api.Cluster
		strimziVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Should return true if strimziVersion is ready",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: "strimzi-cluster-operator.from-cluster",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Should return false if cannot get strimzi version",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: "fake-strimzi-version",
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.providerFactory,
			}
			got, err := c.CheckStrimziVersionReady(tt.args.cluster, tt.args.strimziVersion)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_IsStrimziKafkaVersionAvailableInCluster(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		providerFactory   clusters.ProviderFactory
	}
	type args struct {
		cluster        *api.Cluster
		strimziVersion string
		kafkaVersion   string
		ibpVersion     string
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

	false_availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   false,
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

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should return true if Strimzi kafka versions are available in cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: strimziOperatorVersion,
				kafkaVersion:   "2.7.0",
				ibpVersion:     "2.7",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Returns false if the provided kafka and kafka ibp versions are not available in the cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: strimziOperatorVersion,
				kafkaVersion:   "fake-kafka-version",
				ibpVersion:     "fake-ibpversion",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Returns false if kafka version is available in the cluster but the kafka ibp version is not available in the cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: strimziOperatorVersion,
				kafkaVersion:   "2.7.0",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Returns false if kafka ibp version is available in the cluster but the kafka version is not available in the cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster:        mockCluster,
				strimziVersion: strimziOperatorVersion,
				ibpVersion:     "2.7",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Returns false if kafka version and kafka ibp versions are in the cluster but its corresponding strimzi version is not ready",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   &clusters.ProviderFactoryMock{},
			},
			args: args{
				cluster: &api.Cluster{
					AvailableStrimziVersions: false_availableStrimziVersions,
				},
				strimziVersion: "fake-strimzi-version",
				kafkaVersion:   "2.7.0",
				ibpVersion:     "2.7",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.providerFactory,
			}
			got, err := c.IsStrimziKafkaVersionAvailableInCluster(tt.args.cluster, tt.args.strimziVersion, tt.args.kafkaVersion, tt.args.ibpVersion)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_FindStreamingUnitCountByClusterAndInstanceType(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	supportedInstanceTypeConfig := config.KafkaSupportedInstanceTypesConfig{
		Configuration: config.SupportedKafkaInstanceTypesConfig{
			SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
				{
					Id:          "standard",
					DisplayName: "Standard",
					Sizes: []config.KafkaInstanceSize{
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x1"
							kis.QuotaConsumed = 1
							kis.CapacityConsumed = 1
						}),
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x2"
							kis.QuotaConsumed = 2
							kis.CapacityConsumed = 2
						}),
					},
				},
				{
					Id:          "developer",
					DisplayName: "Trial",
					Sizes: []config.KafkaInstanceSize{
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x1"
							kis.QuotaConsumed = 1
							kis.CapacityConsumed = 1
						}),
					},
				},
			},
		},
	}

	testClusterID1 := "test-cluster-1"
	testClusterID2 := "test-cluster-2"

	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		want      KafkaStreamingUnitCountPerClusterList
		setupFunc func()
	}{
		{
			name: "return an error when clusters query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT cloud_provider, region, count(1) as Count, size_id, cluster_id, instance_type FROM "kafka_requests"`).
					WithReply([]map[string]interface{}{})

				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "clusters"`).
					WithQueryException().
					WithExecException()

				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want: nil,
		},
		{
			name: "return an error when kafkas query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT cloud_provider, region, count(1) as Count, size_id, cluster_id, instance_type FROM "kafka_requests"`).
					WithQueryException().
					WithExecException()

				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "clusters"`).
					WithReply([]map[string]interface{}{})

				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want: nil,
		},
		{
			name: "should return an empty list when there are no data plane clusters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			setupFunc: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT cloud_provider, region, count(1) as Count, size_id, cluster_id, instance_type FROM "kafka_requests"`).
					WithReply([]map[string]interface{}{})

				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "clusters"`).
					WithReply([]map[string]interface{}{})

				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want: KafkaStreamingUnitCountPerClusterList{},
		},
		{
			name: "should return the counts of Kafkas per region and instance type",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"region":         "us-east-1",
						"instance_type":  "standard",
						"cluster_id":     testClusterID1,
						"cloud_provider": testKafkaRequestProvider,
						"Count":          8,
						"SizeId":         "x1",
					},
					{
						"region":         "us-east-1",
						"instance_type":  "standard",
						"cluster_id":     testClusterID1,
						"cloud_provider": testKafkaRequestProvider,
						"Count":          2,
						"SizeId":         "x2",
					},
					{
						"region":         "eu-west-1",
						"instance_type":  "developer",
						"cluster_id":     testClusterID2,
						"cloud_provider": testKafkaRequestProvider,
						"Count":          1,
						"SizeId":         "x1",
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT cloud_provider, region, count(1) as Count, size_id, cluster_id, instance_type FROM "kafka_requests"`).
					WithReply(counters)

				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "clusters"`).
					WithReply([]map[string]interface{}{
						{
							"region":                  "us-east-1",
							"cloud_provider":          testKafkaRequestProvider,
							"cluster_id":              testClusterID1,
							"supported_instance_type": api.StandardTypeSupport.String(),
							"dynamic_capacity_info":   []byte(`{"standard":{"max_nodes":20,"max_units":20,"remaining_units":8}}`),
						},
						{
							"region":                  "eu-west-1",
							"cloud_provider":          testKafkaRequestProvider,
							"cluster_id":              testClusterID2,
							"supported_instance_type": api.DeveloperTypeSupport.String(),
							"dynamic_capacity_info":   []byte(`{"developer":{"max_nodes":1,"max_units":2,"remaining_units":1}}`),
						},
						{
							"region":                  "eu-west-2",
							"cloud_provider":          testKafkaRequestProvider,
							"cluster_id":              testClusterID2,
							"supported_instance_type": api.AllInstanceTypeSupport.String(),
							"dynamic_capacity_info":   []byte(`{}`), // set to empty to mimick cluster that do not have dynamic capacity info e.g during manual scaling
						},
					})

				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: KafkaStreamingUnitCountPerClusterList{
				{
					Region:        "us-east-1",
					InstanceType:  "standard",
					ClusterId:     testClusterID1,
					Count:         12,
					MaxUnits:      20,
					CloudProvider: "aws",
				},
				{
					Region:        "eu-west-1",
					InstanceType:  "developer",
					ClusterId:     testClusterID2,
					Count:         1,
					MaxUnits:      2,
					CloudProvider: "aws",
				},
				{
					Region:        "eu-west-2",
					InstanceType:  "standard",
					ClusterId:     testClusterID2,
					Count:         0,
					CloudProvider: "aws",
					MaxUnits:      0,
				},
				{
					Region:        "eu-west-2",
					InstanceType:  "developer",
					ClusterId:     testClusterID2,
					Count:         0,
					CloudProvider: "aws",
					MaxUnits:      0,
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(testing *testing.T) {
			g := gomega.NewWithT(testing)
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &supportedInstanceTypeConfig,
				},
			}
			streamingUnitsCountPerRegion, err := c.FindStreamingUnitCountByClusterAndInstanceType()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if !tt.wantErr {
				g.Expect(streamingUnitsCountPerRegion).To(gomega.Equal(tt.want))
			}

		})
	}
}

func Test_clusterService_ComputeConsumedStreamingUnitCountPerInstanceType(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	supportedInstanceTypeConfig := config.KafkaSupportedInstanceTypesConfig{
		Configuration: config.SupportedKafkaInstanceTypesConfig{
			SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
				{
					Id:          "standard",
					DisplayName: "Standard",
					Sizes: []config.KafkaInstanceSize{
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x1"
							kis.QuotaConsumed = 1
							kis.CapacityConsumed = 1
						}),
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x2"
							kis.QuotaConsumed = 2
							kis.CapacityConsumed = 2
						}),
					},
				},
				{
					Id:          "developer",
					DisplayName: "Trial",
					Sizes: []config.KafkaInstanceSize{
						*instanceTypesMocks.BuildKafkaInstanceSize(func(kis *config.KafkaInstanceSize) {
							kis.Id = "x1"
							kis.QuotaConsumed = 1
							kis.CapacityConsumed = 1
						}),
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		want      StreamingUnitCountPerInstanceType
		setupFunc func()
	}{
		{
			name: "return an error when kafkas query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT size_id, instance_type, count(1) as Count FROM "kafka_requests"`).
					WithQueryException().
					WithExecException()

				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want: nil,
		},
		{
			name: "should return an empty list when there are no data plane clusters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			setupFunc: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT size_id, instance_type, count(1) as Count FROM "kafka_requests"`).
					WithReply([]map[string]interface{}{})

				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want: StreamingUnitCountPerInstanceType{
				kafkaTypes.DEVELOPER: 0,
				kafkaTypes.STANDARD:  0,
			},
		},
		{
			name: "should return the counts of Kafkas per region and instance type",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"region":         "us-east-1",
						"instance_type":  "standard",
						"cloud_provider": testKafkaRequestProvider,
						"Count":          8,
						"SizeId":         "x1",
					},
					{
						"region":         "us-east-1",
						"instance_type":  "standard",
						"cloud_provider": testKafkaRequestProvider,
						"Count":          2,
						"SizeId":         "x2",
					},
					{
						"region":         "eu-west-1",
						"instance_type":  "developer",
						"cloud_provider": testKafkaRequestProvider,
						"Count":          1,
						"SizeId":         "x1",
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT size_id, instance_type, count(1) as Count FROM "kafka_requests"`).
					WithReply(counters)

				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: StreamingUnitCountPerInstanceType{
				kafkaTypes.DEVELOPER: 1,
				kafkaTypes.STANDARD:  12,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(testing *testing.T) {
			g := gomega.NewWithT(testing)
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &supportedInstanceTypeConfig,
				},
			}
			count, err := c.ComputeConsumedStreamingUnitCountPerInstanceType("some-cluster-id")
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if !tt.wantErr {
				g.Expect(count).To(gomega.Equal(tt.want))
			}

		})
	}
}
