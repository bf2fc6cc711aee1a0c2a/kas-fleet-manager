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
					AllowedUsers: []string{},
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
					AllowedUsers: []string{"username"},
				},
				Organisation{
					Id:           id,
					AllowedUsers: []string{"user-1"},
				},
			},
			want: result{
				ok: true,
				organisation: Organisation{
					Id:           id,
					AllowedUsers: []string{"user-1"},
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
				AllowedUsers: []string{"user-1"},
			},
			want: false,
		},
		{
			name: "return 'true' and the organisation allow list contains the given username",
			arg:  username,
			org: Organisation{
				AllowedUsers: []string{"user-1", username},
			},
			want: true,
		},
		{
			name: "return 'true' if organisation is empty and all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedUsers: []string{},
				AllowAll:     true,
			},
			want: true,
		},
		{
			name: "return 'false' if organisation is empty and not all users are allowed",
			arg:  username,
			org: Organisation{
				AllowedUsers: []string{},
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
				AllowedUsers: []string{},
			},
			want: false,
		},
		{
			name: "return 'true' when the organisation contains a non empty allow list of users",
			org: Organisation{
				AllowedUsers: []string{"username-1", "username-2"},
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
