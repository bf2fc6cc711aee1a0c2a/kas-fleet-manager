package services

import (
	"errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/patrickmn/go-cache"
	"reflect"
	"testing"
	"time"
)

func Test_CloudProvider_List(t *testing.T) {
	var cloudProviderWithRegions []CloudProviderWithRegions

	type fields struct {
		providerFactory clusters.ProviderFactory
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
						GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
							return nil, errors.New("GetRegions fail to get the list of regions")
						},
					}, nil
				}},
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
						GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
							return &types.CloudProviderRegionInfoList{}, nil
						},
					}, nil
				}},
			},
			wantErr: false,
			want:    cloudProviderWithRegions,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := cloudProvidersService{
				providerFactory: tt.fields.providerFactory,
			}
			got, err := p.GetCloudProvidersWithRegions()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCloudProvidersWithRegions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCloudProvidersWithRegions() got = %+v, want %+v", got, tt.want)
			}

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
