package acl

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_OrganisationList_GetById(t *testing.T) {
	t.Parallel()
	id := "org-id"
	type result struct {
		ok           bool
		organisation Organisation
	}

	tests := []struct {
		name string
		arg  string
		orgs OrganisationList
		want result
	}{
		{
			name: "return 'false' when organisation list is empty",
			arg:  id,
			orgs: OrganisationList{},
			want: result{
				ok:           false,
				organisation: Organisation{},
			},
		},
		{
			name: "return 'false' when organisation list does not contain the organisation for the given id",
			arg:  id,
			orgs: OrganisationList{
				Organisation{
					Id:              "org-1",
					AllowedAccounts: AllowedAccounts{},
				},
			},
			want: result{
				ok:           false,
				organisation: Organisation{},
			},
		},
		{
			name: "return 'true' and the corresponding organisation when organisation list contains the organisation for the given id",
			arg:  id,
			orgs: OrganisationList{
				Organisation{
					Id:              "org-2",
					AllowedAccounts: AllowedAccounts{AllowedAccount{Username: "username"}},
				},
				Organisation{
					Id:              id,
					AllowedAccounts: AllowedAccounts{AllowedAccount{Username: "user-1"}},
				},
			},
			want: result{
				ok: true,
				organisation: Organisation{
					Id:              id,
					AllowedAccounts: AllowedAccounts{AllowedAccount{Username: "user-1"}},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			organisation, ok := tt.orgs.GetById(tt.arg)
			Expect(organisation).To(Equal(tt.want.organisation))
			Expect(ok).To(Equal(tt.want.ok))
		})
	}
}

func Test_Organisation_IsUserAllowed(t *testing.T) {
	t.Parallel()
	username := "username"
	tests := []struct {
		name string
		arg  string
		org  Organisation
		want bool
	}{
		{
			name: "return 'false' when allow list is empty",
			arg:  username,
			org:  Organisation{},
			want: false,
		},
		{
			name: "return 'false' when allow list does not contain the given username",
			arg:  username,
			org: Organisation{
				AllowedAccounts: AllowedAccounts{AllowedAccount{Username: "user-1"}},
			},
			want: false,
		},
		{
			name: "return 'true' and the organisation allow list contains the given username",
			arg:  username,
			org: Organisation{
				AllowedAccounts: AllowedAccounts{
					AllowedAccount{Username: "user-1"},
					AllowedAccount{Username: username},
				},
			},
			want: true,
		},
		{
			name: "return 'true' if organisation is empty and all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedAccounts: AllowedAccounts{},
				AllowAll:        true,
			},
			want: true,
		},
		{
			name: "return 'false' if organisation is empty and not all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedAccounts: AllowedAccounts{},
				AllowAll:        false,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.org.IsUserAllowed(tt.arg)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_Organisation_HasAllowedAccounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		org  Organisation
		want bool
	}{
		{
			name: "return 'false' when the organisation allow list of users is empty",
			org: Organisation{
				AllowedAccounts: AllowedAccounts{},
			},
			want: false,
		},
		{
			name: "return 'true' when the organisation contains a non empty allow list of users",
			org: Organisation{
				AllowedAccounts: AllowedAccounts{
					AllowedAccount{Username: "username-1"},
					AllowedAccount{Username: "username-2"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.org.HasAllowedAccounts()
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedAccounts_GetByUsername(t *testing.T) {
	t.Parallel()
	username := "username"
	type result struct {
		found          bool
		allowedAccount AllowedAccount
	}
	tests := []struct {
		name            string
		arg             string
		allowedAccounts AllowedAccounts
		want            result
	}{
		{
			name:            "return 'false' when allowed users is empty",
			arg:             username,
			allowedAccounts: AllowedAccounts{},
			want: result{
				found:          false,
				allowedAccount: AllowedAccount{},
			},
		},
		{
			name:            "return 'false' when allowed users does not contain the given username",
			arg:             username,
			allowedAccounts: AllowedAccounts{AllowedAccount{Username: "user-1"}},
			want: result{
				found:          false,
				allowedAccount: AllowedAccount{},
			},
		},
		{
			name: "return 'true' and the allowed user when when allowed users contains the given username",
			arg:  username,
			allowedAccounts: AllowedAccounts{
				AllowedAccount{Username: "user-1"},
				AllowedAccount{Username: username},
			},
			want: result{
				found:          true,
				allowedAccount: AllowedAccount{Username: username},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			user, found := tt.allowedAccounts.GetByUsername(tt.arg)
			Expect(user).To(Equal(tt.want.allowedAccount))
			Expect(found).To(Equal(tt.want.found))
		})
	}
}

func Test_AllowedAccount_IsInstanceCountWithinLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		count int
		item  AllowedListItem
		want  bool
	}{
		{
			name:  "return 'false' when count is above or equal max allowed instances for the given organisation",
			count: 3,
			item:  Organisation{MaxAllowedInstances: 2},
			want:  false,
		},
		{
			name:  "return 'true' when count is below max allowed instances for the given organisation",
			count: 1,
			item:  Organisation{MaxAllowedInstances: 2},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal default value (1) of max allowed instances for the given organisation without max allowed instances set",
			count: 1,
			item:  Organisation{},
			want:  false,
		},
		{
			name:  "return 'true' when count is below the default value (1) of max allowed instances for the given organisation without max allowed instances set",
			count: 0,
			item:  Organisation{},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal max allowed instances for the given account",
			count: 3,
			item:  AllowedAccount{MaxAllowedInstances: 2},
			want:  false,
		},
		{
			name:  "return 'true' when count is below max allowed instances for the given account",
			count: 1,
			item:  AllowedAccount{MaxAllowedInstances: 2},
			want:  true,
		},
		{
			name:  "return 'false' when count is above or equal default value (1) of max allowed instances for the given account without max allowed instances set",
			count: 1,
			item:  AllowedAccount{},
			want:  false,
		},
		{
			name:  "return 'true' when count is below the default value (1) of max allowed instances for the given account without max allowed instances set",
			count: 0,
			item:  AllowedAccount{},
			want:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.item.IsInstanceCountWithinLimit(tt.count)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedAccount_GetMaxAllowedInstances(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		allowedAccount AllowedAccount
		want           int
	}{
		{
			name:           "return max allowed instances of the user when its value is given",
			allowedAccount: AllowedAccount{MaxAllowedInstances: 2},
			want:           2,
		},
		{
			name:           "return default value (1) of max allowed instances when max allowed instance per user is not set",
			allowedAccount: AllowedAccount{},
			want:           1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.allowedAccount.GetMaxAllowedInstances()
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_DeniedUsers_IsUserDenied(t *testing.T) {
	t.Parallel()
	username := "user-name"
	tests := []struct {
		name           string
		arg            string
		deniedAccounts DeniedUsers
		want           bool
	}{
		{
			name:           "return 'false' when denied accounts list is empty",
			arg:            username,
			deniedAccounts: DeniedUsers{},
			want:           false,
		},
		{
			name:           "return 'false' when denied accounts list does not contain the given username",
			arg:            username,
			deniedAccounts: DeniedUsers{"user1", "user2"},
			want:           false,
		},
		{
			name:           "return 'true' when denied accounts list contains the given username",
			arg:            username,
			deniedAccounts: DeniedUsers{"user1", "user-name", "user2"},
			want:           true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.deniedAccounts.IsUserDenied(tt.arg)
			Expect(ok).To(Equal(tt.want))
		})
	}
}
