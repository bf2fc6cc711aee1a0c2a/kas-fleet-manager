package account

import (
	"testing"

	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func TestNewAccountService(t *testing.T) {
	type args struct {
		connection *sdk.Connection
	}
	tests := []struct {
		name string
		args args
		want AccountService
	}{
		{
			name: "should return a new account service",
			args: args{
				connection: &sdk.Connection{},
			},
			want: &accountService{
				connection: &sdk.Connection{},
			},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g.Expect(NewAccountService(tt.args.connection)).To(Equal(tt.want))
		})
	}
}

func Test_convertOrganization(t *testing.T) {
	type args struct {
		o *v1.Organization
	}
	tests := []struct {
		name string
		args args
		want *Organization
	}{
		{
			name: "should successfully convert a v1 Organization to regular",
			args: args{
				o: &v1.Organization{},
			},
			want: &Organization{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g.Expect(convertOrganization(tt.args.o)).To(Equal(tt.want))
		})
	}
}

func Test_convertOrganizationList(t *testing.T) {
	type args struct {
		organizationList *v1.OrganizationList
	}
	tests := []struct {
		name string
		args args
		want *OrganizationList
	}{
		{
			name: "should successfully convert a v1 Organization list to regular list",
			args: args{
				organizationList: &v1.OrganizationList{},
			},
			want: &OrganizationList{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g.Expect(convertOrganizationList(tt.args.organizationList)).To(Equal(tt.want))
		})
	}
}
