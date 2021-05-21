package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"reflect"
	"testing"
)

func Test_Reserve_Quota(t *testing.T) {

	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		productID    string
		clusterID    string
		kafkaID      string
		reserve      bool
		owner        string
		availability string
	}
	tests := []struct {
		// name is just a description of the test
		name   string
		fields fields
		args   args
		// want (there can be more than one) is the outputs that we expect, they can be compared after the test
		// function has been executed
		want bool
		// wantErr is similar to want, but instead of testing the actual returned error, we're just testing than any
		// error has been returned
		wantErr bool
	}{
		{
			name: "Is owner allowed to reserve a quota",
			args: args{
				"RHOSAKTrial",
				"",
				"",
				false,
				"testUser",
				"single",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Build()
						return ca, nil
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "reserve a quota & get subscription id",
			args: args{
				"RHOSAKTrial",
				"123",
				"12231",
				true,
				"testUser",
				"single",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						sub := v1.SubscriptionBuilder{}
						sub.ID("12334")
						sub.Status("Active")
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(true).Subscription(&sub).Build()
						return ca, nil
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "failed to reserve a quota",
			args: args{
				"RHOSAKTrial",
				"123",
				"12231",
				false,
				"testUser",
				"single",
			},
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ClusterAuthorizationFunc: func(cb *v1.ClusterAuthorizationRequest) (*v1.ClusterAuthorizationResponse, error) {
						ca, _ := v1.NewClusterAuthorizationResponse().Allowed(false).Build()
						return ca, nil
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			QuotaService := quotaService{
				tt.fields.ocmClient,
			}
			got, _, err := QuotaService.ReserveQuota(tt.args.productID, tt.args.clusterID, tt.args.kafkaID, tt.args.owner, tt.args.reserve, tt.args.availability)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReserveQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReserveQuota() got = %+v, want %+v", got, tt.want)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			QuotaService := quotaService{
				tt.fields.ocmClient,
			}
			err := QuotaService.DeleteQuota(tt.args.subscriptionId)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
