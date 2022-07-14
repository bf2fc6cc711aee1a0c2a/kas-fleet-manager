package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateKafkaClusterNameIsUnique(&tt.arg.name, tt.arg.kafkaService, tt.arg.context)
			err := validateFn()
			g.Expect(err).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidKafkaClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				g.Expect(err).Should(gomega.HaveOccurred())
			} else {
				g.Expect(err).ShouldNot(gomega.HaveOccurred())
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateCloudProvider(context.Background(), &tt.arg.kafkaService, &tt.arg.kafkaRequest, tt.arg.ProviderConfig, "creating-kafka")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				g.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
				return
			}

			g.Expect(err != nil).To(gomega.Equal(tt.want.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateKafkaUserFacingUpdateFields(tt.arg.ctx, tt.arg.authService, tt.arg.kafka, &tt.arg.kafkaUpdateRequest)
			err := validateFn()
			g.Expect(err != nil).To(gomega.Equal(tt.want.wantErr), "ValidateKafkaUserFacingUpdateFields() expected not to throw error but threw %v", err)
			if tt.want.wantErr {
				g.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
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
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateBillingCloudAccountIdAndMarketplace(tt.args.ctx, &tt.args.kafkaService, tt.args.kafkaRequestPayload)
			err := validateFn()
			g.Expect(err).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateKafkaPlan(tt.args.ctx, &tt.args.kafkaService, tt.args.kafkaConfig, tt.args.kafkaRequestPayload)
			err := validateFn()
			g.Expect(err).To(gomega.Equal(tt.want))
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
					StrimziVersion:             "StrimziVersion",
					KafkaVersion:               "KafkaVersion",
					KafkaIbpVersion:            "KafkaIbpVersion",
					DeprecatedKafkaStorageSize: "KafkaStorageSize",
				},
			},
			want: nil,
		},
		{
			name: "should return error if all fields are empty",
			args: args{
				kafkaUpdateRequest: &private.KafkaUpdateRequest{
					StrimziVersion:             "",
					KafkaVersion:               "",
					KafkaIbpVersion:            "",
					DeprecatedKafkaStorageSize: "",
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Expecting at least one of the following fields: strimzi_version, kafka_version, kafka_ibp_version, kafka_storage_size or max_data_retention_size to be provided"),
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			validateFn := ValidateKafkaUpdateFields(tt.args.kafkaUpdateRequest)
			err := validateFn()
			g.Expect(err).To(gomega.Equal(tt.want))
		})
	}
}

func TestValidateKafkaStorageSize(t *testing.T) {
	type args struct {
		kafkaRequest   *dbapi.KafkaRequest
		kafkaUpdateReq *private.KafkaUpdateRequest
	}

	currentStorageSize := "1Gi"
	increaseStorageSizeReq := "10Gi"
	decreaseStorageSizeReq := "10Mi"
	invalidStorageSize := "2abc"

	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "should return nil if kafka_storage_size or max_data_retention_size is not specified",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{},
			},
			want: nil,
		},
		{
			name: "should return nil if specified kafka storage size is valid",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: increaseStorageSizeReq,
				},
			},
			want: nil,
		},
		{
			name: "should return an error if it is unable to parse kafka_storage_size",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: invalidStorageSize,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current requested size: '%s'", invalidStorageSize),
		},
		{
			name: "should return an error if the the kafka request storage size parameter is greater than the the kafka update request storage size parameter",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: decreaseStorageSizeReq,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Requested size: '%s' should be greater than current size: '%s'", decreaseStorageSizeReq, currentStorageSize),
		},
		{
			name: "should return nil if specified max data retention size is valid",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					MaxDataRetentionSize: increaseStorageSizeReq,
				},
			},
			want: nil,
		},
		{
			name: "should return an error if it is unable to parse max_data_retention_size",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					MaxDataRetentionSize: invalidStorageSize,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current requested size: '%s'", invalidStorageSize),
		},
		{
			name: "should return an error if specified max data retention size is lower than current size",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, currentStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					MaxDataRetentionSize: decreaseStorageSizeReq,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Requested size: '%s' should be greater than current size: '%s'", decreaseStorageSizeReq, currentStorageSize),
		},
		{
			name: "should return an error if the current kafka request storage size is an empty string",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: increaseStorageSizeReq,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current storage size: ''"),
		},
		{
			name: "should return an error if the current kafka request storage size is an invalid Quantity value",
			args: args{
				kafkaRequest: mockkafka.BuildKafkaRequest(
					mockkafka.With(mockkafka.STORAGE_SIZE, invalidStorageSize),
				),
				kafkaUpdateReq: &private.KafkaUpdateRequest{
					DeprecatedKafkaStorageSize: increaseStorageSizeReq,
				},
			},
			want: errors.FieldValidationError("Failed to update Kafka Request. Unable to parse current storage size: '%s'", invalidStorageSize),
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			validateFn := ValidateKafkaStorageSize(tt.args.kafkaRequest, tt.args.kafkaUpdateReq)
			err := validateFn()
			g.Expect(err).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Validation_validateBillingModel(t *testing.T) {
	type args struct {
		kafkaRequest public.KafkaRequestPayload
	}

	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "do not throw an error when billing model is not provided",
			arg: args{

				kafkaRequest: public.KafkaRequestPayload{
					BillingModel: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "do not throw an error when marketplace model is provided",
			arg: args{
				kafkaRequest: public.KafkaRequestPayload{
					BillingModel: &[]string{"marketplace"}[0],
				},
			},
			wantErr: false,
		},
		{
			name: "do not throw an error when standard model is provided",
			arg: args{
				kafkaRequest: public.KafkaRequestPayload{
					BillingModel: &[]string{"standard"}[0],
				},
			},
			wantErr: false,
		},
		{
			name: "throw an error when unknown model is provided",
			arg: args{
				kafkaRequest: public.KafkaRequestPayload{
					BillingModel: &[]string{"xyz"}[0],
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			validateFn := ValidateBillingModel(&tt.arg.kafkaRequest)
			err := validateFn()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_validateVersionsCompatibility(t *testing.T) {
	type args struct {
		h              *adminKafkaHandler
		kafkaRequest   dbapi.KafkaRequest
		kafkaUpdateReq private.KafkaUpdateRequest
	}

	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"
	availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		wantStatusCode int
	}{
		{
			name: "should return error if cluster associated with kafka request cannot be found.",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return nil, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "test",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					KafkaIBPUpgrading:      true,
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if IsStrimziKafkaVersionAvailableInCluster returns an error",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, errors.GeneralError("test")
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if strimzi version is not available",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return false, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if strimzi verson is not ready",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return false, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if CheckStrimziVersionReady returns error",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, errors.GeneralError("test")
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if actual ibp version is greater than desired ibp version",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.8",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.7",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if actual ibp version and desired ibp version cannot be compared",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:             "cluster-id",
					ActualKafkaIBPVersion: "2.8",
					ActualKafkaVersion:    "2.7",
					DesiredKafkaVersion:   "2.7",
					DesiredStrimziVersion: "2.7",
					KafkaStorageSize:      "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion: "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:   "2.8.2",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if desired ibp version and desired kafka version cannot be compared.",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if desired ibp version is greater than desired kafka version",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.8",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.9.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return error if desired kafka version is a downgrade.",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.4",
					DesiredKafkaIBPVersion: "2.5",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.6",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.6.2",
					KafkaIbpVersion: "2.5.1",
				},
			},
			wantErr: true,
		},
		{
			name: "should return nil if it can Verify kafka",
			args: args{
				h: &adminKafkaHandler{
					clusterService: &services.ClusterServiceMock{
						FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
							return &api.Cluster{
								Meta: api.Meta{
									ID: "id",
								},
								ClusterID:                "cluster-id",
								AvailableStrimziVersions: availableStrimziVersions,
							}, nil
						},
						IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
							return true, nil
						},
						CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
							return true, nil
						},
					},
				},
				kafkaRequest: dbapi.KafkaRequest{
					Status: constants.KafkaRequestStatusPreparing.String(),
					Meta: api.Meta{
						ID: "id",
					},
					ClusterID:              "cluster-id",
					ActualKafkaIBPVersion:  "2.7",
					DesiredKafkaIBPVersion: "2.7",
					ActualKafkaVersion:     "2.7",
					DesiredKafkaVersion:    "2.7",
					DesiredStrimziVersion:  "2.7",
					KafkaStorageSize:       "100",
				},
				kafkaUpdateReq: private.KafkaUpdateRequest{
					StrimziVersion:  "strimzi-cluster-operator.v0.24.0-0",
					KafkaVersion:    "2.8.2",
					KafkaIbpVersion: "2.8.1",
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			validate := validateVersionsCompatibility(tt.args.h, &tt.args.kafkaRequest, &tt.args.kafkaUpdateReq)
			err := validate()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
