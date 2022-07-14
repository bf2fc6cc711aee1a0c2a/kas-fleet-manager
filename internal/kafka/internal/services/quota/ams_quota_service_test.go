package quota

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var ocmClientMockWithCloudAccounts = &ocm.ClientMock{
	ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
		ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
		return ca, nil
	},
	GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
		return fmt.Sprintf("fake-org-id-%s", externalId), nil
	},
	GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
		if product != string(ocm.RHOSAKProduct) {
			return []*v1.QuotaCost{}, nil
		}
		rrbq1 := v1.NewRelatedResource().
			BillingModel(string(v1.BillingModelStandard)).
			Product(string(ocm.RHOSAKProduct)).
			ResourceName(resourceName).
			Cost(1)

		qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).
			CloudAccounts(
				v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
				v1.NewCloudAccount().CloudProviderID("ibm").CloudAccountID("2345678901"),
				v1.NewCloudAccount().CloudProviderID("azure").CloudAccountID("1234567890"),
			).
			Build()
		if err != nil {
			panic("unexpected error")
		}

		return []*v1.QuotaCost{qcb}, nil
	},
}

func Test_AMSGetBillingModel(t *testing.T) {
	type fields struct {
		ocmClient   ocm.Client
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		request dbapi.KafkaRequest
	}
	tests := []struct {
		name                     string
		fields                   fields
		args                     args
		wantErr                  bool
		wantMatch                bool
		wantBillingModel         string
		wantInferredBillingModel string
	}{
		{
			name: "aws marketplace billing model is detected",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "aws",
					InstanceType:          "standard",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(
								v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
							).
							Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantBillingModel:         string(v1.BillingModelMarketplaceAWS),
			wantInferredBillingModel: string(v1.BillingModelMarketplace),
		},
		{
			name: "rhm marketplace billing model is detected",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "rhm",
					InstanceType:          "standard",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(
								v1.NewCloudAccount().CloudProviderID("rhm").CloudAccountID("1234567890"),
							).
							Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantBillingModel:         string(v1.BillingModelMarketplace),
			wantInferredBillingModel: string(v1.BillingModelMarketplace),
		},
		{
			name: "standard billing model is detected when no billing account provided",
			args: args{
				request: dbapi.KafkaRequest{
					Name:         "test",
					SizeId:       "x1",
					InstanceType: "standard",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).CloudAccounts(
							v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
						).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantBillingModel:         string(v1.BillingModelStandard),
			wantInferredBillingModel: string(v1.BillingModelStandard),
		},
		{
			name: "marketplace billing model is rejected when no marketplace related resource is found",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					InstanceType:          "standard",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "aws",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantBillingModel:         "",
			wantInferredBillingModel: "",
		},
		{
			name: "marketplace billing model is rejected when quota is insufficient",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					InstanceType:          "standard",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "aws",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).CloudAccounts(
							v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
						).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantBillingModel:         "",
			wantInferredBillingModel: "",
		},
		{
			name: "unsupported cloud provider is rejected",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					InstanceType:          "standard",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "xyz",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).CloudAccounts(
							v1.NewCloudAccount().CloudProviderID("xyz").CloudAccountID("1234567890"),
						).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantBillingModel:         "",
			wantInferredBillingModel: "",
		},
		{
			name: "standard billing is preferred when no billing account is provided",
			args: args{
				request: dbapi.KafkaRequest{
					Name:         "test",
					SizeId:       "x1",
					InstanceType: "standard",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						rrbq2 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).CloudAccounts(
							v1.NewCloudAccount().CloudProviderID("xyz").CloudAccountID("1234567890"),
						).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantBillingModel:         string(v1.BillingModelStandard),
			wantInferredBillingModel: string(v1.BillingModelStandard),
		},
		{
			name: "marketplace billing is selected when billing account is provided",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					InstanceType:          "standard",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "aws",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						rrbq2 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).CloudAccounts(
							v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
						).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantBillingModel:         string(v1.BillingModelMarketplaceAWS),
			wantInferredBillingModel: string(v1.BillingModelMarketplace),
		},
		{
			name: "detects non-matching requested billing model (standard)",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                  "test",
					SizeId:                "x1",
					BillingCloudAccountId: "1234567890",
					Marketplace:           "aws",
					InstanceType:          "standard",
					BillingModel:          "standard",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelMarketplace)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(
								v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("1234567890"),
							).
							Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                false,
			wantBillingModel:         string(v1.BillingModelMarketplaceAWS),
			wantInferredBillingModel: "",
		},
		{
			name: "detects non-matching requested billing model (marketplace)",
			args: args{
				request: dbapi.KafkaRequest{
					Name:         "test",
					SizeId:       "x1",
					InstanceType: "standard",
					BillingModel: "marketplace",
				},
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                false,
			wantBillingModel:         string(v1.BillingModelStandard),
			wantInferredBillingModel: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)

			billingModel, err := quotaService.(*amsQuotaService).getBillingModel(&tt.args.request, types.STANDARD)
			g.Expect(billingModel).To(gomega.Equal(tt.wantBillingModel))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			match, bm := quotaService.(*amsQuotaService).billingModelMatches(billingModel, tt.args.request.BillingModel)
			g.Expect(match).To(gomega.Equal(tt.wantMatch))
			g.Expect(bm).To(gomega.Equal(tt.wantInferredBillingModel))
		})
	}

}

func Test_AMSValidateBillingAccount(t *testing.T) {
	type fields struct {
		ocmClient   ocm.Client
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		orgId            string
		billingAccountId string
		marketplace      *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "check passes when requested billing account is found in account list",
			args: args{
				"test",
				"1234567890",
				&[]string{"aws"}[0],
			},
			fields: fields{
				ocmClient:   ocmClientMockWithCloudAccounts,
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: false,
		},
		{
			name: "check passes when marketplace is not provided and account id is unambiguous",
			args: args{
				"test",
				"2345678901",
				nil,
			},
			fields: fields{
				ocmClient:   ocmClientMockWithCloudAccounts,
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: false,
		},
		{
			name: "check fails when billing account id is not found",
			args: args{
				"test",
				"xxx",
				nil,
			},
			fields: fields{
				ocmClient:   ocmClientMockWithCloudAccounts,
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "check fails when billing account id is found in the wrong marketplace",
			args: args{
				"test",
				"2345678901",
				&[]string{"aws"}[0],
			},
			fields: fields{
				ocmClient:   ocmClientMockWithCloudAccounts,
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "check fails when billing account id ambiguous",
			args: args{
				"test",
				"1234567890",
				nil,
			},
			fields: fields{
				ocmClient:   ocmClientMockWithCloudAccounts,
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "check fails when no billing accounts are returned",
			args: args{
				"test",
				"1234567890",
				&[]string{"aws"}[0],
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().
							BillingModel(string(v1.BillingModelStandard)).
							Product(string(ocm.RHOSAKProduct)).
							ResourceName(resourceName).
							Cost(1)

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}

						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			err := quotaService.ValidateBillingAccount(tt.args.orgId, types.STANDARD, tt.args.billingAccountId, tt.args.marketplace)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSCheckQuota(t *testing.T) {
	type fields struct {
		ocmClient   ocm.Client
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaID           string
		reserve           bool
		owner             string
		kafkaInstanceType types.KafkaInstanceType
		hasStandardQuota  bool
		hasDeveloperQuota bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "owner allowed to reserve quota",
			args: args{
				"",
				false,
				"testUser",
				types.STANDARD,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: false,
		},
		{
			name: "no quota error",
			args: args{
				"",
				false,
				"testUser",
				types.DEVELOPER,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						if cb.ProductID() == string(ocm.RHOSAKProduct) {
							ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
							return ca, nil
						}
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "owner not allowed to reserve quota",
			args: args{
				"",
				false,
				"testUser",
				types.STANDARD,
				false,
				false,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "failed to reserve quota",
			args: args{
				"12231",
				false,
				"testUser",
				types.STANDARD,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						return nil, fmt.Errorf("some errors")
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						if product != string(ocm.RHOSAKProduct) {
							return []*v1.QuotaCost{}, nil
						}
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			kafka := &dbapi.KafkaRequest{
				Meta: api.Meta{
					ID: tt.args.kafkaID,
				},
				Owner:          tt.args.owner,
				SizeId:         "x1",
				InstanceType:   string(tt.args.kafkaInstanceType),
				OrganisationId: "test",
			}

			sq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka.Owner, kafka.OrganisationId, types.STANDARD)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			eq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka.Owner, kafka.OrganisationId, types.DEVELOPER)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(sq).To(gomega.Equal(tt.args.hasStandardQuota))
			fmt.Printf("eq is %v\n", eq)
			g.Expect(eq).To(gomega.Equal(tt.args.hasDeveloperQuota))

			_, err = quotaService.ReserveQuota(kafka, tt.args.kafkaInstanceType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSReserveQuota(t *testing.T) {
	type fields struct {
		ocmClient   ocm.Client
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaID           string
		owner             string
		kafkaInstanceType types.KafkaInstanceType
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		want             string
		wantErr          bool
		wantBillingModel string
	}{
		{
			name: "reserve a quota & get subscription id",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when both standard and marketplace billing models are available standard is assigned as billing model",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantBillingModel: string(v1.BillingModelStandard),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when only marketplace billing model has available resources marketplace billing model is assigned",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb2, qcb1}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when a related resource has a supported billing model with cost of 0 that billing model is allowed",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKTrialProduct)).ResourceName(resourceName).Cost(0)
						qcb1, err := v1.NewQuotaCost().Allowed(0).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantBillingModel: string(v1.BillingModelMarketplace),
			want:             "1234",
			wantErr:          false,
		},
		{
			name: "when all matching quota_costs consumed resources are higher or equal than the allowed resources an error is returned",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb2, qcb1}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "when no quota_costs are available for the given product an error is returned",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						return []*v1.QuotaCost{}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "when the quota_costs returned do not contain a supported billing model an error is returned",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("1234")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodelone")).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string("unknownbillingmodeltwo")).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				"12231",
				"testUser",
				types.STANDARD,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
						rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &defaultKafkaConf,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			kafka := &dbapi.KafkaRequest{
				Meta: api.Meta{
					ID: tt.args.kafkaID,
				},
				Owner:        tt.args.owner,
				SizeId:       "x1",
				InstanceType: string(tt.args.kafkaInstanceType),
			}
			subId, err := quotaService.ReserveQuota(kafka, types.STANDARD)
			g.Expect(subId).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

			if tt.wantBillingModel != "" {
				ocmClientMock := tt.fields.ocmClient.(*ocm.ClientMock)
				clusterAuthorizationCalls := ocmClientMock.ClusterAuthorizationCalls()
				g.Expect(len(clusterAuthorizationCalls)).To(gomega.Equal(1))
				clusterAuthorizationResources := clusterAuthorizationCalls[0].Cb.Resources()
				g.Expect(clusterAuthorizationResources).To(gomega.HaveLen(1))
				clusterAuthorizationResource := clusterAuthorizationResources[0]
				g.Expect(clusterAuthorizationResource.BillingModel()).To(gomega.BeEquivalentTo(tt.wantBillingModel))
			}
		})
	}
}

func Test_Delete_Quota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		subscriptionId string
	}
	tests := []struct {
		// name is just a description of the test
		name   string
		fields fields
		args   args
		// want (there can be more than one) is the outputs that we expect, they can be compared after the test
		// function has been executed
		// wantErr is similar to want, but instead of testing the actual returned error, we're just testing than any
		// error has been returned
		wantErr bool
	}{
		{
			name: "delete a quota by id",
			args: args{
				subscriptionId: "1223",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSubscriptionFunc: func(id string) (int, error) {
						return 1, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to delete a quota by id",
			args: args{
				subscriptionId: "1223",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSubscriptionFunc: func(id string) (int, error) {
						return 0, errors.GeneralError("failed to delete subscription")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, &defaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			err := quotaService.DeleteQuota(tt.args.subscriptionId)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_amsQuotaService_CheckIfQuotaIsDefinedForInstanceType(t *testing.T) {
	type args struct {
		kafkaRequest      *dbapi.KafkaRequest
		kafkaInstanceType types.KafkaInstanceType
	}

	tests := []struct {
		name      string
		ocmClient ocm.Client
		args      args
		want      bool
		wantErr   bool
	}{
		{
			name: "returns false if no quota cost exists for the kafka's organization",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns false if the quota cost billing model is not among the supported ones",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocm.RHOSAKTrialProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "returns true if there is at least a 'standard' quota cost billing model",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodel2").Product(string(ocm.RHOSAKTrialProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1, rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true if there is at least a 'marketplace' quota cost billing model",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodel").Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(2).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}

					return []*v1.QuotaCost{qcb, qcb2}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false if there is no supported billing model with an 'allowed' value greater than 0",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelMarketplace)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					qcb, err := v1.NewQuotaCost().Allowed(0).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
					if err != nil {
						panic("unexpected error")
					}
					rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
					qcb2, err := v1.NewQuotaCost().Allowed(0).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
					if err != nil {
						panic("unexpected error")
					}
					return []*v1.QuotaCost{qcb, qcb2}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns an error if it fails retrieving the organization ID",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return "", fmt.Errorf("error getting org")
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, nil
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			wantErr: true,
		},
		{
			name: "returns an error if it fails retrieving quota costs",
			ocmClient: &ocm.ClientMock{
				GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
					return fmt.Sprintf("fake-org-id-%s", externalId), nil
				},
				GetQuotaCostsForProductFunc: func(organizationID, resourceName, product string) ([]*v1.QuotaCost, error) {
					return []*v1.QuotaCost{}, fmt.Errorf("error getting quota costs")
				},
			},
			args: args{
				kafkaRequest:      &dbapi.KafkaRequest{OrganisationId: "kafka-org-1"},
				kafkaInstanceType: types.STANDARD,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			quotaServiceFactory := NewDefaultQuotaServiceFactory(tt.ocmClient, nil, nil, &defaultKafkaConf)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)
			res, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(tt.args.kafkaRequest.Owner, tt.args.kafkaRequest.OrganisationId, tt.args.kafkaInstanceType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
