package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

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
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
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
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
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
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
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
			gomega.RegisterTestingT(t)
			validateFn := ValidateKafkaClusterNameIsUnique(&tt.arg.name, tt.arg.kafkaService, tt.arg.context)
			err := validateFn()
			gomega.Expect(tt.want).To(gomega.Equal(err))
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
			gomega.RegisterTestingT(t)
			validateFn := ValidKafkaClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				gomega.Expect(err).Should(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}
		})
	}
}

func Test_Validation_validateCloudProvider(t *testing.T) {
	limit := int(5)
	developerMap := config.InstanceTypeMap{
		"developer": {
			Limit: &limit,
		},
	}
	standardMap := config.InstanceTypeMap{
		"standard": {
			Limit: &limit,
		},
	}
	type args struct {
		kafkaRequest   dbapi.KafkaRequest
		ProviderConfig *config.ProviderConfig
		kafkaService   services.KafkaService
	}

	type result struct {
		wantErr      bool
		reason       string
		kafkaRequest public.KafkaRequest
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error when default provider and region are picked",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(kafkaRequest *dbapi.KafkaRequest) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: dbapi.KafkaRequest{},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "aws",
								Default: true,
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										Default:                true,
										SupportedInstanceTypes: developerMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: false,
				kafkaRequest: public.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "do not throw an error when cloud provider and region matches",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(kafkaRequest *dbapi.KafkaRequest) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "gcp",
								Regions: config.RegionList{
									config.Region{
										Name:                   "eu-east-1",
										SupportedInstanceTypes: developerMap,
									},
								},
							},
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										SupportedInstanceTypes: developerMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: false,
				kafkaRequest: public.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "throws an error when cloud provider and region do not match",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(kafkaRequest *dbapi.KafkaRequest) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										SupportedInstanceTypes: developerMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "region us-east is not supported for aws, supported regions are: [us-east-1]",
			},
		},
		{
			name: "throws an error when instance type is not supported",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(kafkaRequest *dbapi.KafkaRequest) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east",
										SupportedInstanceTypes: standardMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "instance type 'developer' not supported for region 'us-east'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidateCloudProvider(&tt.arg.kafkaService, &tt.arg.kafkaRequest, tt.arg.ProviderConfig, "creating-kafka")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				gomega.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
				return
			}

			gomega.Expect(tt.want.wantErr).To(gomega.Equal(err != nil))

			if !tt.want.wantErr {
				gomega.Expect(tt.arg.kafkaRequest.CloudProvider).To(gomega.Equal(tt.want.kafkaRequest.CloudProvider))
				gomega.Expect(tt.arg.kafkaRequest.Region).To(gomega.Equal(tt.want.kafkaRequest.Region))
			}

		})
	}
}

func Test_Validation_ValidateKafkaUserFacingUpdateFields(t *testing.T) {
	emptyOwner := ""
	newOwner := "some-owner"
	reauthenticationEnabled := true
	username := "username"
	orgId := "organisation_id"
	token := &jwt.Token{
		Claims: jwt.MapClaims{
			"username": username,
			"org_id":   orgId,
		},
	}
	type args struct {
		kafkaUpdateRequest public.KafkaUpdateRequest
		ctx                context.Context
		authService        authorization.Authorization
		kafka              *dbapi.KafkaRequest
	}

	type result struct {
		wantErr bool
		reason  string
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error if update payload is empty and current user is owner",
			arg: args{
				ctx: auth.SetTokenInContext(context.TODO(), token),
				kafka: &dbapi.KafkaRequest{
					Owner:          username,
					OrganisationId: orgId,
				},
				kafkaUpdateRequest: public.KafkaUpdateRequest{},
				authService:        authorization.NewMockAuthorization(),
			},
			want: result{
				wantErr: false,
			},
		},
		{
			name: "throw an error when user is not owner of the kafka and update payload is empty",
			arg: args{
				ctx:                auth.SetTokenInContext(context.TODO(), token),
				kafka:              &dbapi.KafkaRequest{},
				kafkaUpdateRequest: public.KafkaUpdateRequest{},
				authService:        authorization.NewMockAuthorization(),
			},
			want: result{
				wantErr: true,
				reason:  "User not authorized to perform this action",
			},
		},
		{
			name: "throw an error when empty owner passed",
			arg: args{
				ctx: auth.SetTokenInContext(context.TODO(), token),
				kafka: &dbapi.KafkaRequest{
					Owner:          username,
					OrganisationId: orgId,
				},
				kafkaUpdateRequest: public.KafkaUpdateRequest{
					Owner: &emptyOwner,
				},
				authService: authorization.NewMockAuthorization(),
			},
			want: result{
				wantErr: true,
				reason:  "owner is not valid. Minimum length 1 is required.",
			},
		},
		{
			name: "throw an error when an admin from another organisation tries to update kafka",
			arg: args{
				ctx: auth.SetTokenInContext(context.TODO(), &jwt.Token{
					Claims: jwt.MapClaims{
						"username":     username,
						"org_id":       orgId,
						"is_org_admin": true,
					},
				}),
				kafka: &dbapi.KafkaRequest{
					Owner:          username,
					OrganisationId: "some-other-organisation",
				},
				kafkaUpdateRequest: public.KafkaUpdateRequest{
					Owner:                   &newOwner,
					ReauthenticationEnabled: &reauthenticationEnabled,
				},
				authService: authorization.NewMockAuthorization(),
			},
			want: result{
				wantErr: true,
				reason:  "User not authorized to perform this action",
			},
		},
		{
			name: "throw an error when new owner does not belong to the organisation",
			arg: args{
				ctx: auth.SetTokenInContext(context.TODO(), &jwt.Token{
					Claims: jwt.MapClaims{
						"username":     username,
						"org_id":       orgId,
						"is_org_admin": true,
					},
				}),
				kafka: &dbapi.KafkaRequest{
					Owner:          username,
					OrganisationId: orgId,
				},
				kafkaUpdateRequest: public.KafkaUpdateRequest{
					Owner: &newOwner,
				},
				authService: &authorization.AuthorizationMock{
					CheckUserValidFunc: func(username, orgId string) (bool, error) {
						return false, nil
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "User some-owner does not belong in your organization",
			},
		},
		{
			name: "should succeed when all the validation passes",
			arg: args{
				ctx: auth.SetTokenInContext(context.TODO(), &jwt.Token{
					Claims: jwt.MapClaims{
						"username":     username,
						"org_id":       orgId,
						"is_org_admin": true,
					},
				}),
				kafka: &dbapi.KafkaRequest{
					Owner:          username,
					OrganisationId: orgId,
				},
				kafkaUpdateRequest: public.KafkaUpdateRequest{
					ReauthenticationEnabled: &reauthenticationEnabled,
					Owner:                   &newOwner,
				},
				authService: &authorization.AuthorizationMock{
					CheckUserValidFunc: func(username, orgId string) (bool, error) {
						return true, nil
					},
				},
			},
			want: result{
				wantErr: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidateKafkaUserFacingUpdateFields(tt.arg.ctx, tt.arg.authService, tt.arg.kafka, &tt.arg.kafkaUpdateRequest)
			err := validateFn()
			gomega.Expect(tt.want.wantErr).To(gomega.Equal(err != nil))
			if !tt.want.wantErr && err != nil {
				t.Errorf("ValidateKafkaUserFacingUpdateFields() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				gomega.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
				return
			}
		})
	}
}
