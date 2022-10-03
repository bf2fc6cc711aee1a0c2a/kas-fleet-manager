package quota

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/converters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/quota/internal/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"

	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/gorm"
)

func Test_QuotaManagementListCheckQuota(t *testing.T) {
	type fields struct {
		connectionFactory   *db.ConnectionFactory
		QuotaManagementList *quota_management.QuotaManagementListConfig
	}

	type args struct {
		instanceType types.KafkaInstanceType
		billingModel config.KafkaBillingModel
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
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
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
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
			},
			want: true,
		},
		{
			name: "return false when user is not part of the quota list and instance type is standard",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				instanceType: types.STANDARD,
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
			},
			want: false,
		},
		{
			name: "Test user is part of the quota list as a service account and instance type is standard",
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
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
			},
			want: true,
		},
		{
			name: "Test user is part of the quota list as a service account and instance type is standard/BILLING_MODEL=EVAL, but eval is not allowed",
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
				instanceType: "STANDARD",
				billingModel: config.KafkaBillingModel{ID: "EVAL"},
			},
			want: false,
		},
		{
			name: "Test user is part of the quota list as a service account and instance type is standard/BILLING_MODEL=EVAL and eval is allowed",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				QuotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							quota_management.Account{
								Username:            "username",
								MaxAllowedInstances: 4,
								GrantedQuota: []quota_management.Quota{
									{
										InstanceTypeID: "standard",
										KafkaBillingModels: []quota_management.BillingModel{
											{
												Id: "EVAL",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				instanceType: "standard",
				billingModel: config.KafkaBillingModel{
					ID: "EVAL",
				},
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
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
			},
			want: true,
		},
		{
			name: "return false when user is part of the quota list under an organisation and instance type is standard/EVAL, but EVAL is not allowed",
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
				billingModel: config.KafkaBillingModel{
					ID: "EVAL",
				},
			},
			want: false,
		},
		{
			name: "return true when user is part of the quota list under an organisation and instance type is standard/EVAL and EVAL is allowed",
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
								GrantedQuota: []quota_management.Quota{
									{
										InstanceTypeID: "standard",
										KafkaBillingModels: []quota_management.BillingModel{
											{
												Id: "EVAL",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				instanceType: types.STANDARD,
				billingModel: config.KafkaBillingModel{
					ID: "EVAL",
				},
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
				billingModel: config.KafkaBillingModel{ID: "STANDARD"},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(nil, tt.fields.connectionFactory, tt.fields.QuotaManagementList, &defaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)
			kafka := &dbapi.KafkaRequest{
				Owner:          "username",
				OrganisationId: "org-id",
			}
			allowed, _ := quotaService.CheckIfQuotaIsDefinedForInstanceType(kafka.Owner, kafka.OrganisationId, tt.args.instanceType, tt.args.billingModel)
			g.Expect(tt.want).To(gomega.Equal(allowed))
		})
	}
}

var defaultKafkaConf = config.KafkaConfig{
	Quota:                  config.NewKafkaQuotaConfig(),
	SupportedInstanceTypes: test.NewQuotaListTestKafkaSupportedInstanceTypesConfig(),
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
			wantErr: errors.GeneralError(fmt.Sprintf("failed to check kafka capacity for instance type '%s'", types.DEVELOPER.String())),
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
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (actual_kafka_billing_model = $2 or desired_kafka_billing_model = $3) AND organisation_id = $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.STANDARD.String(), "standard", "standard", "org-id").
					WithReply(converters.ConvertKafkaRequest(buildKafkaRequest(nil)))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "organization 'org-id' has reached a maximum number of 1 allowed streaming units",
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
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (actual_kafka_billing_model = $2 or desired_kafka_billing_model = $3) AND owner = $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "standard", "standard", "username").
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: errors.InsufficientQuotaError("Insufficient quota"),
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
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (actual_kafka_billing_model = $2 or desired_kafka_billing_model = $3) AND owner = $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "standard", "standard", "username").
					WithReply(converters.ConvertKafkaRequest(
						buildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
							kafkaRequest.Owner = "username"
							kafkaRequest.InstanceType = types.DEVELOPER.String()
							kafkaRequest.OrganisationId = "org-id"
							kafkaRequest.ActualKafkaBillingModel = "standard"
						})))
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: &errors.ServiceError{
				HttpCode: http.StatusForbidden,
				Reason:   "user 'username' has reached a maximum number of 1 allowed streaming units",
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
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (actual_kafka_billing_model = $2 or desired_kafka_billing_model = $3) AND organisation_id = $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.STANDARD.String(), "standard", "standard", "org-id").
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
					WithQuery(`SELECT * FROM "kafka_requests" WHERE instance_type = $1 AND (actual_kafka_billing_model = $2 or desired_kafka_billing_model = $3) AND owner = $4 AND "kafka_requests"."deleted_at" IS NULL`).
					WithArgs(types.DEVELOPER.String(), "standard", "standard", "username").
					WithReply(nil)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			args: args{
				instanceType: types.DEVELOPER,
			},
			wantErr: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			_, err := quotaService.ReserveQuota(kafka)
			g.Expect(tt.wantErr).To(gomega.Equal(err))
		})
	}
}
func Test_DefaultQuotaServiceFactory_GetQuotaService(t *testing.T) {
	type fields struct {
		QuotaServiceContainer map[api.QuotaType]services.QuotaService
	}
	type args struct {
		quotaType api.QuotaType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    services.QuotaService
		wantErr *errors.ServiceError
	}{
		{
			name: "Should return nil and error if QuotaType is invalid",
			fields: fields{
				QuotaServiceContainer: map[api.QuotaType]services.QuotaService{},
			},
			args: args{
				quotaType: api.UndefinedQuotaType,
			},
			want:    nil,
			wantErr: errors.GeneralError("invalid quota service type: %v", api.QuotaManagementListQuotaType),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := &DefaultQuotaServiceFactory{
				quotaServiceContainer: map[api.QuotaType]services.QuotaService{},
			}
			quotaService, err := factory.GetQuotaService(tt.args.quotaType)
			g.Expect(quotaService).To(gomega.BeNil())
			g.Expect(err).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestQuotaManagementListService_IsQuotaEntitlementActive(t *testing.T) {
	type fields struct {
		quotaManagementList *quota_management.QuotaManagementListConfig
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}

	expiredDate := quota_management.ExpirationDate(time.Now().AddDate(0, 0, -1))
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "return true if org has an active quota entitlement",
			fields: fields{
				quotaManagementList: &quota_management.QuotaManagementListConfig{
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:      "org-id",
								AnyUser: true,
								GrantedQuota: quota_management.QuotaList{
									{
										InstanceTypeID: "instance-type-1",
										KafkaBillingModels: quota_management.BillingModelList{
											{
												Id:                  "kafka-billing-model-1",
												MaxAllowedInstances: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					OrganisationId:          "org-id",
					ActualKafkaBillingModel: "kafka-billing-model-1",
					InstanceType:            "instance-type-1",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "return false if quota entitled to an organisation but user is not registered to that organisation",
			fields: fields{
				quotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						Organisations: quota_management.OrganisationList{
							quota_management.Organisation{
								Id:      "org-id",
								AnyUser: false,
								RegisteredUsers: quota_management.AccountList{
									{
										Username: "user-1",
									},
								},
								GrantedQuota: quota_management.QuotaList{
									{
										InstanceTypeID: "instance-type-1",
										KafkaBillingModels: quota_management.BillingModelList{
											{
												Id:                  "kafka-billing-model-1",
												MaxAllowedInstances: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					OrganisationId:          "org-id",
					ActualKafkaBillingModel: "kafka-billing-model-1",
					InstanceType:            "instance-type-1",
					Owner:                   "unregistered-user",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "return false if quota is not entitled to the organisation or user",
			fields: fields{
				quotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList:                  quota_management.RegisteredUsersListConfiguration{},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					OrganisationId:          "org-id",
					ActualKafkaBillingModel: "kafka-billing-model-1",
					InstanceType:            "instance-type-1",
					Owner:                   "user-1",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "return true if user has an active quota entitlement",
			fields: fields{
				quotaManagementList: &quota_management.QuotaManagementListConfig{
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							{
								Username: "user-1",
								GrantedQuota: quota_management.QuotaList{
									{
										InstanceTypeID: "instance-type-1",
										KafkaBillingModels: quota_management.BillingModelList{
											{
												Id:                  "kafka-billing-model-1",
												MaxAllowedInstances: 1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					OrganisationId:          "org-id",
					ActualKafkaBillingModel: "kafka-billing-model-1",
					InstanceType:            "instance-type-1",
					Owner:                   "user-1",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "return false if quota entitled to an individual user is expired",
			fields: fields{
				quotaManagementList: &quota_management.QuotaManagementListConfig{
					EnableInstanceLimitControl: true,
					QuotaList: quota_management.RegisteredUsersListConfiguration{
						ServiceAccounts: quota_management.AccountList{
							{
								Username: "user-1",
								GrantedQuota: quota_management.QuotaList{
									{
										InstanceTypeID: "instance-type-1",
										KafkaBillingModels: quota_management.BillingModelList{
											{
												Id:                  "kafka-billing-model-1",
												MaxAllowedInstances: 1,
												ExpirationDate:      &expiredDate,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					OrganisationId:          "org-id",
					ActualKafkaBillingModel: "kafka-billing-model-1",
					InstanceType:            "instance-type-1",
					Owner:                   "user-1",
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			factory := NewDefaultQuotaServiceFactory(nil, nil, tt.fields.quotaManagementList, &defaultKafkaConf)
			quotaService, _ := factory.GetQuotaService(api.QuotaManagementListQuotaType)

			got, err := quotaService.IsQuotaEntitlementActive(tt.args.kafka)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
