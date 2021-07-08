package quota

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
)

func Test_AllowListCheckQuota(t *testing.T) {
	type args struct {
		connectionFactory *db.ConnectionFactory
		KafkaConfig       *config.KafkaConfig
		AccessControlList *acl.AccessControlListConfig
	}

	tests := []struct {
		name    string
		arg     args
		want    *errors.ServiceError
		setupFn func()
	}{
		{
			name: "do not throw an error when instance limit control is disabled",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "2"}})
			},
			want: nil,
		},
		{
			name: "throw an error when the query db throws an error",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						ServiceAccounts: acl.AllowedAccounts{
							acl.AllowedAccount{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithQueryException()
			},
			want: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding allowed organisation limits",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						Organisations: acl.OrganisationList{
							acl.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AllowAll:            true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "4"}})
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organization 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     5,
			},
		},
		{
			name: "throw an error when user cannot create any more instances after exceeding allowed limits",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						ServiceAccounts: acl.AllowedAccounts{
							acl.AllowedAccount{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "4"}})
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
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						ServiceAccounts: acl.AllowedAccounts{
							acl.AllowedAccount{
								Username: "username",
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "1"}})
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
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "1"}})
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
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						Organisations: acl.OrganisationList{
							acl.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AllowAll:            false,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "1"}})
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
			},
		},
		{
			name: "does not return an error if user is within limits",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						Organisations: acl.OrganisationList{
							acl.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AllowAll:            true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("count").WithReply([]map[string]interface{}{{"count": "1"}})
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			factory := NewDefaultQuotaServiceFactory(nil, tt.arg.connectionFactory, tt.arg.KafkaConfig, tt.arg.AccessControlList)
			quotaService, _ := factory.GetQuotaService(api.AllowListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			_, err := quotaService.CheckQuota(kafka, types.EVAL)
			gomega.Expect(tt.want).To(gomega.Equal(err))
		})
	}
}
