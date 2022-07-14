package quota_management

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewQuotaManagementListConfig(t *testing.T) {
	tests := []struct {
		name string
		want *QuotaManagementListConfig
	}{
		{
			name: "Should return the QuotaManagementListConfig",
			want: &QuotaManagementListConfig{
				QuotaList: RegisteredUsersListConfiguration{
					Organisations:   nil,
					ServiceAccounts: nil,
				},
				QuotaListConfigFile:        "config/quota-management-list-configuration.yaml",
				EnableInstanceLimitControl: false,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewQuotaManagementListConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_QuotaManagementListConfig_ReadFiles(t *testing.T) {
	type fields struct {
		QuotaList                  RegisteredUsersListConfiguration
		QuotaListConfigFile        string
		EnableInstanceLimitControl bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should sucesfully read the config files without error",
			fields: fields{
				QuotaList: RegisteredUsersListConfiguration{
					Organisations:   nil,
					ServiceAccounts: nil,
				},
				QuotaListConfigFile:        "config/quota-management-list-configuration.yaml",
				EnableInstanceLimitControl: false,
			},
			wantErr: false,
		},
		{
			name: "Should return nil if the file cannot be found",
			fields: fields{
				QuotaList: RegisteredUsersListConfiguration{
					Organisations:   nil,
					ServiceAccounts: nil,
				},
				QuotaListConfigFile:        "fake-file-path",
				EnableInstanceLimitControl: false,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &QuotaManagementListConfig{
				QuotaList:                  tt.fields.QuotaList,
				QuotaListConfigFile:        tt.fields.QuotaListConfigFile,
				EnableInstanceLimitControl: tt.fields.EnableInstanceLimitControl,
			}
			g.Expect(c.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_QuotaManagementListConfig_GetAllowedAccountByUsernameAndOrgId(t *testing.T) {
	type fields struct {
		QuotaList                  RegisteredUsersListConfiguration
		QuotaListConfigFile        string
		EnableInstanceLimitControl bool
	}
	type args struct {
		username string
		orgId    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Account
		found  bool
	}{
		{
			name:   "Should return false if Account is not found",
			fields: fields{},
			args:   args{},
			found:  false,
		},
		{
			name: "Should return true and the account if the account is found",
			fields: fields{
				QuotaList: RegisteredUsersListConfiguration{
					Organisations: OrganisationList{
						Organisation{
							Id: "1234",
							RegisteredUsers: AccountList{
								Account{
									Username: "account-username",
								},
							},
						},
					},
				},
				QuotaListConfigFile:        "config/quota-management-list-configuration.yaml",
				EnableInstanceLimitControl: false,
			},
			args: args{
				username: "account-username",
				orgId:    "1234",
			},
			want: Account{
				Username: "account-username",
			},
			found: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &QuotaManagementListConfig{
				QuotaList:                  tt.fields.QuotaList,
				QuotaListConfigFile:        tt.fields.QuotaListConfigFile,
				EnableInstanceLimitControl: tt.fields.EnableInstanceLimitControl,
			}
			got, found := c.GetAllowedAccountByUsernameAndOrgId(tt.args.username, tt.args.orgId)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(found).To(gomega.Equal(tt.found))
		})
	}
}
