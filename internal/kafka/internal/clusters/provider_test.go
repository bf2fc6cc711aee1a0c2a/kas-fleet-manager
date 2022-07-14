package clusters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
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
					api.ClusterProviderOCM: &OCMProvider{
						clusterBuilder: &clusterBuilder{
							idGenerator: ocm.NewIDGenerator("mk-"),
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
			got := NewDefaultProviderFactory(tt.args.ocmClient, tt.args.connectionFactory, tt.args.ocmConfig, tt.args.awsConfig, tt.args.dataplaneClusterConfig)
			g.Expect(got).To(gomega.Equal(tt.want))
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
			wantErr: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			d := &DefaultProviderFactory{
				providerContainer: tt.fields.providerContainer,
			}
			got, err := d.GetProvider(tt.args.providerType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got == nil).To(gomega.Equal(tt.want == nil))
			if got != nil {
				g.Expect(got).To(gomega.Equal(tt.want))
			}
		})
	}
}
