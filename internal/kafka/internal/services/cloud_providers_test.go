package services

import (
	"errors"
	"testing"

	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/cloud_providers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"

	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/patrickmn/go-cache"
	mocket "github.com/selvatico/go-mocket"
)

const (
	MockCloudProviderID_Azure          = "azure"
	MockCloudProviderName_Azure        = "azure"
	MockCloudProviderDisplayName_Azure = "Microsoft Azure"
	MockCloudProviderDisplayName_AWS   = "Amazon Web Services"
	MockRegionName_south               = "af-south-1"
	MockRegionName_east                = "us-east-1"
)

func Test_CloudProvider_ListCloudProviders(t *testing.T) {
	type fields struct {
		providerFactory   clusters.ProviderFactory
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		setupFn func()
		wantErr bool
		want    []api.CloudProvider
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   nil, // should not be called
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name: "fail to get cloud provider",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
							return nil, errors.New("GetCloudProviders fail to get the list of providers")
						},
						GetCloudProviderRegionsFunc: nil, // should not be called
					}, nil
				}},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT DISTINCT "provider_type" FROM "clusters" WHERE status NOT IN ($1,$2)`).
					WithArgs(api.ClusterCleanup.String(), api.ClusterDeprovisioning.String()).
					WithReply([]map[string]interface{}{{"provider_type": "standalone"}})
			},
			wantErr: true,
		},
		{
			name: "successful return an empty list of cloud providers when providers types list is empty",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   nil, // should not be called
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT DISTINCT "provider_type" FROM "clusters" WHERE status NOT IN ($1,$2)`).
					WithArgs(api.ClusterCleanup.String(), api.ClusterDeprovisioning.String()).
					WithReply([]map[string]interface{}{})
				// Both the caught query and any uncaught query will return an empty result. We have to ensure
				// we are not getting the result from an unexpected query
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    []api.CloudProvider{},
		},
		{
			name: "successful merge the various list of cloud providers from different providers and return it",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						if providerType == "ocm" {
							return &clusters.ProviderMock{
								GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
									return &types.CloudProviderInfoList{
										Items: []types.CloudProviderInfo{
											*mock.BuildCloudProviderInfoList(nil),
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: nil, // should not be called
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										*mock.BuildCloudProviderInfoList(func(CloudProviderInfo *types.CloudProviderInfo) {
											CloudProviderInfo.ID = MockCloudProviderID_Azure
											CloudProviderInfo.Name = MockCloudProviderName_Azure
											CloudProviderInfo.DisplayName = MockCloudProviderDisplayName_Azure
										}),
										*mock.BuildCloudProviderInfoList(nil),
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: nil, // should not be called
						}, nil
					}},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
			},
			wantErr: false,
			want: []api.CloudProvider{
				*mock.BuildApiCloudProvider(func(apiCloudProvider *api.CloudProvider) {
					apiCloudProvider.Kind = ""
					apiCloudProvider.Enabled = false
					apiCloudProvider.DisplayName = MockCloudProviderDisplayName_AWS
				}),
				*mock.BuildApiCloudProvider(func(apiCloudProvider *api.CloudProvider) {
					apiCloudProvider.Kind = ""
					apiCloudProvider.Enabled = false
					apiCloudProvider.Id = MockCloudProviderID_Azure
					apiCloudProvider.Name = MockCloudProviderName_Azure
					apiCloudProvider.DisplayName = MockCloudProviderDisplayName_Azure
				}),
			},
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			wantErr: true,
			setupFn: func() {},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.ListCloudProviders()
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_CachedCloudProviderWithRegions(t *testing.T) {
	type fields struct {
		providerFactory   clusters.ProviderFactory
		connectionFactory *db.ConnectionFactory
		cache             *cache.Cache
	}

	tests := []struct {
		name    string
		fields  fields
		setupFn func()
		wantErr bool
		want    []CloudProviderWithRegions
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				cache:             cache.New(5*time.Minute, 10*time.Minute),
				providerFactory:   nil, // should not be called
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name: "fail to get cloud provider regions",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
							return nil, errors.New("GetCloudProviders fail to get the list of providers")
						},
					}, nil
				}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "standalone"}})
			},
			wantErr: true,
		},
		{
			name: "successful return an empty list of regions when provider types list is empty",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory:   nil, // should not be called,
				cache:             cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT DISTINCT "provider_type" FROM "clusters" WHERE status NOT IN ($1,$2)`).
					WithArgs(api.ClusterCleanup.String(), api.ClusterDeprovisioning.String()).
					WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    []CloudProviderWithRegions{},
		},
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						if providerType == "ocm" {
							return &clusters.ProviderMock{
								GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
									return &types.CloudProviderInfoList{
										Items: []types.CloudProviderInfo{
											*mock.BuildCloudProviderInfoList(nil),
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
									return &types.CloudProviderRegionInfoList{
										Items: []types.CloudProviderRegionInfo{
											*mock.BuildCloudProviderRegionInfoList(nil),
										},
									}, nil
								},
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										*mock.BuildCloudProviderInfoList(func(CloudProviderInfo *types.CloudProviderInfo) {
											CloudProviderInfo.ID = MockCloudProviderID_Azure
											CloudProviderInfo.Name = MockCloudProviderName_Azure
											CloudProviderInfo.DisplayName = MockCloudProviderDisplayName_Azure
										}),
										*mock.BuildCloudProviderInfoList(nil),
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										*mock.BuildCloudProviderRegionInfoList(nil),
										*mock.BuildCloudProviderRegionInfoList(func(CloudProviderRegionInfo *types.CloudProviderRegionInfo) {
											CloudProviderRegionInfo.ID = MockRegionName_south
											CloudProviderRegionInfo.Name = MockRegionName_south
											CloudProviderRegionInfo.DisplayName = MockRegionName_south
										}),
									},
								}, nil
							},
						}, nil
					}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
			},
			wantErr: false,
			want: []CloudProviderWithRegions{
				{
					ID: "aws",
					RegionList: &types.CloudProviderRegionInfoList{
						Items: []types.CloudProviderRegionInfo{
							*mock.BuildCloudProviderRegionInfoList(nil),
							*mock.BuildCloudProviderRegionInfoList(func(CloudProviderRegionInfo *types.CloudProviderRegionInfo) {
								CloudProviderRegionInfo.ID = MockRegionName_south
								CloudProviderRegionInfo.Name = MockRegionName_south
								CloudProviderRegionInfo.DisplayName = MockRegionName_south
							}),
						},
					},
				},
				{
					ID: "azure",
					RegionList: &types.CloudProviderRegionInfoList{
						Items: []types.CloudProviderRegionInfo{
							*mock.BuildCloudProviderRegionInfoList(nil),
							*mock.BuildCloudProviderRegionInfoList(func(CloudProviderRegionInfo *types.CloudProviderRegionInfo) {
								CloudProviderRegionInfo.ID = MockRegionName_south
								CloudProviderRegionInfo.Name = MockRegionName_south
								CloudProviderRegionInfo.DisplayName = MockRegionName_south
							}),
						},
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				cache:             tt.fields.cache,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.GetCachedCloudProvidersWithRegions()
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ListCloudProviderRegions(t *testing.T) {
	type fields struct {
		providerFactory   clusters.ProviderFactory
		connectionFactory *db.ConnectionFactory
		cache             *cache.Cache
	}

	tests := []struct {
		name    string
		fields  fields
		setupFn func()
		wantErr bool
		want    []api.CloudRegion
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				cache:             cache.New(5*time.Minute, 10*time.Minute),
				providerFactory:   nil, // should not be called
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						if providerType == "ocm" {
							return &clusters.ProviderMock{
								GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
									return &types.CloudProviderInfoList{
										Items: []types.CloudProviderInfo{
											*mock.BuildCloudProviderInfoList(nil),
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
									return &types.CloudProviderRegionInfoList{
										Items: []types.CloudProviderRegionInfo{
											*mock.BuildCloudProviderRegionInfoList(nil),
										},
									}, nil
								},
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										*mock.BuildCloudProviderInfoList(func(CloudProviderInfo *types.CloudProviderInfo) {
											CloudProviderInfo.ID = MockCloudProviderID_Azure
											CloudProviderInfo.Name = MockCloudProviderName_Azure
											CloudProviderInfo.DisplayName = MockCloudProviderDisplayName_Azure
										}),
										*mock.BuildCloudProviderInfoList(nil),
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										*mock.BuildCloudProviderRegionInfoList(nil),
										*mock.BuildCloudProviderRegionInfoList(func(CloudProviderRegionInfo *types.CloudProviderRegionInfo) {
											CloudProviderRegionInfo.ID = MockRegionName_south
											CloudProviderRegionInfo.Name = MockRegionName_south
											CloudProviderRegionInfo.DisplayName = MockRegionName_south
										}),
									},
								}, nil
							},
						}, nil
					}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: []api.CloudRegion{
				*mock.BuildApiCloudRegion(func(apiCloudRegion *api.CloudRegion) {
					apiCloudRegion.Kind = ""
					apiCloudRegion.Enabled = false
					apiCloudRegion.CloudProvider = MockRegionName_east
				}),
				*mock.BuildApiCloudRegion(func(apiCloudRegion *api.CloudRegion) {
					apiCloudRegion.Kind = ""
					apiCloudRegion.DisplayName = MockRegionName_south
					apiCloudRegion.Id = MockRegionName_south
					apiCloudRegion.Enabled = false
					apiCloudRegion.CloudProvider = MockRegionName_east
				}),
			},
		},
		{
			name: "An error is returned when the cloud provider cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "An error is returned when the cloud regions cannot be obtained",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										*mock.BuildCloudProviderInfoList(nil),
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return nil, errors.New("failed to retrieve cloud regions")
							},
						}, nil
					},
				},
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				cache:             tt.fields.cache,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.ListCloudProviderRegions("aws")
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_cloudProvidersService_ListCachedCloudProviderRegions(t *testing.T) {
	type fields struct {
		providerFactory   clusters.ProviderFactory
		connectionFactory *db.ConnectionFactory
		cache             *cache.Cache
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []api.CloudRegion
		setupFn func()
		wantErr *svcErrors.ServiceError
	}{
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										*mock.BuildCloudProviderInfoList(func(CloudProviderInfo *types.CloudProviderInfo) {
											CloudProviderInfo.ID = MockCloudProviderID_Azure
											CloudProviderInfo.Name = MockCloudProviderName_Azure
											CloudProviderInfo.DisplayName = MockCloudProviderDisplayName_Azure
										}),
										*mock.BuildCloudProviderInfoList(nil),
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										*mock.BuildCloudProviderRegionInfoList(nil),
										*mock.BuildCloudProviderRegionInfoList(func(CloudProviderRegionInfo *types.CloudProviderRegionInfo) {
											CloudProviderRegionInfo.ID = MockRegionName_south
											CloudProviderRegionInfo.Name = MockRegionName_south
											CloudProviderRegionInfo.DisplayName = MockRegionName_south
										}),
									},
								}, nil
							},
						}, nil
					}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				id: "azure",
			},
			want: []api.CloudRegion{
				*mock.BuildApiCloudRegion(func(apiCloudRegion *api.CloudRegion) {
					apiCloudRegion.Kind = ""
					apiCloudRegion.Enabled = false
					apiCloudRegion.CloudProvider = "us-east-1"
				}),
				*mock.BuildApiCloudRegion(func(apiCloudRegion *api.CloudRegion) {
					apiCloudRegion.Kind = ""
					apiCloudRegion.DisplayName = MockRegionName_south
					apiCloudRegion.Id = MockRegionName_south
					apiCloudRegion.Enabled = false
					apiCloudRegion.CloudProvider = MockRegionName_east
				}),
			},
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		tt.setupFn()
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				connectionFactory: tt.fields.connectionFactory,
				cache:             tt.fields.cache,
			}
			got, err := p.ListCachedCloudProviderRegions(tt.args.id)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err).To(gomega.Equal(tt.wantErr))
		})
	}
}
