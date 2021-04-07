package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	. "github.com/onsi/gomega"
)

const (
	jwtKeyFile = "test/support/jwt_private_key.pem"
	jwtCAFile  = "test/support/jwt_ca.pem"
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
					config.ApplicationConfig{
						SupportedProviders: &config.ProviderConfig{
							ProvidersConfig: config.ProviderConfiguration{
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
						},
					}),
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
					config.ApplicationConfig{
						SupportedProviders: &config.ProviderConfig{
							ProvidersConfig: config.ProviderConfiguration{
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
						},
					},
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
				configService: services.NewConfigService(config.ApplicationConfig{
					SupportedProviders: &config.ProviderConfig{
						ProvidersConfig: config.ProviderConfiguration{
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
					},
				}),
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

	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, environments.Environment().Config.OCM.TokenIssuerURL)
	if err != nil {
		t.Fatalf("failed to create auth helper: %s", err.Error())
	}
	account, err := authHelper.NewAccount("username", "", "", "org-id")
	if err != nil {
		t.Fatal("failed to build a new account")
	}

	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	if err != nil {
		t.Fatalf("failed to create jwt: %s", err.Error())
	}
	authenticatedCtx := auth.SetTokenInContext(context.TODO(), jwt)

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "do not throw an error when instance limit control is disabled",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return api.KafkaList{}, &api.PagingMeta{}, nil
					},
				},
				configService: services.NewConfigService(
					config.ApplicationConfig{
						AccessControlList: &config.AccessControlListConfig{
							EnableInstanceLimitControl: false,
						},
					},
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
				configService: services.NewConfigService(config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableInstanceLimitControl: true,
						AllowList: config.AllowListConfiguration{
							ServiceAccounts: config.AllowedAccounts{
								config.AllowedAccount{
									Username:            account.Username(),
									MaxAllowedInstances: 4,
								},
							},
						},
					},
				}),
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
					config.ApplicationConfig{
						AccessControlList: &config.AccessControlListConfig{
							EnableInstanceLimitControl: true,
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
					},
				),
				context: authenticatedCtx,
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organisation 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     5,
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
					config.ApplicationConfig{
						AccessControlList: &config.AccessControlListConfig{
							EnableInstanceLimitControl: true,
							AllowList: config.AllowListConfiguration{
								ServiceAccounts: config.AllowedAccounts{
									config.AllowedAccount{
										Username:            account.Username(),
										MaxAllowedInstances: 4,
									},
								},
							},
						},
					},
				),
				context: authenticatedCtx,
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 4 allowed instances.",
				Code:     5,
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
					config.ApplicationConfig{
						AccessControlList: &config.AccessControlListConfig{
							EnableInstanceLimitControl: true,
							AllowList: config.AllowListConfiguration{
								ServiceAccounts: config.AllowedAccounts{
									config.AllowedAccount{
										Username: account.Username(),
									},
								},
							},
						},
					},
				),
				context: authenticatedCtx,
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
			},
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding default allowed limits of 1 instance and the user is not listed in the allow list",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					AccessControlList: &config.AccessControlListConfig{
						EnableInstanceLimitControl: true,
					},
				}),
				context: authenticatedCtx,
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
			},
		},
		{
			name: "throw an error if user is not allowed in their org and they cannot create any more instances after exceeding default allowed user limits",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				configService: services.NewConfigService(
					config.ApplicationConfig{
						AccessControlList: &config.AccessControlListConfig{
							EnableInstanceLimitControl: true,
							AllowList: config.AllowListConfiguration{
								Organisations: config.OrganisationList{
									config.Organisation{
										Id:                  "org-id",
										MaxAllowedInstances: 4,
										AllowAll:            false,
									},
								},
							},
						},
					},
				),
				context: authenticatedCtx,
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
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

func Test_Validation_validateKafkaClusterNameIsUnique(t *testing.T) {
	type args struct {
		kafkaService services.KafkaService
		name         string
		context      context.Context
	}

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "throw an error when the KafkaService call throws an error",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, errors.GeneralError("count failed from database")
					},
				},
				name:    "some-name",
				context: context.TODO(),
			},
			want: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when name is already used",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				name:    "duplicate-name",
				context: context.TODO(),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusConflict,
				Reason:   "Kafka cluster name is already used",
				Code:     36,
			},
		},
		{
			name: "does not throw an error when name is unique",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *services.ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 0}, nil
					},
				},
				name:    "unique-name",
				context: context.TODO(),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			validateFn := validateKafkaClusterNameIsUnique(&tt.arg.name, tt.arg.kafkaService, tt.arg.context)
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
			validateFn := validKafkaClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	}
}
