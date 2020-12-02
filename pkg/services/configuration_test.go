package services

import (
	"reflect"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"

	. "github.com/onsi/gomega"
)

func Test_configService_GetDefaultProvider(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	tests := []struct {
		name    string
		fields  fields
		want    config.Provider
		wantErr bool
	}{
		{
			name: "error when no default provider found",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{},
				},
			},
			wantErr: true,
			want:    config.Provider{},
		},
		{
			name: "success when default provider found",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test",
							Default: true,
						},
					},
				},
			},
			want: config.Provider{
				Name:    "test",
				Default: true,
			},
		},
		{
			name: "first default returned when multiple defaults specified",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test1",
							Default: true,
						},
						config.Provider{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			want: config.Provider{
				Name:    "test1",
				Default: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			got, err := c.GetDefaultProvider()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_GetDefaultRegionForProvider(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	type args struct {
		provider config.Provider
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    config.Region
		wantErr bool
	}{
		{
			name: "error when no default region found",
			args: args{
				provider: config.Provider{
					Regions: config.RegionList{},
				},
			},
			want:    config.Region{},
			wantErr: true,
		},
		{
			name: "success when default region found",
			args: args{
				provider: config.Provider{
					Regions: config.RegionList{
						config.Region{
							Name:    "test",
							Default: true,
						},
					},
				},
			},
			want: config.Region{
				Name:    "test",
				Default: true,
			},
		},
		{
			name: "first default returned when multiple defaults specified",
			args: args{
				provider: config.Provider{
					Regions: config.RegionList{
						config.Region{
							Name:    "test1",
							Default: true,
						},
						config.Region{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			want: config.Region{
				Name:    "test1",
				Default: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			got, err := c.GetDefaultRegionForProvider(tt.args.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultRegionForProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultRegionForProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_GetSupportedProviders(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	tests := []struct {
		name   string
		fields fields
		want   config.ProviderList
	}{
		{
			name: "successful get",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "test",
						},
					},
				},
			},
			want: config.ProviderList{
				config.Provider{
					Name: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			if got := c.GetSupportedProviders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSupportedProviders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_IsProviderSupported(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	type args struct {
		providerName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "false when provider not in supported list",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{},
				},
			},
			args: args{
				providerName: "test",
			},
			want: false,
		},
		{
			name: "true when provider in supported list",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "test",
						},
					},
				},
			},
			args: args{
				providerName: "test",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			if got := c.IsProviderSupported(tt.args.providerName); got != tt.want {
				t.Errorf("IsProviderSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configService_IsRegionSupportedForProvider(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	type args struct {
		providerName string
		regionName   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "false when provider is not supported",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{},
				},
			},
			args: args{
				providerName: "testProvider",
				regionName:   "testRegion",
			},
			want: false,
		},
		{
			name: "false when region is not supported",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "testProvider",
						},
					},
				},
			},
			args: args{
				providerName: "testProvider",
				regionName:   "testRegion",
			},
			want: false,
		},
		{
			name: "true when region is supported",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "testProvider",
							Regions: config.RegionList{
								config.Region{
									Name: "testRegion",
								},
							},
						},
					},
				},
			},
			args: args{
				providerName: "testProvider",
				regionName:   "testRegion",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			if got := c.IsRegionSupportedForProvider(tt.args.providerName, tt.args.regionName); got != tt.want {
				t.Errorf("IsRegionSupportedForProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_configService_IsAllowListEnabled(t *testing.T) {
	tests := []struct {
		name    string
		service configService
		want    bool
	}{
		{
			name: "return 'false' when allow list feature disabled",
			service: configService{
				allowListConfig: config.AllowListConfig{
					EnableAllowList: false,
				},
			},
			want: false,
		},
		{
			name: "return 'true' when allow list feature enabled",
			service: configService{
				allowListConfig: config.AllowListConfig{
					EnableAllowList: true,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			enabled := tt.service.IsAllowListEnabled()
			Expect(enabled).To(Equal(tt.want))
		})
	}
}
func Test_configService_GetOrganisationById(t *testing.T) {
	type result struct {
		found        bool
		organisation config.Organisation
	}

	tests := []struct {
		name    string
		service configService
		arg     string
		want    result
	}{
		{
			name: "return 'false' when organisation does not exist in the allowed list",
			arg:  "some-id",
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id: "different-id",
							},
						},
					},
				},
			},
			want: result{
				found:        false,
				organisation: config.Organisation{},
			},
		},
		{
			name: "return 'true' when organisation exists in the allowed list",
			arg:  "some-id",
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							config.Organisation{
								Id: "some-id",
							},
						},
					},
				},
			},
			want: result{
				found: true,
				organisation: config.Organisation{
					Id: "some-id",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			org, found := tt.service.GetOrganisationById(tt.arg)
			Expect(org).To(Equal(tt.want.organisation))
			Expect(found).To(Equal(tt.want.found))
		})
	}
}

func Test_configService_IsUserAllowed(t *testing.T) {
	type args struct {
		organisation config.Organisation
		username     string
	}

	organisation := config.Organisation{
		Id: "some-id",
		AllowedUsers: config.AllowedUsers{
			config.AllowedUser{Username: "username-0"},
			config.AllowedUser{Username: "username-1"},
		},
	}

	tests := []struct {
		name    string
		service configService
		arg     args
		want    bool
	}{
		{
			name: "return 'false' when organisation has not list of allowed users and AllowAll is set to false",
			arg: args{
				username: "username-0",
				organisation: config.Organisation{
					AllowAll: false,
				},
			},
			service: configService{},
			want:    false,
		},
		{
			name: "return 'true' when organisation has not list of allowed users and AllowAll is set to true",
			arg: args{
				username: "username-0",
				organisation: config.Organisation{
					AllowAll: true,
				},
			},
			service: configService{},
			want:    true,
		},
		{
			name: "return 'true' when organisation contains the user",
			arg: args{
				username:     "username-0",
				organisation: organisation,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "return 'false' when user is not allowed within the organisation",
			arg: args{
				username:     "username-10",
				organisation: organisation,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "return 'true' when user is not among the listed organisation but is contained in list of allowed users",
			arg: args{
				username:     "username-10",
				organisation: organisation,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
						AllowedUsers: config.AllowedUsers{
							config.AllowedUser{Username: "username-0"},
							config.AllowedUser{Username: "username-10"},
							config.AllowedUser{Username: "username-3"},
						},
					},
				},
			},
			want: true,
		},

		{
			name: "return 'false' when user is not among the listed organisation and in list of allowed users",
			arg: args{
				username:     "username-10",
				organisation: organisation,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
						AllowedUsers: config.AllowedUsers{
							config.AllowedUser{Username: "username-0"},
							config.AllowedUser{Username: "username-3"},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			allowed := tt.service.IsUserAllowed(tt.arg.username, tt.arg.organisation)
			Expect(allowed).To(Equal(tt.want))
		})
	}
}

func Test_configService_GetAllowedUserByUsernameAndOrgId(t *testing.T) {
	type args struct {
		username string
		orgId    string
	}

	type result struct {
		allowedUser config.AllowedUser
		found       bool
	}

	organisation := config.Organisation{
		Id: "some-id",
		AllowedUsers: config.AllowedUsers{
			config.AllowedUser{Username: "username-0"},
			config.AllowedUser{Username: "username-1"},
		},
	}

	tests := []struct {
		name    string
		service configService
		arg     args
		want    result
	}{
		{
			name: "return 'true' and the found user when organisation contains the user",
			arg: args{
				username: "username-1",
				orgId:    organisation.Id,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
					},
				},
			},
			want: result{
				found:       true,
				allowedUser: config.AllowedUser{Username: "username-1"},
			},
		},
		{
			name: "return 'true' and the user when user is not among the listed organisation but is contained in list of allowed users",
			arg: args{
				username: "username-10",
				orgId:    organisation.Id,
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
						AllowedUsers: config.AllowedUsers{
							config.AllowedUser{Username: "username-0"},
							config.AllowedUser{Username: "username-10"},
							config.AllowedUser{Username: "username-3"},
						},
					},
				},
			},
			want: result{
				found:       true,
				allowedUser: config.AllowedUser{Username: "username-10"},
			},
		},
		{
			name: "return 'false' when user is not among the listed organisation and in list of allowed users",
			arg: args{
				username: "username-10",
				orgId:    "some-org-id",
			},
			service: configService{
				allowListConfig: config.AllowListConfig{
					AllowList: config.AllowListConfiguration{
						Organisations: config.OrganisationList{
							organisation,
						},
						AllowedUsers: config.AllowedUsers{
							config.AllowedUser{Username: "username-0"},
							config.AllowedUser{Username: "username-3"},
						},
					},
				},
			},
			want: result{
				found: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			user, ok := tt.service.GetAllowedUserByUsernameAndOrgId(tt.arg.username, tt.arg.orgId)
			Expect(user).To(Equal(tt.want.allowedUser))
			Expect(ok).To(Equal(tt.want.found))
		})
	}
}

func Test_configService_Validate(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "error when no default provider provided",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test",
							Default: false,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when no default region in default provider",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test",
							Default: true,
							Regions: config.RegionList{
								config.Region{
									Name:    "test",
									Default: false,
								},
							},
						},
						config.Provider{
							Name: "test",
							Regions: config.RegionList{
								config.Region{
									Name:    "test",
									Default: true,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when multiple default providers provided",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test1",
							Default: true,
						},
						config.Provider{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error when multiple default regions in default provider",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test",
							Default: true,
							Regions: config.RegionList{
								config.Region{
									Name:    "test1",
									Default: true,
								},
								config.Region{
									Name:    "test2",
									Default: true,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success when default provider and region provided",
			fields: fields{
				providersConfig: config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "test",
							Default: true,
							Regions: config.RegionList{
								config.Region{
									Name:    "test1",
									Default: true,
								},
								config.Region{
									Name:    "test2",
									Default: false,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{providersConfig: tt.fields.providersConfig}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configService_validateProvider(t *testing.T) {
	type fields struct {
		providersConfig config.ProviderConfiguration
	}
	type args struct {
		provider config.Provider
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when no default region in provider",
			args: args{
				provider: config.Provider{
					Name:    "test",
					Default: false,
				},
			},
			wantErr: true,
		},
		{
			name: "error when more than one default region in provider",
			args: args{
				provider: config.Provider{
					Name: "test",
					Regions: config.RegionList{
						config.Region{
							Name:    "test1",
							Default: true,
						},
						config.Region{
							Name:    "test2",
							Default: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "success when default region provided",
			args: args{
				provider: config.Provider{
					Name: "test",
					Regions: config.RegionList{
						config.Region{
							Name:    "test",
							Default: true,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
			if err := c.validateProvider(tt.args.provider); (err != nil) != tt.wantErr {
				t.Errorf("validateProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_configService_IsAutoCreateOSDEnabled(t *testing.T) {
	type fields struct {
	}

	tests := []struct {
		name         string
		serverConfig config.ServerConfig
		want         bool
	}{
		{
			name:         "return true if auto osd creation is enabled",
			serverConfig: config.ServerConfig{AutoOSDCreation: true},
			want:         true,
		},
		{

			name:         "return false if auto osd creation is disabled",
			serverConfig: config.ServerConfig{AutoOSDCreation: false},
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			c := configService{
				serverConfig: tt.serverConfig,
			}
			enabled := c.IsAutoCreateOSDEnabled()
			Expect(enabled).To(Equal(tt.want))
		})
	}
}
