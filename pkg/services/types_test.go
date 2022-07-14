package services

import (
	"net/url"
	"testing"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func makeParams(orderByAry []string) map[string][]string {
	params := make(map[string][]string)
	params["orderBy"] = orderByAry
	return params
}

func getValidTestParams() []string {
	return []string{"bootstrap_server_host", "cloud_provider", "cluster_id", "created_at", "href", "id", "instance_type", "multi_az", "name", "organisation_id", "owner", "reauthentication_enabled", "region", "status", "updated_at", "version"}
}

func Test_ValidateOrderBy(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string][]string
		wantErr     bool
		validParams []string
	}{
		{
			name:        "One Column Asc",
			params:      makeParams([]string{"name asc"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "One Column Desc",
			params:      makeParams([]string{"region desc"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "Multiple Columns Mixed Sorting",
			params:      makeParams([]string{"region desc, name asc, cloud_provider desc"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "Multiple Columns Mixed Sorting with invalid column",
			params:      makeParams([]string{"region desc, name asc, invalid desc"}),
			wantErr:     true,
			validParams: getValidTestParams(),
		},
		{
			name:        "Multiple Columns Mixed Sorting with invalid sort",
			params:      makeParams([]string{"region desc, name asc, cloud_provider random"}),
			wantErr:     true,
			validParams: getValidTestParams(),
		},
		{
			name:        "Multiple Columns Multiple spaces",
			params:      makeParams([]string{"region    desc  ,    name   asc ,    cloud_provider asc"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "Invalid Column Desc",
			params:      makeParams([]string{"invalid desc"}),
			wantErr:     true,
			validParams: getValidTestParams(),
		},
		{
			name:        "No ordering - first",
			params:      makeParams([]string{"region"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "No ordering - mixed",
			params:      makeParams([]string{"region, cloud_provider desc, name"}),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
		{
			name:        "No ordering - all valid params",
			params:      makeParams(getValidTestParams()),
			wantErr:     false,
			validParams: getValidTestParams(),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			la := NewListArguments(tt.params)
			err := la.Validate(tt.validParams)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
func Test_NewListArguments(t *testing.T) {
	page := "5"
	size := "-1"
	search := "search"
	overriddenListArgs := &ListArguments{
		Page:   5,
		Size:   65500,
		Search: "search",
	}
	type args struct {
		params url.Values
	}

	tests := []struct {
		name string
		args args
		want *ListArguments
	}{
		{
			name: "should override default param",
			args: args{
				params: url.Values{
					"page":   []string{page},
					"size":   []string{size},
					"search": []string{search},
				},
			},
			want: overriddenListArgs,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(NewListArguments(tt.args.params)).To(gomega.Equal(tt.want))
		})
	}
}

func TestListArguments_Validate(t *testing.T) {
	type fields struct {
		Page    int
		Size    int
		Search  string
		OrderBy []string
	}
	type args struct {
		acceptedOrderByParams []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   error
	}{
		{
			name: "should return an error if page is < 0",
			fields: fields{
				Page: -1,
			},
			args: args{
				acceptedOrderByParams: getValidTestParams(),
			},
			want: errors.Errorf("page must be equal or greater than 0"),
		},
		{
			name: "should return an error if page is < 1",
			fields: fields{
				Page: 0,
			},
			args: args{
				acceptedOrderByParams: getValidTestParams(),
			},
			want: errors.Errorf("size must be equal or greater than 1"),
		},
		{
			name: "should return an error if there are too many keywords in orderBy",
			fields: fields{
				Page:    1,
				Size:    100,
				OrderBy: []string{"region desc name owner"},
			},
			args: args{
				acceptedOrderByParams: getValidTestParams(),
			},
			want: errors.Errorf("invalid order by clause 'region desc name owner'"),
		},
		{
			name: "should return nil if the validation is completed",
			fields: fields{
				Page:    1,
				Size:    100,
				Search:  "",
				OrderBy: []string{"name asc"},
			},
			args: args{
				acceptedOrderByParams: getValidTestParams(),
			},
			want: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			la := &ListArguments{
				Page:    tt.fields.Page,
				Size:    tt.fields.Size,
				Search:  tt.fields.Search,
				OrderBy: tt.fields.OrderBy,
			}
			err := la.Validate(tt.args.acceptedOrderByParams)
			if err != nil {
				g.Expect(err.Error()).To(gomega.Equal(tt.want.Error()))
			}
		})
	}
}
