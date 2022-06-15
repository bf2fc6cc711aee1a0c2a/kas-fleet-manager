package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateKafkaClusterNameIsUnique(&tt.arg.name, tt.arg.kafkaService, tt.arg.context)
			err := validateFn()
			Expect(err).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidKafkaClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
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
		kafkaRequest   public.KafkaRequestPayload
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
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: public.KafkaRequestPayload{},
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
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: public.KafkaRequestPayload{
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
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: public.KafkaRequestPayload{
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
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: public.KafkaRequestPayload{
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
		{
			name: "should return an error if it fails to assign an instance type",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return "", errors.New(errors.ErrorGeneral, "error assigning instance type: ")
					},
				},
				kafkaRequest: public.KafkaRequestPayload{
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
				reason:  "error assigning instance type: KAFKAS-MGMT-9: error assigning instance type: ",
			},
		},
		{
			name: "should throw an error if the provider is not supported",
			arg: args{
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequest: public.KafkaRequestPayload{
					CloudProvider: "invalid_provider",
					Region:        "us-east",
				},
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
				wantErr: true,
				reason:  "provider invalid_provider is not supported, supported providers are: [aws]",
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateCloudProvider(context.Background(), &tt.arg.kafkaService, &tt.arg.kafkaRequest, tt.arg.ProviderConfig, "creating-kafka")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				Expect(err.Reason).To(Equal(tt.want.reason))
				return
			}

			Expect(err != nil).To(Equal(tt.want.wantErr))
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
		{
			name: "should throw an error if user is not valid",
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
						return false, errors.New(errors.ErrorGeneral, "Unable to update kafka request owner")
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "Unable to update kafka request owner",
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateKafkaUserFacingUpdateFields(tt.arg.ctx, tt.arg.authService, tt.arg.kafka, &tt.arg.kafkaUpdateRequest)
			err := validateFn()
			Expect(err != nil).To(Equal(tt.want.wantErr), "ValidateKafkaUserFacingUpdateFields() expected not to throw error but threw %v", err)
			if tt.want.wantErr {
				Expect(err.Reason).To(Equal(tt.want.reason))
				return
			}
		})
	}
}

func TestValidateBillingCloudAccountIdAndMarketplace(t *testing.T) {
	type args struct {
		ctx                 context.Context
		kafkaService        services.KafkaService
		kafkaRequestPayload *public.KafkaRequestPayload
	}
	marketPlace := "MarketPlace"
	BillingCloudAccountId := "1234"

	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return nil if BillingCloudAccountId and Marketplace is empty",
			args: args{
				ctx:                 context.Background(),
				kafkaRequestPayload: &public.KafkaRequestPayload{},
			},
			want: nil,
		},
		{
			name: "should return an error if BillingCloudAccountId is empty and marketplace is not empty",
			args: args{
				ctx: context.Background(),
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Marketplace: &marketPlace,
				},
			},
			want: errors.InvalidBillingAccount("no billing account provided for marketplace: %s", marketPlace),
		},
		{
			name: "should return nil if it can validate the Billing CloudAccountId And Marketplace",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Marketplace:           &marketPlace,
					BillingCloudAccountId: &BillingCloudAccountId,
				},
			},
			want: nil,
		},
		{
			name: "should return an error if it cannot validate the Billing CloudAccountId And Marketplace",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return errors.GeneralError("error assigning instance type")
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Marketplace:           &marketPlace,
					BillingCloudAccountId: &BillingCloudAccountId,
				},
			},
			want: errors.GeneralError("error assigning instance type"),
		},
		{
			name: "should return an error if it fails to assign an instance type",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return "", errors.New(errors.ErrorGeneral, "error assigning instance type: ")
					},
					ValidateBillingAccountFunc: func(externalId string, instanceType types.KafkaInstanceType, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
						return nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Marketplace:           &marketPlace,
					BillingCloudAccountId: &BillingCloudAccountId,
				},
			},
			want: errors.NewWithCause(errors.ErrorGeneral, errors.GeneralError("error assigning instance type: "), "error assigning instance type: KAFKAS-MGMT-9: error assigning instance type: "),
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateBillingCloudAccountIdAndMarketplace(tt.args.ctx, &tt.args.kafkaService, tt.args.kafkaRequestPayload)
			err := validateFn()
			g.Expect(err).To(Equal(tt.want))
		})
	}
}

func TestValidateKafkaPlan(t *testing.T) {
	type args struct {
		ctx                 context.Context
		kafkaService        services.KafkaService
		kafkaConfig         *config.KafkaConfig
		kafkaRequestPayload *public.KafkaRequestPayload
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should not return an error if the kafka plan was successfully validated",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Plan: fmt.Sprintf("%s.x1", types.DEVELOPER.String()),
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "should not return an error if KafkaRequestPayload.plan in not set and the kafka plan was successfully validated",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "should return an error if the plan provided is invalid ",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Plan: fmt.Sprintf("%s.x1", "invalid_plan"),
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: errors.BadRequest("Unable to detect instance type in plan provided: 'invalid_plan.x1'"),
		},
		{
			name: "should return an error if the plan provided is not supported",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Plan: "developer.x2",
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: errors.InstancePlanNotSupported("Unsupported plan provided: 'developer.x2'"),
		},
		{
			name: "should return an error if KafkaRequestPayload.plan in not set and the kafka plan is not supported",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.KafkaInstanceType(types.STANDARD.String()), nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: errors.InstanceTypeNotSupported("Unsupported kafka instance type: 'standard' provided"),
		},
		{
			name: "should return an error if it is unable to detect instance size in plan provided",
			args: args{
				ctx: context.Background(),
				kafkaService: &services.KafkaServiceMock{
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.DEVELOPER, nil
					},
				},
				kafkaRequestPayload: &public.KafkaRequestPayload{
					Plan: "developer.invalidPlan",
				},
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id:          "developer",
									DisplayName: "Trial",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:          "x1",
											DisplayName: "1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: errors.InstancePlanNotSupported("Unsupported plan provided: 'developer.invalidPlan'"),
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateKafkaPlan(tt.args.ctx, &tt.args.kafkaService, tt.args.kafkaConfig, tt.args.kafkaRequestPayload)
			err := validateFn()
			g.Expect(err).To(Equal(tt.want))
		})
	}
}

func TestValidateKafkaUpdateFields(t *testing.T) {
	type args struct {
		kafkaUpdateRequest *private.KafkaUpdateRequest
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return nil if it validates kafka update fields successfully",
			args: args{
				kafkaUpdateRequest: &private.KafkaUpdateRequest{
					StrimziVersion:   "StrimziVersion",
					KafkaVersion:     "KafkaVersion",
					KafkaIbpVersion:  "KafkaIbpVersion",
					KafkaStorageSize: "KafkaStorageSize",
				},
			},
			want: nil,
		},
		{
			name: "should return error if all fields are empty",
			args: args{
				kafkaUpdateRequest: &private.KafkaUpdateRequest{
					StrimziVersion:   "",
					KafkaVersion:     "",
					KafkaIbpVersion:  "",
					KafkaStorageSize: "",
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Expecting at least one of the following fields: strimzi_version, kafka_version, kafka_ibp_version or kafka_storage_size to be provided"),
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateKafkaUpdateFields(tt.args.kafkaUpdateRequest)
			err := validateFn()
			g.Expect(err).To(Equal(tt.want))
		})
	}
}

func TestValidateKafkaStorageSize(t *testing.T) {
	type args struct {
		kafkaRequest   *dbapi.KafkaRequest
		kafkaUpdateReq *private.KafkaUpdateRequest
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return nil if it successfully validates the kafka storage size",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkaStorageSize: "2",
				},
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					KafkaStorageSize: "2",
				},
			},
			want: nil,
		},
		{
			name: "should return an error if the kafka request storage size is missing",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{},
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					KafkaStorageSize: "2",
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current storage size: ''"),
		},
		{
			name: "should return an error if it is unable to parse the kafka update request storage size parameter",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkaStorageSize: "2",
				},
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					KafkaStorageSize: "string",
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current requested size: 'string'"),
		},
		{
			name: "should return an error if the the kafka request storage size parameter is greater than the the kafka update request storage size parameter",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					KafkaStorageSize: "3",
				},
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					KafkaStorageSize: "2",
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Requested size: '2' should be greater than current size: '3'"),
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateFn := ValidateKafkaStorageSize(tt.args.kafkaRequest, tt.args.kafkaUpdateReq)
			err := validateFn()
			g.Expect(err).To(Equal(tt.want))
		})
	}
}
