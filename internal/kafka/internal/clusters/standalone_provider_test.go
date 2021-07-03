package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
)

func TestStandaloneProvider_GetCloudProviders(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"cloud_provider": "aws"}, {"cloud_provider": "azure"}})
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{
					{
						ID:          "aws",
						Name:        "aws",
						DisplayName: "aws",
					},
					{
						ID:          "azure",
						Name:        "azure",
						DisplayName: "azure",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviders()
			Expect(test.wantErr).To(Equal(err != nil))
			if !test.wantErr {
				Expect(resp.Items).To(Equal(test.want.Items))
			}
		})
	}
}

func TestStandaloneProvider_GetCloudProviderRegions(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderRegionInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters in a given cloud provider exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers regions",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"region": "af-east-1", "multi_az": false}, {"region": "eu-central-0", "multi_az": true}})
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{
					{
						ID:              "af-east-1",
						Name:            "af-east-1",
						DisplayName:     "af-east-1",
						CloudProviderID: "aws",
						SupportsMultiAZ: false,
					},
					{
						ID:              "eu-central-0",
						Name:            "eu-central-0",
						DisplayName:     "eu-central-0",
						CloudProviderID: "aws",
						SupportsMultiAZ: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviderRegions(types.CloudProviderInfo{ID: "aws"})
			Expect(test.wantErr).To(Equal(err != nil))
			if !test.wantErr {
				Expect(resp.Items).To(Equal(test.want.Items))
			}
		})
	}
}
