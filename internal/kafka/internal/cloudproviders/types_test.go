package cloudproviders

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_CloudProviderCollection_Contains(t *testing.T) {
	type fields struct {
		cloudProvidercollection CloudProviderCollection
	}

	type args struct {
		id CloudProviderID
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Contains returns false when the cloud provider id is not in the collection",
			fields: fields{
				cloudProvidercollection: CloudProviderCollection{
					map[CloudProviderID]struct{}{
						AWS: {},
						GCP: {},
					},
				},
			},
			args: args{
				id: Azure,
			},
			want: false,
		},
		{
			name: "Contains returns false when the cloud provider is Unknown and it is not in the collection",
			fields: fields{
				cloudProvidercollection: CloudProviderCollection{
					map[CloudProviderID]struct{}{
						AWS: {},
						GCP: {},
					},
				},
			},
			args: args{
				id: Unknown,
			},
			want: false,
		},
		{
			name: "Contains returns true when the cloud provider id is in the collection",
			fields: fields{
				cloudProvidercollection: CloudProviderCollection{
					map[CloudProviderID]struct{}{
						AWS: {},
						GCP: {},
					},
				},
			},
			args: args{
				id: AWS,
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.fields.cloudProvidercollection.Contains(tt.args.id)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ParseCloudProviderID(t *testing.T) {
	type args struct {
		id string
	}

	tests := []struct {
		name string
		args args
		want CloudProviderID
	}{
		{
			name: "returns the appropriate CloudProviderID when the cloud provider id is a known one",
			args: args{
				id: "aws",
			},
			want: AWS,
		},
		{
			name: "returns Unknown when the provided cloud provider id is not a known one",
			args: args{
				id: "nonexistingcloudprovider",
			},
			want: Unknown,
		},
		{
			name: "case normalization is performed when parsing the cloud provider id",
			args: args{
				id: "GcP",
			},
			want: GCP,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := ParseCloudProviderID(tt.args.id)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
