package config

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
					Id:           "org-1",
					AllowedUsers: AllowedUsers{},
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
					Id:           "org-2",
					AllowedUsers: AllowedUsers{AllowedUser{Username: "username"}},
				},
				Organisation{
					Id:           id,
					AllowedUsers: AllowedUsers{AllowedUser{Username: "user-1"}},
				},
			},
			want: result{
				ok: true,
				organisation: Organisation{
					Id:           id,
					AllowedUsers: AllowedUsers{AllowedUser{Username: "user-1"}},
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
				AllowedUsers: AllowedUsers{AllowedUser{Username: "user-1"}},
			},
			want: false,
		},
		{
			name: "return 'true' and the organisation allow list contains the given username",
			arg:  username,
			org: Organisation{
				AllowedUsers: AllowedUsers{
					AllowedUser{Username: "user-1"},
					AllowedUser{Username: username},
				},
			},
			want: true,
		},
		{
			name: "return 'true' if organisation is empty and all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedUsers: AllowedUsers{},
				AllowAll:     true,
			},
			want: true,
		},
		{
			name: "return 'false' if organisation is empty and not all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedUsers: AllowedUsers{},
				AllowAll:     false,
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

func Test_Organisation_HasAllowedUsers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		org  Organisation
		want bool
	}{
		{
			name: "return 'false' when the organisation allow list of users is empty",
			org: Organisation{
				AllowedUsers: AllowedUsers{},
			},
			want: false,
		},
		{
			name: "return 'true' when the organisation contains a non empty allow list of users",
			org: Organisation{
				AllowedUsers: AllowedUsers{
					AllowedUser{Username: "username-1"},
					AllowedUser{Username: "username-2"},
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
			ok := tt.org.HasAllowedUsers()
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedUsers_GetByUsername(t *testing.T) {
	t.Parallel()
	username := "username"
	type result struct {
		found       bool
		allowedUser AllowedUser
	}
	tests := []struct {
		name         string
		arg          string
		allowedUsers AllowedUsers
		want         result
	}{
		{
			name:         "return 'false' when allowed users is empty",
			arg:          username,
			allowedUsers: AllowedUsers{},
			want: result{
				found:       false,
				allowedUser: AllowedUser{},
			},
		},
		{
			name:         "return 'false' when allowed users does not contain the given username",
			arg:          username,
			allowedUsers: AllowedUsers{AllowedUser{Username: "user-1"}},
			want: result{
				found:       false,
				allowedUser: AllowedUser{},
			},
		},
		{
			name: "return 'true' and the allowed user when when allowed users contains the given username",
			arg:  username,
			allowedUsers: AllowedUsers{
				AllowedUser{Username: "user-1"},
				AllowedUser{Username: username},
			},
			want: result{
				found:       true,
				allowedUser: AllowedUser{Username: username},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			user, found := tt.allowedUsers.GetByUsername(tt.arg)
			Expect(user).To(Equal(tt.want.allowedUser))
			Expect(found).To(Equal(tt.want.found))
		})
	}
}

func Test_AllowedUser_IsInstanceCountWithinLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		count       int
		allowedUser AllowedUser
		want        bool
	}{
		{
			name:        "return 'false' when count is above or equal max allowed instances for the given user",
			count:       3,
			allowedUser: AllowedUser{MaxAllowedInstances: 2},
			want:        false,
		},
		{
			name:        "return 'true' when count is below max allowed instances for the given user",
			count:       1,
			allowedUser: AllowedUser{MaxAllowedInstances: 2},
			want:        true,
		},
		{
			name:        "return 'false' when count is above or equal default value (1) of max allowed instances for the given user without max allowed instances set",
			count:       1,
			allowedUser: AllowedUser{},
			want:        false,
		},
		{
			name:        "return 'true' when count is below the default value (1) of max allowed instances for the given user without max allowed instances set",
			count:       0,
			allowedUser: AllowedUser{},
			want:        true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.allowedUser.IsInstanceCountWithinLimit(tt.count)
			Expect(ok).To(Equal(tt.want))
		})
	}
}

func Test_AllowedUser_GetMaxAllowedInstances(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		allowedUser AllowedUser
		want        int
	}{
		{
			name:        "return max allowed instances of the user when its value is given",
			allowedUser: AllowedUser{MaxAllowedInstances: 2},
			want:        2,
		},
		{
			name:        "return default value (1) of max allowed instances when max allowed instance per user is not set",
			allowedUser: AllowedUser{},
			want:        1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			RegisterTestingT(t)
			ok := tt.allowedUser.GetMaxAllowedInstances()
			Expect(ok).To(Equal(tt.want))
		})
	}
}
