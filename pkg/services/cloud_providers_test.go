package services

import (
	"errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/patrickmn/go-cache"
	"reflect"
	"time"

	"testing"
)

func Test_CloudProvider_List(t *testing.T) {

	type fields struct {
		ocmClient ocm.Client
		cache     *cache.Cache
	}

	newCache := cache.New(5*time.Minute, 10*time.Minute)
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []CloudProviderWithRegions
	}{
		{
			name: "fail to get cloud provider regions",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*v1.CloudProviderList, error) {
						return nil, errors.New("GetCloudProviders fail to get the list of providers")
					},
					GetRegionsFunc: func(provider *v1.CloudProvider) (*v1.CloudRegionList, error) {
						return nil, errors.New("GetRegions fail to get the list of regions")
					},
				},
				cache: newCache,
			},
			wantErr: true,
		},
		{
			name: "successful get cloud provider regions",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*v1.CloudProviderList, error) {
						return &v1.CloudProviderList{}, nil
					},
					GetRegionsFunc: func(provider *v1.CloudProvider) (*v1.CloudRegionList, error) {
						return &v1.CloudRegionList{}, nil
					},
				},
				cache: newCache,
			},
			wantErr: false,
			want:    []CloudProviderWithRegions{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := cloudProvidersService{
				ocmClient: tt.fields.ocmClient,
				cache: tt.fields.cache,
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
