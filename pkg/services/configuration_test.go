package services

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"reflect"
	"testing"
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
			c := configService{
				providersConfig: tt.fields.providersConfig,
			}
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
