package services

import (
	"errors"
	"testing"

	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/cloud_providers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/patrickmn/go-cache"
)

const (
	MockCloudProviderID_Azure          = "azure"
	MockCloudProviderName_Azure        = "azure"
	MockCloudProviderDisplayName_Azure = "Microsoft Azure"
	MockCloudProviderDisplayName_AWS   = "Amazon Web Services"
	MockRegionName_south               = "af-south-1"
	MockRegionName_east                = "us-east-1"
)

func defaultListClusterProviderTypeMockFn() []api.ClusterProviderType {
	return []api.ClusterProviderType{api.ClusterProviderOCM, api.ClusterProviderStandalone}
}

func Test_CloudProvider_ListCloudProviders(t *testing.T) {
	type fields struct {
		providerFactory clusters.ProviderFactory
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []api.CloudProvider
	}{
		{
			name: "fail to get cloud provider",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return nil, errors.New("GetCloudProviders fail to get the list of providers")
							},
							GetCloudProviderRegionsFunc: nil, // should not be called
						}, nil
					}},
			},
			wantErr: true,
		},
		{
			name: "successful merge the various list of cloud providers from different providers and return it",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
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
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
			}
			got, err := p.ListCloudProviders()
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_CachedCloudProviderWithRegions(t *testing.T) {
	type fields struct {
		providerFactory clusters.ProviderFactory
		cache           *cache.Cache
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []CloudProviderWithRegions
	}{
		{
			name: "fail to get cloud provider regions",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return nil, errors.New("GetCloudProviders fail to get the list of providers")
							},
						}, nil
					}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			wantErr: true,
		},
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
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
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
				cache:           tt.fields.cache,
			}
			got, err := p.GetCachedCloudProvidersWithRegions()
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ListCloudProviderRegions(t *testing.T) {
	type fields struct {
		providerFactory clusters.ProviderFactory
		cache           *cache.Cache
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []api.CloudRegion
	}{
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
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
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider implementation")
					},
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
				},
			},
			wantErr: true,
		},
		{
			name: "An error is returned when the cloud regions cannot be obtained",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
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
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
				cache:           tt.fields.cache,
			}
			got, err := p.ListCloudProviderRegions("aws")
			g.Expect(tt.wantErr).To(gomega.Equal(err != nil))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_cloudProvidersService_ListCachedCloudProviderRegions(t *testing.T) {
	type fields struct {
		providerFactory clusters.ProviderFactory
		cache           *cache.Cache
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []api.CloudRegion
		wantErr *svcErrors.ServiceError
	}{
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{
					ListClusterProviderTypesFunc: defaultListClusterProviderTypeMockFn,
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
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
				cache:           tt.fields.cache,
			}
			got, err := p.ListCachedCloudProviderRegions(tt.args.id)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err).To(gomega.Equal(tt.wantErr))
		})
	}
}
