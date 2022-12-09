package quota_management

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"testing"
	"time"

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
			name: "Should successfully read the config files without error",
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

const testCaseEval string = `
registered_service_accounts:
  - username: testuser1@example.com
    max_allowed_instances: 3
    granted_quota:
      - instance_type_id: standard
        kafka_billing_models:
          - id: eval
            expiration_date: %s
            grace_period_days: %s
registered_users_per_organisation:
  - id: 13640203
    any_user: true
    max_allowed_instances: 5
    registered_users: []
    granted_quota:
      - instance_type_id: standard
        kafka_billing_models:
          - id: eval
            max_allowed_instances: 5
            expiration_date: %s
            grace_period_days: %s
`

func Test_QuotaManagementListConfig_QuotaExpiration(t *testing.T) {

	utcLocation, _ := time.LoadLocation("UTC")

	type fields struct {
		ConfigFileContent string
	}
	tests := []struct {
		name      string
		fields    fields
		wantDate1 *time.Time
		wantDate2 *time.Time
	}{
		{
			name:      "Test config with expiration in UTC",
			wantDate1: func() *time.Time { res := time.Date(2023, 1, 30, 0, 0, 0, 0, utcLocation); return &res }(),
			wantDate2: func() *time.Time { res := time.Date(2023, 1, 30, 0, 0, 0, 0, utcLocation); return &res }(),
			fields: fields{
				ConfigFileContent: fmt.Sprintf(testCaseEval, "2023-01-30 +00:00", "3", "2023-01-30 +00:00", "5"),
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			var cfg RegisteredUsersListConfiguration
			err := yaml.UnmarshalStrict([]byte(tt.fields.ConfigFileContent), &cfg)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			org, ok := cfg.Organisations.GetById("13640203")
			g.Expect(ok).To(gomega.BeTrue())
			gq := org.GrantedQuota
			g.Expect(gq).To(gomega.HaveLen(1))
			q := gq[0]
			bm, ok := q.GetKafkaBillingModelByID("eval")
			g.Expect(ok).To(gomega.BeTrue())
			g.Expect(time.Time(*bm.ExpirationDate)).To(gomega.BeTemporally("==", *tt.wantDate1))
		})
	}
}
