package quota

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota/internal/mocks/ams"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota/internal/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
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

	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

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
		wantErrMsg               string
		wantMatch                bool
		wantKafkaBillingModel    string
		wantBillingModel         string
		wantInferredBillingModel string
	}{
		{
			name: "standard/eval instance is requested",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                     "test",
					SizeId:                   "x1",
					DesiredKafkaBillingModel: "eval",
					InstanceType:             "standard",
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
						rrbq := []*v1.RelatedResourceBuilder{
							v1.NewRelatedResource().
								BillingModel(string(v1.BillingModelMarketplace)).
								Product(string(ocm.RHOSAKProduct)).
								ResourceName(resourceName).
								Cost(1),
							v1.NewRelatedResource().
								BillingModel(string(v1.BillingModelStandard)).
								Product(string(ocm.RHOSAKProduct)).
								ResourceName(resourceName).
								Cost(1),
						}

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(rrbq...).
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    "eval",
			wantBillingModel:         string(v1.BillingModelStandard),
			wantInferredBillingModel: "eval",
		},
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    string(v1.BillingModelMarketplace),
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    string(v1.BillingModelMarketplace),
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    string(v1.BillingModelStandard),
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantKafkaBillingModel:    "",
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantBillingModel:         "",
			wantKafkaBillingModel:    "",
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  true,
			wantMatch:                false,
			wantKafkaBillingModel:    "",
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    string(v1.BillingModelStandard),
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  false,
			wantMatch:                true,
			wantKafkaBillingModel:    string(v1.BillingModelMarketplace),
			wantBillingModel:         string(v1.BillingModelMarketplaceAWS),
			wantInferredBillingModel: string(v1.BillingModelMarketplace),
		},
		{
			name: "detects non-matching requested billing model (standard)",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                     "test",
					SizeId:                   "x1",
					BillingCloudAccountId:    "1234567890",
					Marketplace:              "aws",
					InstanceType:             "standard",
					DesiredKafkaBillingModel: "standard",
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  true,
			wantErrMsg:               "KAFKAS-MGMT-120: unable to detect billing model\n caused by: KAFKAS-MGMT-120: Insufficient quota: marketplace value 'aws' is not compatible with billing model 'standard'",
			wantMatch:                false,
			wantKafkaBillingModel:    "",
			wantBillingModel:         "",
			wantInferredBillingModel: "",
		},
		{
			name: "detects non-matching requested billing model (marketplace)",
			args: args{
				request: dbapi.KafkaRequest{
					Name:                     "test",
					SizeId:                   "x1",
					InstanceType:             "standard",
					DesiredKafkaBillingModel: "marketplace",
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                  true,
			wantErrMsg:               "Insufficient quota: no quota available for any of the matched kafka billing models [marketplace]",
			wantMatch:                false,
			wantKafkaBillingModel:    "",
			wantBillingModel:         "",
			wantInferredBillingModel: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)

			kafkaBillingModel, billingModel, err := quotaService.(*amsQuotaService).getBillingModel(&tt.args.request)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "Unexpected error value %v", err)
			if tt.wantErrMsg != "" {
				g.Expect(err.Error()).To(gomega.ContainSubstring(tt.wantErrMsg))
			}
			g.Expect(billingModel).To(gomega.Equal(tt.wantBillingModel))
			g.Expect(kafkaBillingModel.ID).To(gomega.Equal(tt.wantKafkaBillingModel))

			match, bm := quotaService.(*amsQuotaService).billingModelMatches(billingModel, tt.args.request.DesiredKafkaBillingModel, kafkaBillingModel)
			g.Expect(match).To(gomega.Equal(tt.wantMatch))
			g.Expect(bm).To(gomega.Equal(tt.wantInferredBillingModel))
		})
	}

}

func Test_AMSValidateBillingAccount(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

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
				kafkaConfig: &amsDefaultKafkaConf,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
			// TODO: add a test value for billing model
			err := quotaService.ValidateBillingAccount(tt.args.orgId, types.STANDARD, "", tt.args.billingAccountId, tt.args.marketplace)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSCheckQuota(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

	type fields struct {
		ocmClient   *ocm.ClientMock
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaID             string
		reserve             bool
		owner               string
		kafkaInstanceType   types.KafkaInstanceType
		hasStandardQuota    bool
		hasDeveloperQuota   bool
		desiredBillingModel string
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
				kafkaID:             "",
				reserve:             false,
				owner:               "testUser",
				kafkaInstanceType:   types.STANDARD,
				desiredBillingModel: "standard",
				hasStandardQuota:    true,
				hasDeveloperQuota:   false,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: false,
		},
		{
			name: "no quota error",
			args: args{
				kafkaID:             "",
				reserve:             false,
				owner:               "testUser",
				kafkaInstanceType:   types.DEVELOPER,
				desiredBillingModel: "trial",
				hasStandardQuota:    true,
				hasDeveloperQuota:   false,
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
						var relatedResourceBuilder *v1.RelatedResourceBuilder

						switch product {
						case string(ocm.RHOSAKProduct):
							relatedResourceBuilder = v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						default:
							return []*v1.QuotaCost{}, nil
						}

						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).OrganizationID(organizationID).RelatedResources(relatedResourceBuilder).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "owner not allowed to reserve quota",
			args: args{
				kafkaID:             "",
				reserve:             false,
				owner:               "testUser",
				kafkaInstanceType:   types.STANDARD,
				desiredBillingModel: "standard",
				hasStandardQuota:    false,
				hasDeveloperQuota:   false,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "failed to reserve quota",
			args: args{
				kafkaID:             "12231",
				reserve:             false,
				owner:               "testUser",
				kafkaInstanceType:   types.STANDARD,
				desiredBillingModel: "standard",
				hasStandardQuota:    true,
				hasDeveloperQuota:   false,
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
				kafkaConfig: &amsDefaultKafkaConf,
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
				Owner:                    tt.args.owner,
				SizeId:                   "x1",
				InstanceType:             string(tt.args.kafkaInstanceType),
				DesiredKafkaBillingModel: tt.args.desiredBillingModel,
				OrganisationId:           "test",
			}

			bm, err1 := tt.fields.kafkaConfig.GetBillingModelByID(types.STANDARD.String(), "standard")
			g.Expect(err1).ToNot(gomega.HaveOccurred())
			sq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka.Owner, kafka.OrganisationId, types.STANDARD, bm)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()).To(gomega.HaveLen(1))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()[0].ResourceName).To(gomega.Equal(ocm.RHOSAKResourceName))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()[0].Product).To(gomega.BeEquivalentTo(ocm.RHOSAKProduct))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()[0].Product).To(gomega.Equal(bm.AMSProduct))
			g.Expect(sq).To(gomega.Equal(tt.args.hasStandardQuota))
			bm, err1 = tt.fields.kafkaConfig.GetBillingModelByID(types.DEVELOPER.String(), "standard")
			g.Expect(err1).ToNot(gomega.HaveOccurred())
			eq, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka.Owner, kafka.OrganisationId, types.DEVELOPER, bm)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			fmt.Printf("eq is %v\n", eq)
			g.Expect(eq).To(gomega.Equal(tt.args.hasDeveloperQuota))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()).To(gomega.HaveLen(2))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()[1].ResourceName).To(gomega.Equal(ocm.RHOSAKResourceName))
			g.Expect(tt.fields.ocmClient.GetQuotaCostsForProductCalls()[1].Product).To(gomega.BeEquivalentTo(ocm.RHOSAKTrialProduct))
			_, err = quotaService.ReserveQuota(kafka)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSReserveQuotaIfNotAlreadyReserved(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}
	type fields struct {
		ocmClient   *ocm.ClientMock
		kafkaConfig config.KafkaConfig
	}
	type args struct {
		kafkaID                           string
		kafkaInstanceType                 types.KafkaInstanceType
		kafkaRequestMarketplace           string
		kafkaRequestDesiredBillingModel   string
		kafkaRequestBillingCloudAccountID string
		kafkaRequestCloudProvider         string

		wantsReserveCalled bool
		wantsProductID     string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Reserve quota for new cluster",
			fields: fields{
				kafkaConfig: amsDefaultKafkaConf,
				ocmClient: &ocm.ClientMock{
					FindSubscriptionsFunc: func(query string) ([]*v1.Subscription, error) {
						return ams.FindSubscriptions(query), nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID(cb.ClusterID() + "subscription")
						sub.Status("Active")
						sub.ClusterBillingModel("standard")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
				},
			},
			args: args{
				kafkaID:                         "1234",
				kafkaInstanceType:               types.STANDARD,
				kafkaRequestDesiredBillingModel: "eval",

				wantsReserveCalled: true,
				wantsProductID:     string(ocm.RHOSAKEvalProduct),
			},
		},
		{
			name: "Reserve quota for existing cluster and existing subscription",
			fields: fields{
				kafkaConfig: amsDefaultKafkaConf,
				ocmClient: &ocm.ClientMock{
					FindSubscriptionsFunc: func(query string) ([]*v1.Subscription, error) {
						return ams.FindSubscriptions(query), nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID(cb.ClusterID() + "subscription")
						sub.Status("Active")
						sub.ClusterBillingModel("standard")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
				},
			},
			args: args{
				kafkaID:                         "cluster-1",
				kafkaInstanceType:               types.STANDARD,
				kafkaRequestDesiredBillingModel: "standard",

				wantsReserveCalled: false,
				wantsProductID:     string(ocm.RHOSAKProduct),
			},
		},
		{
			// In this test we try to reserve a standard for a cluster that has `eval` assigned
			name: "Reserve standard quota for existing cluster with `eval` subscription",
			fields: fields{
				kafkaConfig: amsDefaultKafkaConf,
				ocmClient: &ocm.ClientMock{
					FindSubscriptionsFunc: func(query string) ([]*v1.Subscription, error) {
						return ams.FindSubscriptions(query), nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID(cb.ClusterID() + "subscription")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
				},
			},
			args: args{
				kafkaID:                         "cluster-7",
				kafkaInstanceType:               types.STANDARD,
				kafkaRequestDesiredBillingModel: "standard",

				wantsReserveCalled: true,
				wantsProductID:     string(ocm.RHOSAKProduct),
			},
		},
		{
			// In this test we try to reserve an `eval` for a cluster that has `eval` assigned
			name: "Reserve `eval` quota for existing cluster with `eval` subscription",
			fields: fields{
				kafkaConfig: amsDefaultKafkaConf,
				ocmClient: &ocm.ClientMock{
					FindSubscriptionsFunc: func(query string) ([]*v1.Subscription, error) {
						return ams.FindSubscriptions(query), nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID(cb.ClusterID() + "subscription")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
				},
			},
			args: args{
				kafkaID:                         "cluster-7",
				kafkaInstanceType:               types.STANDARD,
				kafkaRequestDesiredBillingModel: "eval",

				wantsReserveCalled: false,
				wantsProductID:     string(ocm.RHOSAKProduct),
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		kafka := &dbapi.KafkaRequest{
			Meta: api.Meta{
				ID: tt.args.kafkaID,
			},
			ClusterID:                tt.args.kafkaID,
			CloudProvider:            tt.args.kafkaRequestCloudProvider,
			Owner:                    "owner",
			SizeId:                   "x1",
			InstanceType:             string(tt.args.kafkaInstanceType),
			Marketplace:              tt.args.kafkaRequestMarketplace,
			BillingCloudAccountId:    tt.args.kafkaRequestBillingCloudAccountID,
			DesiredKafkaBillingModel: tt.args.kafkaRequestDesiredBillingModel,
		}

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, &tt.fields.kafkaConfig)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)

			_, err := quotaService.ReserveQuotaIfNotAlreadyReserved(kafka)
			g.Expect(err).ToNot(gomega.HaveOccurred(), "Expecting error not to happen:", err)

			if tt.args.wantsReserveCalled {
				g.Expect(tt.fields.ocmClient.ClusterAuthorizationCalls()).To(gomega.HaveLen(1))
				g.Expect(tt.fields.ocmClient.ClusterAuthorizationCalls()[0].Cb.ProductID()).To(gomega.Equal(tt.args.wantsProductID))
			} else {
				g.Expect(tt.fields.ocmClient.ClusterAuthorizationCalls()).To(gomega.HaveLen(0), "Should not reserve a new quota")
			}
		})
	}

}
func Test_AMSReserveQuota(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

	type fields struct {
		ocmClient   ocm.Client
		kafkaConfig *config.KafkaConfig
	}
	type args struct {
		kafkaID                           string
		owner                             string
		kafkaInstanceType                 types.KafkaInstanceType
		kafkaRequestMarketplace           string
		kafkaRequestDesiredBillingModel   string
		kafkaRequestBillingCloudAccountID string
		kafkaRequestCloudProvider         string
	}
	tests := []struct {
		name                         string
		fields                       fields
		args                         args
		wantSubscriptionID           string
		wantErr                      bool
		wantAMSBillingModel          string
		wantDesiredKafkaBillingModel string
		wantActualKafkaBillingModel  string
	}{
		{
			name: "try to reserve for developer",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.DEVELOPER,
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
						if product == string(ocm.RHOSAKTrialProduct) {
							rrbq1 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKTrialProduct)).ResourceName(resourceName).Cost(0)
							qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
								OrganizationID(organizationID).RelatedResources(rrbq1).
								CloudAccounts(v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("12345")).
								Build()
							if err != nil {
								panic("unexpected error")
							}
							return []*v1.QuotaCost{qcb}, nil
						}
						return nil, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantAMSBillingModel:          string(v1.BillingModelStandard),
			wantSubscriptionID:           "1234",
			wantDesiredKafkaBillingModel: "standard",
			wantActualKafkaBillingModel:  "standard",
			wantErr:                      false,
		},
		{
			name: "reserve a quota & get subscription id",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantAMSBillingModel:          string(v1.BillingModelMarketplace),
			wantSubscriptionID:           "1234",
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantErr:                      false,
		},
		{
			name: "when both standard and marketplace billing models are available standard is assigned as billing model",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantAMSBillingModel:          string(v1.BillingModelStandard),
			wantDesiredKafkaBillingModel: "standard",
			wantActualKafkaBillingModel:  "standard",
			wantSubscriptionID:           "1234",
			wantErr:                      false,
		},
		{
			name: "when only marketplace billing model has available resources marketplace billing model is assigned",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantAMSBillingModel:          string(v1.BillingModelMarketplace),
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantSubscriptionID:           "1234",
			wantErr:                      false,
		},
		{
			name: "when a related resource has a supported billing model with cost of 0 that billing model is allowed",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantAMSBillingModel:          string(v1.BillingModelMarketplace),
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantSubscriptionID:           "1234",
			wantErr:                      false,
		},
		{
			name: "when all matching quota_costs consumed resources are higher or equal than the allowed resources an error is returned",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "when no quota_costs are available for the given product an error is returned",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "when the quota_costs returned do not contain a supported billing model an error is returned",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
						rrbq1 := v1.NewRelatedResource().BillingModel("unknownbillingmodelone").Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq1).Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel("unknownbillingmodeltwo").Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(1).OrganizationID(organizationID).RelatedResources(rrbq2).Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr: true,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				kafkaID:           "12231",
				owner:             "testUser",
				kafkaInstanceType: types.STANDARD,
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
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                      true,
			wantDesiredKafkaBillingModel: "marketplace",
			wantAMSBillingModel:          "marketplace",
			wantActualKafkaBillingModel:  "",
		},
		{
			name: "reserving a Kafka located in AWS with marketplace aws AMS billing model is accepted",
			args: args{
				kafkaID:                           "12231",
				owner:                             "testUser",
				kafkaInstanceType:                 types.STANDARD,
				kafkaRequestMarketplace:           "aws",
				kafkaRequestBillingCloudAccountID: "12345",
				kafkaRequestDesiredBillingModel:   "marketplace",
				kafkaRequestCloudProvider:         cloudproviders.AWS.String(),
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
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("12345")).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantSubscriptionID:           "1234",
			wantAMSBillingModel:          string(v1.BillingModelMarketplaceAWS),
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantErr:                      false,
		},
		{
			name: "reserving a Kafka located in GCP with Red Hat marketplace AMS billing model is accepted",
			args: args{
				kafkaID:                           "12231",
				owner:                             "testUser",
				kafkaInstanceType:                 types.STANDARD,
				kafkaRequestMarketplace:           "rhm",
				kafkaRequestBillingCloudAccountID: "12345",
				kafkaRequestDesiredBillingModel:   "marketplace",
				kafkaRequestCloudProvider:         cloudproviders.GCP.String(),
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
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(v1.NewCloudAccount().CloudProviderID("rhm").CloudAccountID("12345")).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantSubscriptionID:           "1234",
			wantAMSBillingModel:          string(v1.BillingModelMarketplace),
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantErr:                      false,
		},
		{
			name: "reserving a Kafka located in GCP with no explicit marketplace reserves 'marketplace' quota when the related resource has marketplace quota only",
			args: args{
				kafkaID:                           "12231",
				owner:                             "testUser",
				kafkaInstanceType:                 types.STANDARD,
				kafkaRequestBillingCloudAccountID: "12345",
				kafkaRequestDesiredBillingModel:   "marketplace",
				kafkaRequestCloudProvider:         cloudproviders.GCP.String(),
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
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(v1.NewCloudAccount().CloudProviderID("rhm").CloudAccountID("12345")).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantSubscriptionID:           "1234",
			wantAMSBillingModel:          string(v1.BillingModelMarketplace),
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "marketplace",
			wantErr:                      false,
		},
		{
			name: "reserving a Kafka located in GCP with no explicit marketplace nor billing model reserves standard if there is standard related resource quota",
			args: args{
				kafkaID:                   "12231",
				owner:                     "testUser",
				kafkaInstanceType:         types.STANDARD,
				kafkaRequestCloudProvider: cloudproviders.GCP.String(),
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
						qcb1, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(v1.NewCloudAccount().CloudProviderID("rhm").CloudAccountID("12345")).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						rrbq2 := v1.NewRelatedResource().BillingModel(string(v1.BillingModelStandard)).Product(string(ocm.RHOSAKProduct)).ResourceName(resourceName).Cost(1)
						qcb2, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq2).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb1, qcb2}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantSubscriptionID:           "1234",
			wantAMSBillingModel:          string(v1.BillingModelStandard),
			wantDesiredKafkaBillingModel: "standard",
			wantActualKafkaBillingModel:  "standard",
			wantErr:                      false,
		},
		{
			name: "when reserving a Kafka located in GCP with a non Red Hat marketplace AMS billing model specified an error is returned",
			args: args{
				kafkaID:                           "12231",
				owner:                             "testUser",
				kafkaInstanceType:                 types.STANDARD,
				kafkaRequestMarketplace:           "aws",
				kafkaRequestBillingCloudAccountID: "12345",
				kafkaRequestDesiredBillingModel:   "marketplace",
				kafkaRequestCloudProvider:         cloudproviders.GCP.String(),
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
						qcb, err := v1.NewQuotaCost().Allowed(1).Consumed(0).
							OrganizationID(organizationID).RelatedResources(rrbq1).
							CloudAccounts(v1.NewCloudAccount().CloudProviderID("aws").CloudAccountID("12345")).
							Build()
						if err != nil {
							panic("unexpected error")
						}
						return []*v1.QuotaCost{qcb}, nil
					},
				},
				kafkaConfig: &amsDefaultKafkaConf,
			},
			wantErr:                      true,
			wantDesiredKafkaBillingModel: "marketplace",
			wantActualKafkaBillingModel:  "",
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
				CloudProvider:            tt.args.kafkaRequestCloudProvider,
				Owner:                    tt.args.owner,
				SizeId:                   "x1",
				InstanceType:             string(tt.args.kafkaInstanceType),
				Marketplace:              tt.args.kafkaRequestMarketplace,
				BillingCloudAccountId:    tt.args.kafkaRequestBillingCloudAccountID,
				DesiredKafkaBillingModel: tt.args.kafkaRequestDesiredBillingModel,
			}
			subId, err := quotaService.ReserveQuota(kafka)

			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "Unexpected error value '%v'", err)
			g.Expect(kafka.DesiredKafkaBillingModel).To(gomega.Equal(tt.wantDesiredKafkaBillingModel))
			g.Expect(kafka.ActualKafkaBillingModel).To(gomega.Equal(tt.wantActualKafkaBillingModel))

			g.Expect(subId).To(gomega.Equal(tt.wantSubscriptionID))
			if tt.wantAMSBillingModel != "" {
				ocmClientMock := tt.fields.ocmClient.(*ocm.ClientMock)
				clusterAuthorizationCalls := ocmClientMock.ClusterAuthorizationCalls()
				g.Expect(len(clusterAuthorizationCalls)).To(gomega.Equal(1))
				clusterAuthorizationResources := clusterAuthorizationCalls[0].Cb.Resources()
				g.Expect(clusterAuthorizationResources).To(gomega.HaveLen(1))
				clusterAuthorizationResource := clusterAuthorizationResources[0]
				g.Expect(clusterAuthorizationResource.BillingModel()).To(gomega.BeEquivalentTo(tt.wantAMSBillingModel))
			}
		})
	}
}

func Test_Delete_Quota(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

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
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil, &amsDefaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			err := quotaService.DeleteQuota(tt.args.subscriptionId)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_amsQuotaService_CheckIfQuotaIsDefinedForInstanceType(t *testing.T) {
	var amsDefaultKafkaConf = config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

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
			quotaServiceFactory := NewDefaultQuotaServiceFactory(tt.ocmClient, nil, nil, &amsDefaultKafkaConf)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)

			// FIXME: fix when implementing support for KAFKA BILLING MODELS
			res, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(tt.args.kafkaRequest.Owner, tt.args.kafkaRequest.OrganisationId, tt.args.kafkaInstanceType, config.KafkaBillingModel{})
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_amsQuotaService_newBaseQuotaReservedResourceBuilder(t *testing.T) {
	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}

	tests := []struct {
		name        string
		args        args
		wantFactory func() v1.ReservedResourceBuilder
	}{
		{
			name: "When the kafka request is in AWS and MultiAZ the correct reserved resource builder is returned",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					MultiAZ:       true,
					CloudProvider: cloudproviders.AWS.String(),
				},
			},
			wantFactory: func() v1.ReservedResourceBuilder {
				rrbuilder := v1.NewReservedResource()
				rrbuilder.Count(1)
				rrbuilder.ResourceType(amsReservedResourceResourceTypeClusterAWS)
				rrbuilder.ResourceName(ocm.RHOSAKResourceName)
				return *rrbuilder
			},
		},
		{
			name: "When the kafka request is single AZ the correct reserved resource builder is returned",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					MultiAZ:       false,
					CloudProvider: cloudproviders.AWS.String(),
				},
			},
			wantFactory: func() v1.ReservedResourceBuilder {
				rrbuilder := v1.NewReservedResource()
				rrbuilder.Count(1)
				rrbuilder.ResourceType(amsReservedResourceResourceTypeClusterAWS)
				rrbuilder.ResourceName(ocm.RHOSAKResourceName)
				return *rrbuilder
			},
		},
		{
			name: "When the kafka request is in GCP the correct reserved resource builder is returned",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					MultiAZ:       true,
					CloudProvider: cloudproviders.GCP.String(),
				},
			},
			wantFactory: func() v1.ReservedResourceBuilder {
				rrbuilder := v1.NewReservedResource()
				rrbuilder.Count(1)
				rrbuilder.ResourceType(amsReservedResourceResourceTypeClusterGCP)
				rrbuilder.ResourceName(ocm.RHOSAKResourceName)
				return *rrbuilder
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			amsQuotaService := amsQuotaService{}
			want := tt.wantFactory()
			kafkaBillingModel := config.KafkaBillingModel{
				ID:          "standard",
				AMSResource: ocm.RHOSAKResourceName,
			}
			res := amsQuotaService.newBaseQuotaReservedResourceBuilder(tt.args.kafkaRequest, kafkaBillingModel)
			g.Expect(res).To(gomega.Equal(want))
		})
	}
}

func Test_amsQuotaService_IsQuotaEntitlementActive(t *testing.T) {
	type fields struct {
		amsClient   ocm.Client
		kafkaConfig config.KafkaConfig
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}

	defaultKafkaConfig := config.KafkaConfig{
		Quota:                  config.NewKafkaQuotaConfig(),
		SupportedInstanceTypes: test.NewAMSTestKafkaSupportedInstanceTypesConfig(),
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		want       bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns an error when it fails to get the kafka billing model",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType: "unsupported-instance-type",
				},
			},
			want:       false,
			wantErr:    true,
			wantErrMsg: "unable to find kafka instance type for 'unsupported-instance-type'",
		},
		{
			name: "returns an error when it fails to get quota costs from ams",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						return nil, fmt.Errorf("failed to get quota cost")
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:       false,
			wantErr:    true,
			wantErrMsg: "failed to get quota cost",
		},
		{
			name: "returns false if no quota cost is found",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns an error if more than one quota cost is found",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						qc1, _ := v1.NewQuotaCost().Build()
						qc2, _ := v1.NewQuotaCost().Build()

						return []*v1.QuotaCost{qc1, qc2}, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:       false,
			wantErr:    true,
			wantErrMsg: `more than 1 quota cost was returned for organisation "org-id" with the following filter: {ResourceName: "rhosak", Product: "RHOSAK", BillingModel: "standard"}`,
		},
		{
			name: "returns false if quota is no longer entitled",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						qc, _ := v1.NewQuotaCost().Consumed(1).Allowed(0).Build()

						return []*v1.QuotaCost{qc}, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "returns true if quota entitlement is active",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						qc, _ := v1.NewQuotaCost().Consumed(1).Allowed(1).Build()

						return []*v1.QuotaCost{qc}, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns true if organisation has exceeded their quota limits but entitlement is still active",
			fields: fields{
				kafkaConfig: defaultKafkaConfig,
				amsClient: &ocm.ClientMock{
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return "org-id", nil
					},
					GetSubscriptionByIDFunc: ams.GetSubscriptionByID,
					GetQuotaCostsFunc: func(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...ocm.QuotaCostRelatedResourceFilter) ([]*v1.QuotaCost, error) {
						qc, _ := v1.NewQuotaCost().Consumed(3).Allowed(1).Build()

						return []*v1.QuotaCost{qc}, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					InstanceType:            "standard",
					ActualKafkaBillingModel: "standard",
					SubscriptionId:          "subscription-2", // rhosak, standard
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			quotaServiceFactory := NewDefaultQuotaServiceFactory(tt.fields.amsClient, nil, nil, &tt.fields.kafkaConfig)
			quotaService, _ := quotaServiceFactory.GetQuotaService(api.AMSQuotaType)

			got, err := quotaService.IsQuotaEntitlementActive(tt.args.kafka)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if tt.wantErr {
				g.Expect(err.Error()).To(gomega.Equal(tt.wantErrMsg))
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
