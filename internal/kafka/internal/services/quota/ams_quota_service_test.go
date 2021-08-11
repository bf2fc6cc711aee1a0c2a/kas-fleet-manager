package quota

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func Test_AMSCheckQuota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		kafkaID           string
		reserve           bool
		owner             string
		kafkaInstanceType types.KafkaInstanceType
		hasStandardQuota  bool
		hasEvalQuota      bool
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
					HasAssignedQuotaFunc: func(organizationId string, filter string) (bool, error) {
						return filter == "quota_id='cluster|rhinfra|rhosak|marketplace'", nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no quota error",
			args: args{
				"",
				false,
				"testUser",
				types.EVAL,
				true,
				false,
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						if cb.ProductID() == "RHOSAK" {
							ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
							return ca, nil
						}
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
					GetOrganisationIdFromExternalIdFunc: func(externalId string) (string, error) {
						return fmt.Sprintf("fake-org-id-%s", externalId), nil
					},
					HasAssignedQuotaFunc: func(organizationId string, filter string) (bool, error) {
						return filter == "quota_id='cluster|rhinfra|rhosak|marketplace'", nil
					},
				},
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
					HasAssignedQuotaFunc: func(organizationId string, filter string) (bool, error) {
						return false, nil
					},
				},
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
					HasAssignedQuotaFunc: func(organizationId string, filter string) (bool, error) {
						return filter == "quota_id='cluster|rhinfra|rhosak|marketplace'", nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			kafka := &dbapi.KafkaRequest{
				Meta: api.Meta{
					ID: tt.args.kafkaID,
				},
				Owner: tt.args.owner,
			}
			sq, err := quotaService.CheckQuota(kafka, types.STANDARD)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			eq, err := quotaService.CheckQuota(kafka, types.EVAL)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sq).To(gomega.Equal(tt.args.hasStandardQuota))
			gomega.Expect(eq).To(gomega.Equal(tt.args.hasEvalQuota))

			_, err = quotaService.ReserveQuota(kafka, tt.args.kafkaInstanceType)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_AMSReserveQuota(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		kafkaID string
		reserve bool
		owner   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "reserve a quota & get subscription id",
			args: args{
				"12231",
				true,
				"testUser",
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
				},
			},
			want:    "1234",
			wantErr: false,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				"12231",
				false,
				"testUser",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			kafka := &dbapi.KafkaRequest{
				Meta: api.Meta{
					ID: tt.args.kafkaID,
				},
				Owner: tt.args.owner,
			}
			subId, err := quotaService.ReserveQuota(kafka, types.STANDARD)
			gomega.Expect(subId).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewDefaultQuotaServiceFactory(tt.fields.ocmClient, nil, nil)
			quotaService, _ := factory.GetQuotaService(api.AMSQuotaType)
			err := quotaService.DeleteQuota(tt.args.subscriptionId)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
