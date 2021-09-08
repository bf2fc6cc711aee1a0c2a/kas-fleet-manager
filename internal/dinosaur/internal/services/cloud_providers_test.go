package services

import (
	"errors"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/clusters/types"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	mocket "github.com/selvatico/go-mocket"
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
											{
												ID:          "aws",
												Name:        "aws",
												DisplayName: "aws",
											},
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
										{
											ID:          "azure",
											Name:        "azure",
											DisplayName: "azure",
										},
										{
											ID:          "aws",
											Name:        "aws",
											DisplayName: "aws",
										},
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
				{
					Id:          "aws",
					Name:        "aws",
					DisplayName: "Amazon Web Services",
				},
				{
					Id:          "azure",
					Name:        "azure",
					DisplayName: "Microsoft Azure",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.ListCloudProviders()
			Expect(tt.wantErr).To(Equal(err != nil))
			Expect(got).To(Equal(tt.want))
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
											{
												ID:          "aws",
												Name:        "aws",
												DisplayName: "aws",
											},
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
									return &types.CloudProviderRegionInfoList{
										Items: []types.CloudProviderRegionInfo{
											{
												ID:              "af-east-1",
												CloudProviderID: providerInf.ID,
												Name:            "af-east-1",
												DisplayName:     "af-east-1",
											},
										},
									}, nil
								},
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										{
											ID:          "azure",
											Name:        "azure",
											DisplayName: "azure",
										},
										{
											ID:          "aws",
											Name:        "aws",
											DisplayName: "aws",
										},
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										{
											ID:              "af-east-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-east-1",
											DisplayName:     "af-east-1",
										},
										{
											ID:              "af-south-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-south-1",
											DisplayName:     "af-south-1",
										},
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
							{
								CloudProviderID: "aws",
								Name:            "af-east-1",
								DisplayName:     "af-east-1",
								ID:              "af-east-1",
							},
							{
								CloudProviderID: "aws",
								Name:            "af-south-1",
								DisplayName:     "af-south-1",
								ID:              "af-south-1",
							},
						},
					},
				},
				{
					ID: "azure",
					RegionList: &types.CloudProviderRegionInfoList{
						Items: []types.CloudProviderRegionInfo{
							{
								CloudProviderID: "azure",
								Name:            "af-east-1",
								DisplayName:     "af-east-1",
								ID:              "af-east-1",
							},
							{
								CloudProviderID: "azure",
								Name:            "af-south-1",
								DisplayName:     "af-south-1",
								ID:              "af-south-1",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				cache:             tt.fields.cache,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.GetCachedCloudProvidersWithRegions()
			Expect(tt.wantErr).To(Equal(err != nil))
			Expect(got).To(Equal(tt.want))
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
											{
												ID:          "aws",
												Name:        "aws",
												DisplayName: "aws",
											},
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
									return &types.CloudProviderRegionInfoList{
										Items: []types.CloudProviderRegionInfo{
											{
												ID:              "af-east-1",
												CloudProviderID: providerInf.ID,
												Name:            "af-east-1",
												DisplayName:     "af-east-1",
											},
										},
									}, nil
								},
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										{
											ID:          "azure",
											Name:        "azure",
											DisplayName: "azure",
										},
										{
											ID:          "aws",
											Name:        "aws",
											DisplayName: "aws",
										},
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										{
											ID:              "af-east-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-east-1",
											DisplayName:     "af-east-1",
										},
										{
											ID:              "af-south-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-south-1",
											DisplayName:     "af-south-1",
										},
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
			want: []api.CloudRegion{
				{
					CloudProvider: "aws",
					DisplayName:   "af-east-1",
					Id:            "af-east-1",
				},
				{
					CloudProvider: "aws",
					DisplayName:   "af-south-1",
					Id:            "af-south-1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				cache:             tt.fields.cache,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.ListCloudProviderRegions("aws")
			Expect(tt.wantErr).To(Equal(err != nil))
			Expect(got).To(Equal(tt.want))
		})
	}
}
