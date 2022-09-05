package ocm

import (
	"testing"

	"github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func TestQuotaCostRelatedResourceFilter_IsMatch(t *testing.T) {
	type fields struct {
		ResourceName *string
		ResourceType *string
		Product      *string
	}
	type args struct {
		buildRelatedResource func() (*amsv1.RelatedResource, error)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should return false if resource name does not match",
			fields: fields{
				ResourceName: &[]string{"resource-name"}[0],
			},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().ResourceName("no-match").Build()
				},
			},
			want: false,
		},
		{
			name: "should return false if resource type does not match",
			fields: fields{
				ResourceType: &[]string{"resource-type"}[0],
			},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().ResourceType("no-match").Build()
				},
			},
			want: false,
		},
		{
			name: "should return false if product does not match",
			fields: fields{
				ResourceType: &[]string{"product"}[0],
			},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().Product("no-match").Build()
				},
			},
			want: false,
		},
		{
			name: "should return false if one of the given properties does not match",
			fields: fields{
				ResourceType: &[]string{"resource-type"}[0],
				Product:      &[]string{"product"}[0],
			},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().ResourceType("no-match").Product("product").Build()
				},
			},
			want: false,
		},
		{
			name: "should return true if all given properties matches",
			fields: fields{
				ResourceName: &[]string{"resource-name"}[0],
				ResourceType: &[]string{"resource-type"}[0],
				Product:      &[]string{"product"}[0],
			},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().ResourceName("resource-name").ResourceType("resource-type").Product("product").Build()
				},
			},
			want: true,
		},
		{
			name:   "should return true if properties for the filter was not defined",
			fields: fields{},
			args: args{
				buildRelatedResource: func() (*amsv1.RelatedResource, error) {
					return amsv1.NewRelatedResource().ResourceName("resource-name").ResourceType("resource-type").Product("product").Build()
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			qcf := &QuotaCostRelatedResourceFilter{
				ResourceName: tc.fields.ResourceName,
				ResourceType: tc.fields.ResourceType,
				Product:      tc.fields.Product,
			}

			mockRelatedResource, err := tc.args.buildRelatedResource()
			g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to build mock related resource object")

			got := qcf.IsMatch(mockRelatedResource)
			g.Expect(got).To(gomega.Equal(tc.want))
		})
	}
}
