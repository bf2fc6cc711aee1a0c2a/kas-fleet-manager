package acl

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func Test_EnterpriseClusterRegistrationAccessControlListConfig_IsOrganizationAccepted(t *testing.T) {
	t.Parallel()
	orgId := "test-org-id"
	tests := []struct {
		name                  string
		arg                   string
		acceptedOrganisations EnterpriseClusterRegistrationAcceptedOrganizations
		want                  bool
	}{
		{
			name:                  "return 'false' when accepted organizations list is empty",
			arg:                   orgId,
			acceptedOrganisations: EnterpriseClusterRegistrationAcceptedOrganizations{},
			want:                  false,
		},
		{
			name:                  "return 'false' when accepted organizations list does not contain the given organization",
			arg:                   orgId,
			acceptedOrganisations: EnterpriseClusterRegistrationAcceptedOrganizations{"org-id-not-on-list", "org-id-not-on-list-2"},
			want:                  false,
		},
		{
			name:                  "return 'true' when accepted organizations list contains the given organization",
			arg:                   orgId,
			acceptedOrganisations: EnterpriseClusterRegistrationAcceptedOrganizations{"org-id-not-on-list", "test-org-id", "org-id-not-on-list-2"},
			want:                  true,
		},
	}

	for _, tastcase := range tests {
		tt := tastcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			ok := tt.acceptedOrganisations.IsOrganizationAccepted(tt.arg)
			g.Expect(ok).To(gomega.Equal(tt.want))
		})
	}
}

func Test_EnterpriseClusterRegistrationAccessControlListConfig_ReadFiles(t *testing.T) {
	type fields struct {
		enterpriseClusterRegistrationAccessControlListConfigFile string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should not return an error when the config file path is valid",
			fields: fields{
				enterpriseClusterRegistrationAccessControlListConfigFile: "config/enterprise-cluster-registration-list-config.yaml",
			},
			wantErr: false,
		},
		{
			name: "should return an error when config file path is invalid",
			fields: fields{
				enterpriseClusterRegistrationAccessControlListConfigFile: "invalid-path",
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := &EnterpriseClusterRegistrationAccessControlListConfig{
				EnterpriseClusterRegistrationAccessControlListConfigFile: tt.fields.enterpriseClusterRegistrationAccessControlListConfigFile,
			}
			err := config.ReadFiles()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AddFlags(t *testing.T) {
	fs := &pflag.FlagSet{}
	g := gomega.NewWithT(t)
	config := NewEnterpriseClusterRegistrationAccessControlListConfig()
	config.AddFlags(fs)
	str, err := fs.GetString("enterprise-cluster-registration-access-control-list-config-file")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(str).To(gomega.Equal(config.EnterpriseClusterRegistrationAccessControlListConfigFile))
}
