package handlers

import (
	"context"
	"net/http"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	. "github.com/onsi/gomega"
)

func Test_Validation_validateCloudProvider(t *testing.T) {
	type args struct {
		kafkaRequest  openapi.KafkaRequestPayload
		configService services.ConfigService
	}

	type result struct {
		wantErr      bool
		reason       string
		kafkaRequest openapi.KafkaRequest
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error when default provider and region are picked",
			arg: args{
				kafkaRequest: openapi.KafkaRequestPayload{},
				configService: services.NewConfigService(
					config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "aws",
								Default: true,
								Regions: config.RegionList{
									config.Region{
										Name:    "us-east-1",
										Default: true,
									},
								},
							},
						},
					},
					config.AllowListConfig{},
					config.ServerConfig{},
					config.ObservabilityConfiguration{}),
			},
			want: result{
				wantErr: false,
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "do not throw an error when cloud provider and region matches",
			arg: args{
				kafkaRequest: openapi.KafkaRequestPayload{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "gcp",
								Regions: config.RegionList{
									config.Region{
										Name: "eu-east-1",
									},
								},
							},
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name: "us-east-1",
									},
								},
							},
						},
					},
					config.AllowListConfig{},
					config.ServerConfig{},
					config.ObservabilityConfiguration{},
				),
			},
			want: result{
				wantErr: false,
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "throws an error when cloud provider and region do not match",
			arg: args{
				kafkaRequest: openapi.KafkaRequestPayload{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name: "us-east-1",
									},
								},
							},
						},
					},
					config.AllowListConfig{},
					config.ServerConfig{},
					config.ObservabilityConfiguration{}),
			},
			want: result{
				wantErr: true,
				reason:  "region us-east is not supported for aws, supported regions are: [us-east-1]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			validateFn := validateCloudProvider(&tt.arg.kafkaRequest, tt.arg.configService, "creating-kafka")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				Expect(err.Reason).To(Equal(tt.want.reason))
				return
			}

			Expect(tt.want.wantErr).To(Equal(err != nil))

			if !tt.want.wantErr {
				Expect(tt.arg.kafkaRequest.CloudProvider).To(Equal(tt.want.kafkaRequest.CloudProvider))
				Expect(tt.arg.kafkaRequest.Region).To(Equal(tt.want.kafkaRequest.Region))
			}

		})
	}
}

func Test_Validation_validateMaxAllowedInstances(t *testing.T) {
	type args struct {
		kafkaService  services.KafkaService
		configService services.ConfigService
		context       context.Context
	}

	username := "username"

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "do not throw an error when allow list disabled",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return api.KafkaList{}, nil, nil
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: false,
					},
					config.ServerConfig{},
					config.ObservabilityConfiguration{},
				),
				context: context.TODO(),
			},
			want: nil,
		},
		{
			name: "throw an error when the KafkaService call throws an error",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, errors.GeneralError("count failed from database")
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: true,
						AllowList: config.AllowListConfiguration{
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{
									Username:            username,
									MaxAllowedInstances: 4,
								},
							},
						},
					}, config.ServerConfig{},
					config.ObservabilityConfiguration{}),
				context: context.TODO(),
			},
			want: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding allowed organisation limits",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, nil
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: true,
						AllowList: config.AllowListConfiguration{
							Organisations: config.OrganisationList{
								config.Organisation{
									Id:                  "org-id",
									MaxAllowedInstances: 4,
									AllowAll:            true,
								},
							},
						},
					},
					config.ServerConfig{},
					config.ObservabilityConfiguration{},
				),
				context: auth.SetOrgIdContext(auth.SetUsernameContext(context.TODO(), username), "org-id"),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organisation 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     4,
			},
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding allowed limits",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, nil
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: true,
						AllowList: config.AllowListConfiguration{
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{
									Username:            username,
									MaxAllowedInstances: 4,
								},
							},
						},
					},
					config.ServerConfig{},
					config.ObservabilityConfiguration{},
				),
				context: auth.SetOrgIdContext(auth.SetUsernameContext(context.TODO(), username), "org-id"),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 4 allowed instances.",
				Code:     4,
			},
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding default allowed limits of 1 instance",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: true,
						AllowList: config.AllowListConfiguration{
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{
									Username: username,
								},
							},
						},
					},
					config.ServerConfig{},
					config.ObservabilityConfiguration{},
				),
				context: auth.SetOrgIdContext(auth.SetUsernameContext(context.TODO(), username), "org-id"),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     4,
			},
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding default allowed limits of 1 instance and the user is not listed in the allowed service accounts list",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				configService: services.NewConfigService(
					config.ProviderConfiguration{},
					config.AllowListConfig{
						EnableAllowList: true,
					},
					config.ServerConfig{},
					config.ObservabilityConfiguration{}),
				context: auth.SetOrgIdContext(auth.SetUsernameContext(context.TODO(), username), "org-id"),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			validateFn := validateMaxAllowedInstances(tt.arg.kafkaService, tt.arg.configService, tt.arg.context)
			err := validateFn()
			Expect(tt.want).To(Equal(err))
		})
	}
}

func Test_Validations_validateKafkaClusterNames(t *testing.T) {
	tests := []struct {
		description string
		name        string
		expectError bool
	}{
		{
			description: "valid kafka cluster name",
			name:        "test-kafka1",
			expectError: false,
		},
		{
			description: "valid kafka cluster name with multiple '-'",
			name:        "test-my-cluster",
			expectError: false,
		},
		{
			description: "invalid kafka cluster name begins with number",
			name:        "1test-cluster",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with invalid characters",
			name:        "test-c%*_2",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with upper-case letters",
			name:        "Test-cluster",
			expectError: true,
		},
		{
			description: "invalid kafka cluster name with spaces",
			name:        "test cluster",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			RegisterTestingT(t)
			validateFn := validateRegexp(validKafkaClusterNameRegexp, &tt.name, "name")
			err := validateFn()
			if tt.expectError {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	}
}
