package clusters

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

func TestNewDefaultProviderFactory(t *testing.T) {
	type args struct {
		ocmClient              ocm.ClusterManagementClient
		connectionFactory      *db.ConnectionFactory
		ocmConfig              *ocm.OCMConfig
		awsConfig              *config.AWSConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name string
		args args
		want *DefaultProviderFactory
	}{
		{
			name: "Should return the default Provider Factory",
			args: args{},
			want: &DefaultProviderFactory{
				providerContainer: map[api.ClusterProviderType]Provider{
					api.ClusterProviderStandalone: &StandaloneProvider{},
					api.ClusterProviderOCM:        &OCMProvider{
						clusterBuilder: &clusterBuilder{
							idGenerator: ocm.NewIDGenerator("mk-"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDefaultProviderFactory(tt.args.ocmClient, tt.args.connectionFactory, tt.args.ocmConfig, tt.args.awsConfig, tt.args.dataplaneClusterConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDefaultProviderFactory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultProviderFactory_GetProvider(t *testing.T) {
	type fields struct {
		providerContainer map[api.ClusterProviderType]Provider
	}
	type args struct {
		providerType api.ClusterProviderType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Provider
		wantErr bool
	}{
		{
			name: "Should return ocm Provider when provider type is empty",
			fields: fields{
				providerContainer: map[api.ClusterProviderType]Provider{
					api.ClusterProviderStandalone: &StandaloneProvider{},
					api.ClusterProviderOCM:        &OCMProvider{},
				},
			},
			args: args{
				providerType: "",
			},
			want: &OCMProvider{},
		},
		{
			name: "Should return correct Provider type",
			fields: fields{
				providerContainer: map[api.ClusterProviderType]Provider{
					api.ClusterProviderStandalone: &StandaloneProvider{},
					api.ClusterProviderOCM:        &OCMProvider{},
				},
			},
			args: args{
				providerType: "ocm",
			},
			want: &OCMProvider{},
		},
		{
			name: "Should return error when invalid provider type is given",
			fields: fields{
				providerContainer: map[api.ClusterProviderType]Provider{
					api.ClusterProviderStandalone: &StandaloneProvider{},
					api.ClusterProviderOCM:        &OCMProvider{},
				},
			},
			args: args{
				providerType: "invalid-provider-type",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DefaultProviderFactory{
				providerContainer: tt.fields.providerContainer,
			}
			got, err := d.GetProvider(tt.args.providerType)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultProviderFactory.GetProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultProviderFactory.GetProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}