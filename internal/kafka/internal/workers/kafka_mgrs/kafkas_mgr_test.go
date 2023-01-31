package kafka_mgrs

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	dpMock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/onsi/gomega"
)

func TestKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService            services.KafkaService
		clusterService          services.ClusterService
		dataplaneClusterConfig  config.DataplaneClusterConfig
		cloudProviders          config.ProviderConfig
		accessControlListConfig *acl.AccessControlListConfig
		kafkaConfig             config.KafkaConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if setKafkaStatusCountMetric returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return nil, errors.GeneralError("failed to count kafkas by status")
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig:             *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if updateClusterStatusCapacityMetrics returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return nil, errors.GeneralError("failed to get kafka streaming unit count kafkas per cluster and instance type")
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig:             *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if DeprovisionExpiredKafkas returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return errors.GeneralError("failed to deprovision expired kafkas")
					},
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig: *config.NewDataplaneClusterConfig(),
				accessControlListConfig: &acl.AccessControlListConfig{
					EnableDenyList: true,
				},
				kafkaConfig: *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if ListAllFunc returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to list kafkas")
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig: *config.NewDataplaneClusterConfig(),
				accessControlListConfig: &acl.AccessControlListConfig{
					EnableDenyList: true,
				},
				kafkaConfig: *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileKafkaExpiresAt returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{
							{
								Meta: api.Meta{
									ID: "test",
								},
								Name:         "test",
								ClusterID:    "test",
								Status:       constants.KafkaRequestStatusAccepted.String(),
								InstanceType: "unsupported-instance-type",
								SizeId:       "unsupported-size-id",
							},
						}, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				accessControlListConfig: &acl.AccessControlListConfig{
					EnableDenyList: false,
				},
				kafkaConfig: *config.NewKafkaConfig(),
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:            tt.fields.kafkaService,
				clusterService:          tt.fields.clusterService,
				dataplaneClusterConfig:  &tt.fields.dataplaneClusterConfig,
				accessControlListConfig: tt.fields.accessControlListConfig,
				cloudProviders:          &tt.fields.cloudProviders,
				kafkaConfig:             &tt.fields.kafkaConfig,
			}

			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_ReconcileExpiredKafkas(t *testing.T) {
	type updateStatusCall struct {
		count  int
		status constants.KafkaStatus
	}

	type fields struct {
		kafkaService            *services.KafkaServiceMock
		clusterService          services.ClusterService
		dataplaneClusterConfig  config.DataplaneClusterConfig
		cloudProviders          config.ProviderConfig
		accessControlListConfig *acl.AccessControlListConfig
		kafkaConfig             config.KafkaConfig
	}
	tests := []struct {
		name             string
		fields           fields
		wantErr          bool
		updateStatusCall updateStatusCall
	}{
		{
			name: "should suspend kafka in grace period",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						k := dbapi.KafkaRequest{
							ClusterID:               "123-cluster-123",
							Name:                    "test-cluster",
							InstanceType:            types.STANDARD.String(),
							SizeId:                  "x1",
							Status:                  constants.KafkaRequestStatusAccepted.String(),
							ActualKafkaBillingModel: "eval",
							ExpiresAt: sql.NullTime{
								Time:  time.Now().AddDate(0, 0, +2),
								Valid: true,
							},
						}
						return dbapi.KafkaList{&k}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
					IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
						return false, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig: func() config.KafkaConfig {
					cfg := config.NewKafkaConfig()
					_ = cfg.ReadFiles()
					return *cfg
				}(),
			},
			wantErr: false,
			updateStatusCall: updateStatusCall{
				count:  1,
				status: constants.KafkaRequestStatusSuspending,
			},
		},
		{
			name: "should ignore expired kafkas",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						k := dbapi.KafkaRequest{
							ClusterID:               "123-cluster-123",
							Name:                    "test-cluster",
							InstanceType:            types.STANDARD.String(),
							SizeId:                  "x1",
							Status:                  constants.KafkaRequestStatusAccepted.String(),
							ActualKafkaBillingModel: "eval",
							ExpiresAt: sql.NullTime{
								Time:  time.Now().AddDate(0, 0, -1),
								Valid: true,
							},
						}
						return dbapi.KafkaList{&k}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
					IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
						return false, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig: func() config.KafkaConfig {
					cfg := config.NewKafkaConfig()
					_ = cfg.ReadFiles()
					return *cfg
				}(),
			},
			wantErr: false,
			updateStatusCall: updateStatusCall{
				count: 0,
			},
		},
		{
			name: "should ignore kafkas in unsupported status",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						k := dbapi.KafkaRequest{
							ClusterID:               "123-cluster-123",
							Name:                    "test-cluster",
							InstanceType:            types.STANDARD.String(),
							SizeId:                  "x1",
							Status:                  constants.KafkaRequestStatusSuspended.String(),
							ActualKafkaBillingModel: "eval",
							ExpiresAt: sql.NullTime{
								Time:  time.Now().AddDate(0, 0, +3),
								Valid: true,
							},
						}
						return dbapi.KafkaList{&k}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
					IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
						return false, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig: func() config.KafkaConfig {
					cfg := config.NewKafkaConfig()
					_ = cfg.ReadFiles()
					return *cfg
				}(),
			},
			wantErr: false,
			updateStatusCall: updateStatusCall{
				count: 0,
			},
		},
		{
			name: "should ignore kafkas that are not in a grace period",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						k := dbapi.KafkaRequest{
							ClusterID:               "123-cluster-123",
							Name:                    "test-cluster",
							InstanceType:            types.STANDARD.String(),
							SizeId:                  "x1",
							Status:                  constants.KafkaRequestStatusReady.String(),
							ActualKafkaBillingModel: "eval",
							ExpiresAt: sql.NullTime{
								Time:  time.Now().AddDate(0, 0, +10),
								Valid: true,
							},
						}
						return dbapi.KafkaList{&k}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
					IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
						return false, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig: func() config.KafkaConfig {
					cfg := config.NewKafkaConfig()
					_ = cfg.ReadFiles()
					return *cfg
				}(),
			},
			wantErr: false,
			updateStatusCall: updateStatusCall{
				count: 0,
			},
		},
		{
			name: "should ignore kafkas that never expire",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						k := dbapi.KafkaRequest{
							ClusterID:               "123-cluster-123",
							Name:                    "test-cluster",
							InstanceType:            types.STANDARD.String(),
							SizeId:                  "x1",
							Status:                  constants.KafkaRequestStatusReady.String(),
							ActualKafkaBillingModel: "eval",
							ExpiresAt: sql.NullTime{
								Time:  time.Time{},
								Valid: false,
							},
						}
						return dbapi.KafkaList{&k}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{}, nil
					},
					DeprovisionExpiredKafkasFunc: func() *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return true, nil
					},
					UpdateZeroValueOfKafkaRequestsExpiredAtFunc: func() error {
						return nil
					},
					IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
						return true, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{}, nil
					},
				},
				dataplaneClusterConfig:  *config.NewDataplaneClusterConfig(),
				accessControlListConfig: acl.NewAccessControlListConfig(),
				kafkaConfig: func() config.KafkaConfig {
					cfg := config.NewKafkaConfig()
					_ = cfg.ReadFiles()
					return *cfg
				}(),
			},
			wantErr: false,
			updateStatusCall: updateStatusCall{
				count: 0,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:            tt.fields.kafkaService,
				clusterService:          tt.fields.clusterService,
				dataplaneClusterConfig:  &tt.fields.dataplaneClusterConfig,
				accessControlListConfig: tt.fields.accessControlListConfig,
				cloudProviders:          &tt.fields.cloudProviders,
				kafkaConfig:             &tt.fields.kafkaConfig,
			}

			//k.Reconcile()
			g.Expect(len(k.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
			g.Expect(tt.fields.kafkaService.UpdateStatusCalls()).To(gomega.HaveLen(tt.updateStatusCall.count))
			if tt.updateStatusCall.count > 0 {
				g.Expect(tt.fields.kafkaService.UpdateStatusCalls()[0].Status).To(gomega.BeEquivalentTo(tt.updateStatusCall.status))
			}

		})
	}
}

func TestKafkaManager_reconcileDeniedKafkaOwners(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		deniedAccounts acl.DeniedUsers
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "do not reconcile when denied accounts list is empty",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					DeprovisionKafkaForUsersFunc: nil, // set to nil as it should not be called
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{},
			},
			wantErr: false,
		},
		{
			name: "should receive error when update in deprovisioning in database returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when update in deprovisioning in database succeed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService: tt.fields.kafkaService,
			}
			g.Expect(k.reconcileDeniedKafkaOwners(tt.args.deniedAccounts) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_setKafkaStatusCountMetric(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if CountByStatus fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return nil, errors.GeneralError("failed to count kafkas by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully set kafka status count metrics",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
					CountByStatusFunc: func(status []constants.KafkaStatus) ([]services.KafkaStatusCount, error) {
						return []services.KafkaStatusCount{
							{
								Status: constants.KafkaRequestStatusAccepted,
								Count:  2,
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := NewKafkaManager(tt.fields.kafkaService, nil, nil, nil, nil, workers.Reconciler{}, nil)

			g.Expect(k.setKafkaStatusCountMetric() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestKafkaManager_updateClusterStatusCapacityMetrics(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig config.DataplaneClusterConfig
		cloudProviders         config.ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if CountStreamingUnitByRegionAndInstanceType fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return nil, errors.GeneralError("failed to get kafka streaming unit count per cluster and instance type")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if calculateCapacityByRegionAndInstanceTypeForManualClusters fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("invalid"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if calculateAvailableAndMaxCapacityForDynamicScaling fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
								MaxUnits:      10,
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:                   "us-east-1",
										Default:                true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully assign metrics for manual clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard,developer"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should successfully assign metrics for enterprise clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
								ClusterType:   api.EnterpriseDataPlaneClusterType.String(),
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should successfully assign metrics for autoscaling mode",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindStreamingUnitCountByClusterAndInstanceTypeFunc: func() (services.KafkaStreamingUnitCountPerClusterList, error) {
						return services.KafkaStreamingUnitCountPerClusterList{
							{
								Region:        "us-east-1",
								InstanceType:  "standard",
								ClusterId:     "a",
								Count:         5,
								CloudProvider: "aws",
							},
						}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				clusterService:         tt.fields.clusterService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			g.Expect(k.updateClusterStatusCapacityMetrics() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

var (
	cloudProviderStandardLimit = 5
)

func TestKafkaManager_calculateCapacityByRegionAndInstanceTypeForManualClusters(t *testing.T) {
	type fields struct {
		kafkaService                 services.KafkaService
		dataplaneClusterConfig       config.DataplaneClusterConfig
		cloudProviders               config.ProviderConfig
		streamingUnitsCountPerRegion services.KafkaStreamingUnitCountPerClusterList
	}

	type counts struct {
		available int32
		max       int32
	}

	tests := []struct {
		name     string
		fields   fields
		expected map[string]map[string]counts
	}{
		{
			name: "expected available capacity with no cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: services.KafkaStreamingUnitCountPerClusterList{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         5,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used, 3 developer instances used
			// ==> expect 5 more standard instances to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 5,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with no cloud provider limit and existing developer instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard,developer"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: nil},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: services.KafkaStreamingUnitCountPerClusterList{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         5,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "cluster-id",
						Count:         3,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used
			// 3 developer instances used
			// ==> expect 2 more standard instances to be available
			// ==> expect 2 more developer instances to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 2,
						max:       10,
					},
					"developer": counts{
						available: 2,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: services.KafkaStreamingUnitCountPerClusterList{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// ==> expect 1 more standard instance to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       5,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing mixed instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("developer,standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitsCountPerRegion: services.KafkaStreamingUnitCountPerClusterList{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "cluster-id",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// 4 developer instances used
			// ==> expect 1 more standard instance to be available
			// ==> expect 2 more developer instance to be available
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       5,
					},
					"developer": counts{
						available: 2,
						max:       10,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			results, err := k.calculateCapacityByRegionAndInstanceTypeForManualClusters(tt.fields.streamingUnitsCountPerRegion)
			g.Expect(err).To(gomega.BeNil())

			for _, result := range results {
				count := tt.expected[result.Region][result.InstanceType]
				g.Expect(result.Count).To(gomega.Equal(count.available))
				g.Expect(result.MaxUnits).To(gomega.Equal(count.max))
			}
		})
	}
}

func TestKafkaManager_calculateAvailableAndMaxCapacityForDynamicScaling(t *testing.T) {
	type fields struct {
		kafkaService                services.KafkaService
		dataplaneClusterConfig      config.DataplaneClusterConfig
		cloudProviders              config.ProviderConfig
		streamingUnitCountPerRegion services.KafkaStreamingUnitCountPerCluster
	}

	type counts struct {
		available int32
		max       int32
	}

	tests := []struct {
		name     string
		fields   fields
		expected map[string]map[string]counts
	}{
		{
			name: "expected available capacity with no cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitCountPerRegion: services.KafkaStreamingUnitCountPerCluster{

					Region:        "us-east-1",
					InstanceType:  "standard",
					ClusterId:     "cluster-id",
					Count:         9,
					MaxUnits:      10,
					CloudProvider: "aws",
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 9 instances used,
			// ==> expect 1 available and max 10
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       10,
					},
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
				streamingUnitCountPerRegion: services.KafkaStreamingUnitCountPerCluster{

					Region:        "us-east-1",
					InstanceType:  "standard",
					ClusterId:     "cluster-id",
					Count:         5,
					MaxUnits:      9,
					CloudProvider: "aws",
				},
			},
			// 9 total capacity
			// 5 cloud provider limits
			// 5 instances used,
			// ==> expect 0 available and max 5
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 0,
						max:       5,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			result, err := k.calculateAvailableAndMaxCapacityForDynamicScaling(tt.fields.streamingUnitCountPerRegion)
			g.Expect(err).To(gomega.BeNil())

			count := tt.expected[result.Region][result.InstanceType]
			g.Expect(result.Count).To(gomega.Equal(count.available))
			g.Expect(result.MaxUnits).To(gomega.Equal(count.max))
		})
	}
}

// calculateAvailableAndMadCapacityForEnterpriseClusters
func TestKafkaManager_calculateAvailableAndMadCapacityForEnterpriseClusters(t *testing.T) {
	type args struct {
		streamingUnitCountPerRegion []services.KafkaStreamingUnitCountPerCluster
	}

	type fields struct {
		kafkaService           services.KafkaService
		dataplaneClusterConfig config.DataplaneClusterConfig
		cloudProviders         config.ProviderConfig
	}

	type counts struct {
		available int32
		max       int32
	}

	tests := []struct {
		name     string
		args     args
		fields   fields
		expected map[string]map[string]counts
	}{
		{
			name: "successfully returns max and available capacity for enterprise clusters",
			args: args{
				streamingUnitCountPerRegion: []services.KafkaStreamingUnitCountPerCluster{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         9,
						MaxUnits:      10,
						CloudProvider: "aws",
						ClusterType:   api.EnterpriseDataPlaneClusterType.String(),
					},
				},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 9 instances used,
			// ==> expect 1 available and max 10
			expected: map[string]map[string]counts{
				"us-east-1": {
					"standard": counts{
						available: 1,
						max:       10,
					},
				},
			},
		},
		{
			name: "should return empty result if no enterprise clusters are found",
			args: args{
				streamingUnitCountPerRegion: []services.KafkaStreamingUnitCountPerCluster{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "cluster-id",
						Count:         9,
						MaxUnits:      10,
						CloudProvider: "aws",
						ClusterType:   api.ManagedDataPlaneClusterType.String(),
					},
				},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return dbapi.KafkaList{}, nil
					},
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						dpMock.BuildManualCluster("standard"),
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]map[string]counts{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			result := k.calculateAvailableAndMadCapacityForEnterpriseClusters(tt.args.streamingUnitCountPerRegion)

			g.Expect(len(result)).To(gomega.Equal(len(tt.expected)))

			if len(result) > 0 {
				for _, res := range result {
					count := tt.expected[res.Region][res.InstanceType]
					g.Expect(res.Count).To(gomega.Equal(count.available))
					g.Expect(res.MaxUnits).To(gomega.Equal(count.max))
				}
			}
		})
	}
}

func TestKafkaManager_reconcileKafkaExpiresAt(t *testing.T) {
	type fields struct {
		kafkaService func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock
		kafkaConfig  *config.KafkaConfig
	}
	type args struct {
		kafkas dbapi.KafkaList
	}

	tests := []struct {
		name                  string
		fields                fields
		args                  args
		wantNewExpiresAtValue sql.NullTime
		wantUpdateCallCount   int
		wantErrCount          int
	}{
		{
			name: "should skip expires_at reconciliation if kafka instance is in a deprovisioning state",
			fields: fields{
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						Status: constants.KafkaRequestStatusDeprovision.String(),
					},
				},
			},
			wantErrCount:          0,
			wantUpdateCallCount:   0,
			wantNewExpiresAtValue: sql.NullTime{},
		},
		{
			name: "should skip expires_at reconciliation if kafka instance is in a deleting state",
			fields: fields{
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						Status: constants.KafkaRequestStatusDeleting.String(),
					},
				},
			},
			wantErrCount:          0,
			wantUpdateCallCount:   0,
			wantNewExpiresAtValue: sql.NullTime{},
		},
		{
			name: "should return an error if it fails to get kafka instance size",
			fields: fields{
				kafkaConfig: config.NewKafkaConfig(),
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType: "unsupported-instance-type",
						SizeId:       "unsupported-size",
						Status:       constants.KafkaRequestStatusReady.String(),
					},
				},
			},
			wantErrCount:          1,
			wantUpdateCallCount:   0,
			wantNewExpiresAtValue: sql.NullTime{},
		},
		{
			name: "should return an error if it fails to check the quota entitlement status",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return false, fmt.Errorf("failed to check quota entitlement status")
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType: "instance-type-1",
						SizeId:       "x1",
						Status:       constants.KafkaRequestStatusReady.String(),
					},
				},
			},
			wantErrCount:          1,
			wantUpdateCallCount:   0,
			wantNewExpiresAtValue: sql.NullTime{},
		},
		{
			name: "should update expires_at to null if quota entitlement is active and expires_at has a value",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return true, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType: "instance-type-1",
						SizeId:       "x1",
						Status:       constants.KafkaRequestStatusReady.String(),
						ExpiresAt: sql.NullTime{
							Time:  time.Now().AddDate(0, 0, 10),
							Valid: true,
						},
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should skip update if quota entitlement is active and expires_at is already null",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return true, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType: "instance-type-1",
						SizeId:       "x1",
						Status:       constants.KafkaRequestStatusReady.String(),
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{},
			wantUpdateCallCount:   0,
			wantErrCount:          0,
		},
		{
			name: "should update expires_at with grace period if quota entitlement is not active and expires_at is null",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID:              "kafka-bm1",
											GracePeriodDays: 10,
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return false, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{Time: time.Now().AddDate(0, 0, 10), Valid: true},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should update expires_at without grace period if quota entitlement is not active and expires_at is null",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return false, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{Time: time.Now(), Valid: true},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should update expires_at if quota entitlement is not active and expires_at is set to a zero time value",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return false, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
						ExpiresAt: sql.NullTime{
							Time:  time.Time{},
							Valid: true,
						},
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{Time: time.Now(), Valid: true},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should skip update if quota entitlement is not active and expires_at is already set to a non-zero time value",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id: "x1",
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						IsQuotaEntitlementActiveFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
							return false, nil
						},
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
						ExpiresAt: sql.NullTime{
							Time:  time.Now(),
							Valid: true,
						},
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{},
			wantUpdateCallCount:   0,
			wantErrCount:          0,
		},
		{
			name: "should update expires_at based on lifespanSeconds if defined and expires_at is set to null",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:              "x1",
											LifespanSeconds: &[]int{172800}[0],
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						Meta: api.Meta{
							CreatedAt: time.Now(),
						},
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{Time: time.Now().Add(time.Duration(172800) * time.Second), Valid: true},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should update expires_at based on lifespanSeconds if defined and expires_at is set to a zero time value",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:              "x1",
											LifespanSeconds: &[]int{172800}[0],
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						Meta: api.Meta{
							CreatedAt: time.Now(),
						},
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
						ExpiresAt: sql.NullTime{
							Time:  time.Time{},
							Valid: true,
						},
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{Time: time.Now().Add(time.Duration(172800) * time.Second), Valid: true},
			wantUpdateCallCount:   1,
			wantErrCount:          0,
		},
		{
			name: "should skip updated based on lifespanSeconds if defined if expires_at is already set",
			fields: fields{
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								{
									Id: "instance-type-1",
									Sizes: []config.KafkaInstanceSize{
										{
											Id:              "x1",
											LifespanSeconds: &[]int{172800}[0],
										},
									},
									SupportedBillingModels: []config.KafkaBillingModel{
										{
											ID: "kafka-bm1",
										},
									},
								},
							},
						},
					},
				},
				kafkaService: func(updatedExpiresAt *sql.NullTime) *services.KafkaServiceMock {
					return &services.KafkaServiceMock{
						UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
							*updatedExpiresAt = values["expires_at"].(sql.NullTime)
							return nil
						},
					}
				},
			},
			args: args{
				kafkas: dbapi.KafkaList{
					{
						InstanceType:            "instance-type-1",
						SizeId:                  "x1",
						Status:                  constants.KafkaRequestStatusReady.String(),
						ActualKafkaBillingModel: "kafka-bm1",
						ExpiresAt: sql.NullTime{
							Time:  time.Now(),
							Valid: true,
						},
					},
				},
			},
			wantNewExpiresAtValue: sql.NullTime{},
			wantUpdateCallCount:   0,
			wantErrCount:          0,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			updatedExpiresAt := sql.NullTime{}
			mockKafkaService := tt.fields.kafkaService(&updatedExpiresAt)
			k := &KafkaManager{
				kafkaService: mockKafkaService,
				kafkaConfig:  tt.fields.kafkaConfig,
			}
			err := k.reconcileKafkaExpiresAt(tt.args.kafkas)
			g.Expect(len(err)).To(gomega.Equal(tt.wantErrCount))

			g.Expect(len(mockKafkaService.UpdatesCalls())).To(gomega.Equal(tt.wantUpdateCallCount), "expected update call count does not match actual")
			g.Expect(updatedExpiresAt.Valid).To(gomega.Equal(tt.wantNewExpiresAtValue.Valid), "expected expires_at.valid does not match actual")

			// The expires_at value set by checking the quota entitlement status is based on time.Now() captured during reconcile.
			// Because of this we can't know the exact time of the new expires_at value to do an equality check, however, we can still do comparison
			// on the year, month and day
			g.Expect(updatedExpiresAt.Time.Year()).To(gomega.Equal(tt.wantNewExpiresAtValue.Time.Year()), "expected expires_at year does not match actual")
			g.Expect(updatedExpiresAt.Time.Month()).To(gomega.Equal(tt.wantNewExpiresAtValue.Time.Month()), "expected expires_at month does not match actual")
			g.Expect(updatedExpiresAt.Time.Day()).To(gomega.Equal(tt.wantNewExpiresAtValue.Time.Day()), "expected expires_at day does not match actual")
		})
	}
}
