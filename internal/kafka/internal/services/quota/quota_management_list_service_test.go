package quota

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"
	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
)

func Test_QuotaManagementListCheckQuota(t *testing.T) {
	type fields struct {
		connectionFactory   *db.ConnectionFactory
		QuotaManagementList *quota_management.QuotaManagementListConfig
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
			name: "do not throw an error when instance limit control is disabled when checking developer instances",
			fields: fields{
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the quota list and instance type is developer",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			want: true,
		},
		{
			name: "return true when user is not part of the quota list and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				instanceType: types.STANDARD,
			},
			want: false,
		},
		{
			name: "return true when user is part of the quota list as a service account and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
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
			name: "return true when user is part of the quota list under an organisation and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
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
			name: "return false when user is part of the quota list under an organisation and instance type is developer",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)

			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.QuotaManagementList, &defaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			allowed, _ := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka, tt.args.instanceType)
			gomega.Expect(tt.want).To(gomega.Equal(allowed))
		})
	}
}

var kafkaSupportedInstanceTypesConfig = config.KafkaSupportedInstanceTypesConfig{
	Configuration: config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			{
				Id:          "standard",
				DisplayName: "Standard",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x1",
						IngressThroughputPerSec:     "30Mi",
						EgressThroughputPerSec:      "30Mi",
						TotalMaxConnections:         1000,
						MaxDataRetentionSize:        "100Gi",
						MaxPartitions:               1000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 100,
						QuotaConsumed:               1,
						QuotaType:                   "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
			{
				Id:          "developer",
				DisplayName: "Trial",
				Sizes: []config.KafkaInstanceSize{
					{
						Id:                          "x1",
						IngressThroughputPerSec:     "60Mi",
						EgressThroughputPerSec:      "60Mi",
						TotalMaxConnections:         2000,
						MaxDataRetentionSize:        "200Gi",
						MaxPartitions:               2000,
						MaxDataRetentionPeriod:      "P14D",
						MaxConnectionAttemptsPerSec: 200,
						QuotaConsumed:               1,
						QuotaType:                   "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
		},
	},
}

var defaultKafkaConf = config.KafkaConfig{
	KafkaCapacity:          config.KafkaCapacityConfig{},
	Quota:                  config.NewKafkaQuotaConfig(),
	SupportedInstanceTypes: &kafkaSupportedInstanceTypesConfig,
}

var (
	testKafkaRequestRegion   = "us-east-1"
	testKafkaRequestProvider = "aws"
	testKafkaRequestName     = "test-cluster"
	testClusterID            = "test-cluster-id"
	testID                   = "test"
	testUser                 = "test-user"
)

func buildKafkaRequest(modifyFn func(kafkaRequest *dbapi.KafkaRequest)) *dbapi.KafkaRequest {
	kafkaRequest := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID:        testID,
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Region:        testKafkaRequestRegion,
		ClusterID:     testClusterID,
		CloudProvider: testKafkaRequestProvider,
		Name:          testKafkaRequestName,
		MultiAZ:       false,
		Owner:         testUser,
		SizeId:        "x1",
		InstanceType:  types.STANDARD.String(),
	}
	if modifyFn != nil {
		modifyFn(kafkaRequest)
	}
	return kafkaRequest
}

func Test_QuotaManagementListReserveQuota(t *testing.T) {
	type fields struct {
		connectionFactory   *db.ConnectionFactory
		QuotaManagementList *quota_management.QuotaManagementListConfig
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
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: false,
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			wantErr: nil,
		},
		{
			name: "return an error when the query db throws an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.GeneralError(fmt.Sprintf("Failed to check kafka capacity for instance type '%s'", types.DEVELOPER.String())),
		},
		{
			name: "return an error when user in an organisation cannot create any more instances after exceeding allowed organisation limits",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 1,
								AnyUser:             true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (organisation_id = $2) AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "Organization 'org-id' has reached a maximum number of 1 allowed streaming units.",
				Code:     5,
			},
			args: args{
				instanceType: types.STANDARD,
			},
		},
		{
			name: "return an error when user in the quota list attempts to create an developer instance",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "username").
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.InsufficientQuotaError("Insufficient Quota"),
		},
		{
			name: "return an error when user is not allowed in their org and they cannot create any more instances developer instances after exceeding default allowed user limits",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 2,
								AnyUser:             false,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "username").
					WithReply(converters.ConvertKafkaRequest(
						buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
							kafkaRequest.Owner = "username"
							kafkaRequest.InstanceType = types.DEVELOPER.String()
							kafkaRequest.OrganisationId = "org-id"
						})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "User 'username' has reached a maximum number of 1 allowed streaming units.",
				Code:     5,
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
		},
		{
			name: "does not return an error if user is within limits for user creating a standard instance",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:                  "org-id",
								MaxAllowedInstances: 4,
								AnyUser:             true,
							},
						},
					},
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (organisation_id = $2) AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.STANDARD.String(), "org-id").
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.STANDARD,
			},
			wantErr: nil,
		},
		{
			name: "do not return an error when user who's not in the quota list can developer instances",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND owner = $2 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "username").
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.DEVELOPER,
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
			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.QuotaManagementList, &defaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
				SizeId:         "x1",
				InstanceType:   tt.args.instanceType.String(),
			}
			_, err := quotaService.ReserveQuota(kafka, tt.args.instanceType)
			gomega.Expect(tt.wantErr).To(gomega.Equal(err))
		})
	}
}
