package quota_management

import (
	"github.com/onsi/gomega"
	"testing"
)

var testOrganisation = Organisation{
	Id:                  "org123",
	AnyUser:             false,
	MaxAllowedInstances: 0,
	RegisteredUsers:     nil,
	GrantedQuota:        nil,
}

func Test_IsUsersRegistered(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name string
		org  Organisation
		want bool
		args args
	}{
		{
			name: "test non-existent user is registered - anyuser: true",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = true
				return org
			}(),
			args: args{username: "testuser123"},
			want: true,
		},
		{
			name: "test non-existent user is registered - anyuser: false",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = false
				return org
			}(),
			args: args{username: "testuser123"},
			want: false,
		},
		{
			name: "test existing user",
			org: func() Organisation {
				org := testOrganisation
				org.RegisteredUsers = AccountList{Account{Username: "testuser123"}}
				org.AnyUser = false
				return org
			}(),
			args: args{username: "testuser123"},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.org.IsUserRegistered(tt.args.username)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_GetMaxAllowedInstances(t *testing.T) {
	type args struct {
		instanceType   string
		billingModelID string
	}
	tests := []struct {
		name string
		org  Organisation
		want int
		args args
	}{
		{
			name: "test default quota (no quota configured)",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 1,
		},
		{
			name: "test organisation quota (configured at org level)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 5,
		},
		{
			name: "test organisation quota (configured at billing model level)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{
					{
						InstanceTypeID: "standard",
						BillingModels: []BillingModel{
							{
								ID:      "standard",
								Allowed: 8,
							},
						},
					},
				}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType:   "standard",
				billingModelID: "standard",
			},
			want: 8,
		},
		{
			name: "test invalid billing model",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{
					{
						InstanceTypeID: "standard",
						BillingModels: []BillingModel{
							{
								ID:      "standard",
								Allowed: 8,
							},
						},
					},
				}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType:   "standard",
				billingModelID: "eval",
			},
			want: 0,
		},
		{
			name: "test invalid instance type",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{
					{
						InstanceTypeID: "standard",
						BillingModels: []BillingModel{
							{
								ID:      "standard",
								Allowed: 8,
							},
						},
					},
				}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType:   "eval",
				billingModelID: "standard",
			},
			want: 0,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.org.GetMaxAllowedInstances(tt.args.instanceType, tt.args.billingModelID)).To(gomega.Equal(tt.want))
			g.Expect(tt.org.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want-1)).To(gomega.BeTrue())
			g.Expect(tt.org.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want)).To(gomega.BeTrue())
			g.Expect(tt.org.IsInstanceCountWithinLimit(tt.args.instanceType, tt.args.billingModelID, tt.want+1)).To(gomega.BeFalse())
		})
	}
}

func Test_GetGrantedQuota(t *testing.T) {
	tests := []struct {
		name string
		org  Organisation
		want QuotaList
	}{
		{
			name: "test default quota (no quota configured)",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = true
				return org
			}(),
			want: []Quota{{InstanceTypeID: "standard"}},
		},
		{
			name: "test configured quota",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{{InstanceTypeID: "eval"}}
				org.AnyUser = true
				return org
			}(),
			want: []Quota{
				{
					InstanceTypeID: "eval",
					BillingModels:  nil,
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			grantedQuota := tt.org.GetGrantedQuota()
			g.Expect(grantedQuota).To(gomega.Equal(tt.want))
		})
	}
}

func Test_HasQuotaFor(t *testing.T) {
	type args struct {
		instanceType string
		billingModel string
	}

	tests := []struct {
		name string
		org  Organisation
		want bool
		args args
	}{
		{
			name: "test for standard/standard quota (no quota configured)",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType: "standard",
				billingModel: "standard",
			},
			want: true,
		},
		{
			name: "test for eval/standard quota (no quota configured)",
			org: func() Organisation {
				org := testOrganisation
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType: "eval",
				billingModel: "standard",
			},
			want: false,
		},
		{
			name: "test standard/standard (configured quota)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{{InstanceTypeID: "eval"}}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType: "standard",
				billingModel: "standard",
			},
			want: false,
		},
		{
			name: "test eval/standard (configured quota - no billing models)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{{InstanceTypeID: "eval"}}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType: "eval",
				billingModel: "standard",
			},
			want: true,
		},
		{
			name: "test eval/eval (configured quota - no billing models)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{{InstanceTypeID: "eval"}}
				org.AnyUser = true
				return org
			}(),
			args: args{
				instanceType: "eval",
				billingModel: "eval",
			},
			want: false,
		},
		{
			name: "test eval/eval (configured quota - full configuration)",
			org: func() Organisation {
				org := testOrganisation
				org.MaxAllowedInstances = 5
				org.GrantedQuota = []Quota{{
					InstanceTypeID: "eval",
					BillingModels: []BillingModel{
						{
							ID:      "eval",
							Allowed: 5,
						},
					},
				}}
				org.AnyUser = true
				return org
			}(),
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
			g.Expect(tt.org.HasQuotaFor(tt.args.instanceType, tt.args.billingModel)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_GetByID(t *testing.T) {
	type args struct {
		orgID string
	}

	tests := []struct {
		name string
		orgs OrganisationList
		args args
		want Organisation
		ok   bool
	}{
		{
			name: "test valid id",
			orgs: func() OrganisationList {
				org := testOrganisation
				org.Id = "test-org-123"
				return OrganisationList{org}
			}(),
			args: args{
				orgID: "test-org-123",
			},
			want: func() Organisation {
				org := testOrganisation
				org.Id = "test-org-123"
				return org
			}(),
			ok: true,
		},
		{
			name: "test invalid id",
			orgs: func() OrganisationList {
				org := testOrganisation
				org.Id = "test-org-123"
				return OrganisationList{org}
			}(),
			args: args{
				orgID: "org-invalid",
			},
			want: Organisation{},
			ok:   false,
		},
		{
			name: "test empty list",
			orgs: nil,
			args: args{
				orgID: "org-invalid",
			},
			want: Organisation{},
			ok:   false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			org, ok := tt.orgs.GetById(tt.args.orgID)
			g.Expect(ok).To(gomega.Equal(tt.ok))
			g.Expect(org).To(gomega.Equal(tt.want))
		})
	}
}
