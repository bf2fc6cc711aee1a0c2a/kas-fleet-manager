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
	type fields struct {
		connectionFactory *db.ConnectionFactory
		AccessControlList *acl.AccessControlListConfig
	}

	type args struct {
		instanceType types.KafkaInstanceType
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "do not throw an error when instance limit control is disabled when checking eval instances",
			fields: fields{
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			want: true,
		},
		{
			name: "do not throw an error when instance limit control is disabled when checking standard instances",
			fields: fields{
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the allow list and instance type is eval",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList:                  acl.AllowListConfiguration{},
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the allow list and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
					AllowList:                  acl.AllowListConfiguration{},
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: false,
		},
		{
			name: "return true when user is part of the allow list as a service account and instance type is standard",
			fields: fields{
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
			args: args{
				instanceType: types.STANDARD,
			},
			want: true,
		},
		{
			name: "return true when user is part of the allow list under an organisation and instance type is standard",
			fields: fields{
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
			args: args{
				instanceType: types.STANDARD,
			},
			want: true,
		},
		{
			name: "return false when user is part of the allow list under an organisation and instance type is eval",
			fields: fields{
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
			args: args{
				instanceType: types.EVAL,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)

			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.AccessControlList)
			quotaService, _ := factory.GetQuotaService(api.AllowListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			allowed, _ := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka, tt.args.instanceType)
			gomega.Expect(tt.want).To(gomega.Equal(allowed))
		})
	}
}

func Test_AllowListReserveQuota(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		AccessControlList *acl.AccessControlListConfig
	}

	type args struct {
		instanceType types.KafkaInstanceType
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr *errors.ServiceError
		setupFn func()
	}{
		{
			name: "do not return an error when instance limit control is disabled ",
			fields: fields{
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.EVAL,
			},
			wantErr: nil,
		},
		{
			name: "return an error when the query db throws an error",
			fields: fields{
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
			args: args{
				instanceType: types.EVAL,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.GeneralError("count failed from database"),
		},
		{
			name: "return an error when user in an organiation cannot create any more instances after exceeding allowed organisation limits",
			fields: fields{
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
					WithQuery(`SELECT count(1) FROM "kafka_requests" WHERE instance_type = $1 AND (organisation_id = $2)`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply([]map[string]interface{}{{"count": "4"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organization 'org-id' has reached a maximum number of 4 allowed instances.",
				Code:     5,
			},
			args: args{
				instanceType: types.STANDARD,
			},
		},
		{
			name: "return an error when user in the allow list attempts to create an eval instance",
			fields: fields{
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
			args: args{
				instanceType: types.EVAL,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "0"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.InsufficientQuotaError("Insufficient Quota"),
		},
		{
			name: "return an error when user is not allowed in their org and they cannot create any more instances eval instances after exceeding default allowed user limits",
			fields: fields{
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
					WithQuery(`SELECT count(1) FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed instances.",
				Code:     5,
			},
			args: args{
				instanceType: types.EVAL,
			},
		},
		{
			name: "does not return an error if user is within limits for user creating a standard instance",
			fields: fields{
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
					WithQuery(`SELECT count(1) FROM "kafka_requests" WHERE instance_type = $1 AND (organisation_id = $2)`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply([]map[string]interface{}{{"count": "1"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.STANDARD,
			},
			wantErr: nil,
		},
		{
			name: "do not return an error when user who's not in the allow list can eval instances",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				AccessControlList: &acl.AccessControlListConfig{
					EnableInstanceLimitControl: true,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT count(1) FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2`).
					WithArgs(types.EVAL.String(), "username").
					WithReply([]map[string]interface{}{{"count": "0"}})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.EVAL,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.AccessControlList)
			quotaService, _ := factory.GetQuotaService(api.AllowListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			_, err := quotaService.ReserveQuota(kafka, tt.args.instanceType)
			gomega.Expect(tt.wantErr).To(gomega.Equal(err))
		})
	}
}
