package account

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestNewMockAccountService(t *testing.T) {
	g := NewWithT(t)

	accountService := NewMockAccountService()
	_, isExpectedType := accountService.(*mock)
	g.Expect(isExpectedType).To(BeTrue())
}

func Test_mock_SearchOrganizations(t *testing.T) {
	type args struct {
		filter string
	}
	tests := []struct {
		name    string
		a       mock
		args    args
		want    *OrganizationList
		wantErr bool
	}{
		{
			name: "should return a list of mock Organizations",
			a:    mock{},
			args: args{
				filter: "",
			},
			want:    buildMockOrganizationList(10),
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := mock{}
			got, err := a.SearchOrganizations(tt.args.filter)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))

		})
	}
}

func Test_mock_GetOrganization(t *testing.T) {
	type args struct {
		filter string
	}
	tests := []struct {
		name    string
		a       mock
		args    args
		want    *Organization
		wantErr bool
	}{
		{
			name: "should return specified Organizations",
			a:    mock{},
			args: args{
				filter: "mock-id-0",
			},
			want: &Organization{
				ID:            "mock-id-0",
				Name:          "mock-org-0",
				AccountNumber: "mock-ebs-0",
				ExternalID:    "mock-extid-0",
				CreatedAt:     time.Now().Truncate(time.Minute),
				UpdatedAt:     time.Now().Truncate(time.Minute),
			},
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := mock{}
			got, err := a.GetOrganization(tt.args.filter)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}
