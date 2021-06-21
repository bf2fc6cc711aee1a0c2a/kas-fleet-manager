package services

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
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
					WithQuery("SELECT DISTINCT").
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
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{})
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
	var cloudProviderWithRegions []CloudProviderWithRegions
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
				providerFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
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
			name: "successful get cloud provider regions",
			fields: fields{
				providerFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
							return &types.CloudProviderInfoList{}, nil
						},
					}, nil
				}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			wantErr: false,
			want:    cloudProviderWithRegions,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
				cache:           tt.fields.cache,
			}
			got, err := p.GetCachedCloudProvidersWithRegions()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCloudProvidersWithRegions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCloudProvidersWithRegions() got = %+v, want %+v", got, tt.want)
			}

		})
	}
}
