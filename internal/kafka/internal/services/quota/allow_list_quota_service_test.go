package quota

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
)

func Test_AllowListCheckQuota(t *testing.T) {
	type args struct {
		connectionFactory *db.ConnectionFactory
		AccessControlList *acl.AccessControlListConfig
	}

	tests := []struct {
		name    string
		arg     args
		wantErr *errors.ServiceError
		want    bool
		setupFn func()
	}{
		{
			name: "do not throw an error when instance limit control is disabled",
			arg: args{
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			wantErr: nil,
			want:    true,
		},
		{
			name: "throw an error when the query db throws an error",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						AllowAnyRegisteredUsers: true,
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
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when user in an organiation cannot create any more instances after exceeding allowed organisation limits plus their one Eval instance",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						AllowAnyRegisteredUsers: true,
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
				mocket.Catcher.NewMock().
					WithQuery(`SELECT instance_type, count(1) as Count FROM "kafka_requests" WHERE (organisation_id = $1)`).
					WithArgs("org-id").
					WithReply([]map[string]interface{}{{"instance_type": "standard", "count": "4"}, {"instance_type": "eval", "count": "1"}})
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organization 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     5,
			},
		},
		{
			name: "do not throw an error when user in the allow list can still create eval instace",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList: acl.AllowListConfiguration{
						AllowAnyRegisteredUsers: true,
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
				mocket.Catcher.NewMock().
					WithQuery(`SELECT instance_type, count(1) as Count FROM "kafka_requests" WHERE owner = $1`).
					WithArgs("username").
					WithReply([]map[string]interface{}{{"instance_type": "standard", "count": "5"}, {"instance_type": "eval", "count": "0"}})
			},
			wantErr: nil,
			want:    false,
		},
		{
			name: "do not throw an error when user who's not in the allow list can eval instances",
			arg: args{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT instance_type, count(1) as Count FROM "kafka_requests" WHERE owner = $1`).
					WithArgs("username").
					WithReply([]map[string]interface{}{{"instance_type": "eval", "count": "0"}})
			},
			want:    false,
			wantErr: nil,
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
								MaxAllowedInstances: 2,
								AllowAll:            false,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT instance_type, count(1) as Count FROM "kafka_requests" WHERE owner = $1`).
					WithArgs("username").
					WithReply([]map[string]interface{}{{"instance_type": "standard", "count": "1"}, {"instance_type": "eval", "count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
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
				mocket.Catcher.NewMock().
					WithQuery(`SELECT instance_type, count(1) as Count FROM "kafka_requests" WHERE (organisation_id = $1)`).
					WithArgs("org-id").
					WithReply([]map[string]interface{}{{"instance_type": "standard", "count": "1"}, {"instance_type": "eval", "count": "0"}})
			},
			wantErr: nil,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			factory := NewDefaultQuotaServiceFactory(nil, tt.arg.connectionFactory, tt.arg.AccessControlList)
			quotaService, _ := factory.GetQuotaService(api.AllowListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			allowed, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka, types.STANDARD)
			gomega.Expect(tt.wantErr).To(gomega.Equal(err))
			gomega.Expect(tt.want).To(gomega.Equal(allowed))
		})
	}
}
