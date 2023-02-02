package quota_management

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/onsi/gomega"
	"testing"
	"time"
)

var testAccounts = AccountList{
	{
		Username: "account-1",
	},
	{
		Username:            "account-2",
		MaxAllowedInstances: 12,
	},
	{
		Username: "account-3",
		GrantedQuota: QuotaList{
			Quota{InstanceTypeID: "enterprise"},
		},
	},
	{
		Username: "account-4",
		GrantedQuota: QuotaList{
			Quota{
				InstanceTypeID: "standard",
				KafkaBillingModels: BillingModelList{
					BillingModel{
						Id:                  "eval",
						ExpirationDate:      func() *ExpirationDate { res := ExpirationDate(time.Now()); return &res }(),
						MaxAllowedInstances: 50,
					},
				},
			},
		},
	},
}

func Test_Account_IsUsersRegistered(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name     string
		accounts AccountList
		want     bool
		args     args
	}{
		{
			name:     "test non-existent user",
			accounts: testAccounts,
			args:     args{username: "account-154"},
			want:     false,
		},
		{
			name:     "test existing user",
			accounts: testAccounts,
			args:     args{username: "account-3"},
			want:     true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			_, userRegistered := tt.accounts.GetByUsername(tt.args.username)
			g.Expect(userRegistered).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Account_GetMaxAllowedInstances(t *testing.T) {
	type args struct {
		instanceType   string
		billingModelID string
	}
	tests := []struct {
		name    string
		account Account
		want    int
		args    args
	}{
		{
			name:    "test default quota (no quota configured)",
			account: Account{Username: "test-account"},
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 1,
		},
		{
			name: "test organisation quota (configured at org level)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
			},
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 5,
		},
		{
			name: "test organisation quota (configured at billing model level)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota: []Quota{
					{
						InstanceTypeID: "standard",
						KafkaBillingModels: []BillingModel{
							{
								Id:                  "standard",
								MaxAllowedInstances: 8,
							},
						},
					},
				},
			},
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 8,
		},
		{
			name: "test invalid billing model",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota: []Quota{
					{
						InstanceTypeID: "standard",
						KafkaBillingModels: []BillingModel{
							{
								Id:                  "standard",
								MaxAllowedInstances: 8,
							},
						},
					},
				},
			},
			args: args{
				instanceType:   "standard",
				billingModelID: "eval",
			},
			want: 0,
		},
		{
			name: "test invalid instance type",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota: []Quota{
					{
						InstanceTypeID: "standard",
						KafkaBillingModels: []BillingModel{
							{
								Id:                  "standard",
								MaxAllowedInstances: 8,
							},
						},
					},
				},
			},
			args: args{
				instanceType:   "eval",
				billingModelID: "standard",
			},
			want: 0,
		},
		{
			name: "test organisation quota (expired billing model)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota: []Quota{
					{
						InstanceTypeID: "standard",
						KafkaBillingModels: []BillingModel{
							{
								Id:                  "eval",
								MaxAllowedInstances: 8,
								ExpirationDate:      func() *ExpirationDate { res := ExpirationDate(time.Now()); return &res }(),
							},
						},
					},
				},
			},
			args: args{
				instanceType:   "standard",
				billingModelID: "eval",
			},
			want: 0,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.account.GetMaxAllowedInstances(tt.args.instanceType, tt.args.billingModelID)).To(gomega.Equal(tt.want))
			g.Expect(tt.account.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want-1)).To(gomega.BeTrue())
			g.Expect(tt.account.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want)).To(gomega.BeTrue())
			g.Expect(tt.account.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want+1)).To(gomega.BeFalse())
		})
	}
}

func Test_Account_GetGrantedQuota(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		want    QuotaList
	}{
		{
			name:    "test default quota (no quota configured)",
			account: Account{Username: "test-account"},
			want:    []Quota{{InstanceTypeID: "standard"}},
		},
		{
			name: "test configured quota",
			account: Account{
				Username:     "test-account",
				GrantedQuota: []Quota{{InstanceTypeID: "eval"}},
			},
			want: []Quota{
				{
					InstanceTypeID:     "eval",
					KafkaBillingModels: nil,
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			grantedQuota := tt.account.GetGrantedQuota()
			g.Expect(grantedQuota).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Account_HasQuotaFor(t *testing.T) {
	type args struct {
		instanceType string
		billingModel string
	}

	tests := []struct {
		name    string
		account Account
		want    bool
		args    args
	}{
		{
			name:    "test for standard/standard quota (no quota configured)",
			account: Account{Username: "test-account"},
			args: args{
				instanceType: "standard",
				billingModel: "standard",
			},
			want: true,
		},
		{
			name:    "test for eval/standard quota (no quota configured)",
			account: Account{Username: "test-account"},
			args: args{
				instanceType: "eval",
				billingModel: "standard",
			},
			want: false,
		},
		{
			name: "test standard/standard (configured quota)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota:        []Quota{{InstanceTypeID: "eval"}},
			},
			args: args{
				instanceType: "standard",
				billingModel: "standard",
			},
			want: false,
		},
		{
			name: "test eval/standard (configured quota - no billing models)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota:        []Quota{{InstanceTypeID: "eval"}},
			},
			args: args{
				instanceType: "eval",
				billingModel: "standard",
			},
			want: true,
		},
		{
			name: "test eval/eval (configured quota - no billing models)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota:        []Quota{{InstanceTypeID: "eval"}},
			},
			args: args{
				instanceType: "eval",
				billingModel: "eval",
			},
			want: false,
		},
		{
			name: "test eval/eval (configured quota - full configuration)",
			account: Account{
				Username:            "test-account",
				MaxAllowedInstances: 5,
				GrantedQuota: []Quota{{
					InstanceTypeID: "eval",
					KafkaBillingModels: []BillingModel{
						{
							Id:                  "eval",
							MaxAllowedInstances: 5,
						},
					}},
				},
			},
			args: args{
				instanceType: "eval",
				billingModel: "eval",
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.account.HasQuotaConfigurationFor(tt.args.instanceType, tt.args.billingModel)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Account_GetByUsername(t *testing.T) {
	type args struct {
		username string
	}

	tests := []struct {
		name     string
		accounts AccountList
		args     args
		want     Account
		ok       bool
	}{
		{
			name:     "test valid username",
			accounts: testAccounts,
			args: args{
				username: "account-1",
			},
			want: func() Account {
				_, a := arrays.FindFirst(testAccounts, func(a Account) bool { return a.Username == "account-1" })
				return a
			}(),
			ok: true,
		},
		{
			name:     "test invalid username",
			accounts: testAccounts,
			args: args{
				username: "username-invalid",
			},
			want: Account{},
			ok:   false,
		},
		{
			name:     "test empty list",
			accounts: nil,
			args: args{
				username: "username-invalid",
			},
			want: Account{},
			ok:   false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			org, ok := tt.accounts.GetByUsername(tt.args.username)
			g.Expect(ok).To(gomega.Equal(tt.ok))
			g.Expect(org).To(gomega.Equal(tt.want))
		})
	}
}
