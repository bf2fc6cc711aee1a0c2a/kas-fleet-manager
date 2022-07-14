package acl

import (
	"testing"

	"github.com/onsi/gomega"
)

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

	for _, rangevar := range tests {
		tt := rangevar
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			ok := tt.deniedAccounts.IsUserDenied(tt.arg)
			g.Expect(ok).To(gomega.Equal(tt.want))
		})
	}
}

func Test_AcceptedOrg_IsOrganisationAccepted(t *testing.T) {
	t.Parallel()
	orgId := "test-org-id"
	tests := []struct {
		name                  string
		arg                   string
		acceptedOrganisations AcceptedOrganisations
		want                  bool
	}{
		{
			name:                  "return 'false' when accepted accounts list is empty",
			arg:                   orgId,
			acceptedOrganisations: AcceptedOrganisations{},
			want:                  false,
		},
		{
			name:                  "return 'false' when accepted accounts list does not contain the given organisation",
			arg:                   orgId,
			acceptedOrganisations: AcceptedOrganisations{"org-id-not-on-list", "org-id-not-on-list-2"},
			want:                  false,
		},
		{
			name:                  "return 'true' when accepted accounts list contains the given organisation",
			arg:                   orgId,
			acceptedOrganisations: AcceptedOrganisations{"org-id-not-on-list", "test-org-id", "org-id-not-on-list-2"},
			want:                  true,
		},
	}

	for _, rangevar := range tests {
		tt := rangevar
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			ok := tt.acceptedOrganisations.IsOrganisationAccepted(tt.arg)
			g.Expect(ok).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ReadDenyListConfigFile(t *testing.T) {
	type args struct {
		file        string
		deniedUsers *DeniedUsers
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when attempting to read deny list with non-existent file",
			args: args{
				file:        "non-existent path",
				deniedUsers: &DeniedUsers{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when attempting to read deny list with existing file",
			args: args{
				file:        "config/deny-list-configuration.yaml",
				deniedUsers: &DeniedUsers{},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := readDenyListConfigFile(tt.args.file, tt.args.deniedUsers)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AccessControlListConfig_ReadFiles(t *testing.T) {
	type fields struct {
		denyList             DeniedUsers
		accessList           AcceptedOrganisations
		denyListConfigFile   string
		accessListConfigFile string
		enableDenyList       bool
		enableAccessList     bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should not return an error when deny list is enabled and deny list config file path is valid",
			fields: fields{
				denyListConfigFile: "config/deny-list-configuration.yaml",
				denyList:           DeniedUsers{},
				enableDenyList:     true,
			},
			wantErr: false,
		},
		{
			name: "should return an error when deny list is enabled and config file path is invalid",
			fields: fields{
				denyListConfigFile: "invalid-path",
				denyList:           DeniedUsers{},
				enableDenyList:     true,
			},
			wantErr: true,
		},
		{
			name: "should not return an error when deny list is disabled",
			fields: fields{
				denyListConfigFile: "config/deny-list-configuration.yaml",
				denyList:           DeniedUsers{},
				enableDenyList:     false,
			},
			wantErr: false,
		},
		{
			name: "should not return an error when deny list is disabled and deny list config file path is invalid",
			fields: fields{
				denyListConfigFile: "config/deny-list-configuration.yaml",
				denyList:           DeniedUsers{},
				enableDenyList:     false,
			},
			wantErr: false,
		},
		{
			name: "should not return an error when access list is enabled and access list config file path is valid",
			fields: fields{
				accessListConfigFile: "config/access-list-configuration.yaml",
				accessList:           AcceptedOrganisations{},
				enableAccessList:     true,
			},
			wantErr: false,
		},
		{
			name: "should return an error when access list is enabled and config file path is invalid",
			fields: fields{
				accessListConfigFile: "invalid-path",
				accessList:           AcceptedOrganisations{},
				enableAccessList:     true,
			},
			wantErr: true,
		},
		{
			name: "should not return an error when access list is disabled",
			fields: fields{
				accessListConfigFile: "config/access-list-configuration.yaml",
				accessList:           AcceptedOrganisations{},
				enableAccessList:     false,
			},
			wantErr: false,
		},
		{
			name: "should not return an error when access list is disabled and access list config file path is invalid",
			fields: fields{
				accessListConfigFile: "config/access-list-configuration.yaml",
				accessList:           AcceptedOrganisations{},
				enableAccessList:     false,
			},
			wantErr: false,
		},
		{
			name: "should not return an error when access list is disabled and access list config file path is invalid",
			fields: fields{
				accessListConfigFile: "config/access-list-configuration.yaml",
				accessList:           AcceptedOrganisations{},
				enableAccessList:     false,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			aclConfig := &AccessControlListConfig{
				DenyList:             tt.fields.denyList,
				AccessList:           tt.fields.accessList,
				DenyListConfigFile:   tt.fields.denyListConfigFile,
				AccessListConfigFile: tt.fields.accessListConfigFile,
				EnableDenyList:       tt.fields.enableDenyList,
				EnableAccessList:     tt.fields.enableAccessList,
			}
			err := aclConfig.ReadFiles()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_ReadAccessListConfigFile(t *testing.T) {
	type args struct {
		file                  string
		acceptedOrganisations *AcceptedOrganisations
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should return an error when attempting to read access list with invalid filepath",
			args: args{
				file:                  "non-existent path",
				acceptedOrganisations: &AcceptedOrganisations{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error when attempting to read access list with valid filepath",
			args: args{
				file:                  "config/access-list-configuration.yaml",
				acceptedOrganisations: &AcceptedOrganisations{},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := readAccessListConfigFile(tt.args.file, tt.args.acceptedOrganisations)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
