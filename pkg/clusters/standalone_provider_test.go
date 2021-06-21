package clusters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
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
			provider := newStandaloneProvider(test.fields.connectionFactory)
			resp, err := provider.GetCloudProviders()
			Expect(test.wantErr).To(Equal(err != nil))
			if !test.wantErr {
				Expect(resp.Items).To(Equal(test.want.Items))
			}
		})
	}
}
