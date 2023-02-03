package presenters

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
)

func TestConvertKafkaRequest(t *testing.T) {
	type args struct {
		kafkaRequestPayload public.KafkaRequestPayload
		dbKafkaRequests     []*dbapi.KafkaRequest
	}

	reauthEnabled := true
	reauthDisabled := false
	clusterID := "dsdhsjdhsd"

	tests := []struct {
		name string
		args args
		want *dbapi.KafkaRequest
	}{
		{
			name: "should return converted KafkaRequest from empty db KafkaRequest",
			args: args{
				kafkaRequestPayload: *mocks.BuildKafkaRequestPayload(nil),
				dbKafkaRequests:     []*dbapi.KafkaRequest{},
			},
			want: mocks.BuildKafkaRequest(
				mocks.With(mocks.REGION, mocks.DefaultKafkaRequestRegion),
				mocks.With(mocks.CLOUD_PROVIDER, mocks.DefaultKafkaRequestProvider),
				mocks.With(mocks.NAME, mocks.DefaultKafkaRequestName),
				mocks.WithReauthenticationEnabled(reauthEnabled),
			),
		},
		{
			name: "Should return empty kafka request with reauthentication disabled",
			args: args{
				kafkaRequestPayload: *mocks.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					payload.ReauthenticationEnabled = &reauthDisabled
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{mocks.BuildKafkaRequest(
					mocks.With(mocks.REGION, mocks.DefaultKafkaRequestRegion),
					mocks.With(mocks.CLOUD_PROVIDER, mocks.DefaultKafkaRequestProvider),
					mocks.With(mocks.NAME, mocks.DefaultKafkaRequestName),
					mocks.WithReauthenticationEnabled(reauthDisabled),
				)},
			},
			want: mocks.BuildKafkaRequest(
				mocks.With(mocks.REGION, mocks.DefaultKafkaRequestRegion),
				mocks.With(mocks.CLOUD_PROVIDER, mocks.DefaultKafkaRequestProvider),
				mocks.With(mocks.NAME, mocks.DefaultKafkaRequestName),
				mocks.WithReauthenticationEnabled(reauthDisabled),
			),
		},
		{
			name: "should convert and return kafka request from non-empty db kafka request slice with reauth enabled",
			args: args{
				kafkaRequestPayload: *mocks.BuildKafkaRequestPayload(nil),
				dbKafkaRequests: []*dbapi.KafkaRequest{mocks.BuildKafkaRequest(
					mocks.WithPredefinedTestValues(),
					mocks.WithReauthenticationEnabled(reauthEnabled),
				)},
			},
			want: mocks.BuildKafkaRequest(
				mocks.WithPredefinedTestValues(),
				mocks.WithReauthenticationEnabled(reauthEnabled),
			),
		},
		{
			name: "Should convert clusterID if provided",
			args: args{
				kafkaRequestPayload: *mocks.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					payload.ClusterId = &clusterID
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{},
			},
			want: mocks.BuildKafkaRequest(
				mocks.With(mocks.REGION, mocks.DefaultKafkaRequestRegion),
				mocks.With(mocks.CLOUD_PROVIDER, mocks.DefaultKafkaRequestProvider),
				mocks.With(mocks.NAME, mocks.DefaultKafkaRequestName),
				mocks.With(mocks.CLUSTER_ID, clusterID),
				mocks.WithReauthenticationEnabled(reauthEnabled),
				mocks.With(mocks.DESIRED_KAFKA_BILLING_MODEL, constants.BillingModelEnterprise.String()),
			),
		},
		{
			name: "should convert and return kafka request from empty db kafka request with desired billing model",
			args: args{
				kafkaRequestPayload: *mocks.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					billingModelStr := "mybillingmodel"
					payload.BillingModel = &billingModelStr
					payload.ClusterId = &clusterID
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{},
			},
			want: mocks.BuildKafkaRequest(
				mocks.With(mocks.REGION, mocks.DefaultKafkaRequestRegion),
				mocks.With(mocks.CLOUD_PROVIDER, mocks.DefaultKafkaRequestProvider),
				mocks.With(mocks.NAME, mocks.DefaultKafkaRequestName),
				mocks.WithReauthenticationEnabled(reauthEnabled),
				mocks.With(mocks.CLUSTER_ID, clusterID),
				mocks.With(mocks.DESIRED_KAFKA_BILLING_MODEL, "mybillingmodel"),
			),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(ConvertKafkaRequest(tt.args.kafkaRequestPayload, tt.args.dbKafkaRequests...)).To(gomega.Equal(tt.want))
		})
	}
}

func TestPresentKafkaRequest(t *testing.T) {
	g := gomega.NewWithT(t)

	type args struct {
		dbKafkaRequest *dbapi.KafkaRequest
	}

	bootstrapServer := "http://test.com"
	failedReason := ""
	version := "2.8.0"
	reauthEnabled := true
	kafkaStorageSize := "1000Gi"

	clusterID := mocks.DefaultClusterID

	defaultInstanceSize := *mocksupportedinstancetypes.BuildKafkaInstanceSize()
	nowTime := time.Now()

	tests := []struct {
		name   string
		args   args
		want   public.KafkaRequest
		config config.KafkaConfig
	}{
		{
			name: "should return kafka request as presented to an end user",
			args: args{
				dbKafkaRequest: mocks.BuildKafkaRequest(
					mocks.WithPredefinedTestValues(),
					mocks.WithReauthenticationEnabled(reauthEnabled),
					mocks.With(mocks.BOOTSTRAP_SERVER_HOST, bootstrapServer),
					mocks.With(mocks.FAILED_REASON, failedReason),
					mocks.With(mocks.ACTUAL_KAFKA_VERSION, version),
					mocks.With(mocks.STORAGE_SIZE, kafkaStorageSize),
					mocks.With(mocks.DESIRED_KAFKA_BILLING_MODEL, "mydesiredkafkabillingmodel"),
					mocks.With(mocks.ACTUAL_KAFKA_BILLING_MODEL, "myactualkafkabillingmodel"),
					mocks.WithCreatedAt(nowTime),
					mocks.WithExpiresAt(sql.NullTime{Time: nowTime.Add(time.Duration(*defaultInstanceSize.LifespanSeconds) * time.Second), Valid: true}),
				),
			},
			want: *mocks.BuildPublicKafkaRequest(func(kafkaRequest *public.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(bootstrapServer)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = mocks.DefaultInstanceType
				kafkaRequest.DeprecatedKafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
				kafkaRequest.SizeId = defaultInstanceSize.Id
				kafkaRequest.DeprecatedIngressThroughputPerSec = defaultInstanceSize.IngressThroughputPerSec.String()
				kafkaRequest.DeprecatedEgressThroughputPerSec = defaultInstanceSize.EgressThroughputPerSec.String()
				kafkaRequest.DeprecatedTotalMaxConnections = int32(defaultInstanceSize.TotalMaxConnections)
				kafkaRequest.DeprecatedMaxPartitions = int32(defaultInstanceSize.MaxPartitions)
				kafkaRequest.DeprecatedMaxDataRetentionPeriod = defaultInstanceSize.MaxDataRetentionPeriod
				kafkaRequest.DeprecatedMaxConnectionAttemptsPerSec = int32(defaultInstanceSize.MaxConnectionAttemptsPerSec)

				kafkaRequest.CreatedAt = nowTime
				expireTime := kafkaRequest.CreatedAt.Add(time.Duration(*defaultInstanceSize.LifespanSeconds) * time.Second)
				kafkaRequest.ExpiresAt = &expireTime

				dataRetentionSizeQuantity := config.Quantity(kafkaStorageSize)
				dataRetentionSizeBytes, err := dataRetentionSizeQuantity.ToInt64()
				g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to convert kafka data retention size '%s' to bytes", kafkaStorageSize)

				kafkaRequest.MaxDataRetentionSize = public.SupportedKafkaSizeBytesValueItem{
					Bytes: dataRetentionSizeBytes,
				}
				kafkaRequest.BillingModel = "myactualkafkabillingmodel"
			}),
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: mocks.DefaultInstanceType,
								Sizes: []config.KafkaInstanceSize{
									defaultInstanceSize,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should return kafka request as presented to an end user when billing model is enterprise",
			args: args{
				dbKafkaRequest: mocks.BuildKafkaRequest(
					mocks.WithPredefinedTestValues(),
					mocks.WithReauthenticationEnabled(reauthEnabled),
					mocks.With(mocks.BOOTSTRAP_SERVER_HOST, bootstrapServer),
					mocks.With(mocks.FAILED_REASON, failedReason),
					mocks.With(mocks.ACTUAL_KAFKA_VERSION, version),
					mocks.With(mocks.STORAGE_SIZE, kafkaStorageSize),
					mocks.With(mocks.DESIRED_KAFKA_BILLING_MODEL, constants.BillingModelEnterprise.String()),
					mocks.With(mocks.ACTUAL_KAFKA_BILLING_MODEL, constants.BillingModelEnterprise.String()),
					mocks.WithCreatedAt(nowTime),
					mocks.WithExpiresAt(sql.NullTime{Time: nowTime.Add(time.Duration(*defaultInstanceSize.LifespanSeconds) * time.Second), Valid: true}),
				),
			},
			want: *mocks.BuildPublicKafkaRequest(func(kafkaRequest *public.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(bootstrapServer)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = mocks.DefaultInstanceType
				kafkaRequest.DeprecatedKafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
				kafkaRequest.SizeId = defaultInstanceSize.Id
				kafkaRequest.DeprecatedIngressThroughputPerSec = defaultInstanceSize.IngressThroughputPerSec.String()
				kafkaRequest.DeprecatedEgressThroughputPerSec = defaultInstanceSize.EgressThroughputPerSec.String()
				kafkaRequest.DeprecatedTotalMaxConnections = int32(defaultInstanceSize.TotalMaxConnections)
				kafkaRequest.DeprecatedMaxPartitions = int32(defaultInstanceSize.MaxPartitions)
				kafkaRequest.DeprecatedMaxDataRetentionPeriod = defaultInstanceSize.MaxDataRetentionPeriod
				kafkaRequest.DeprecatedMaxConnectionAttemptsPerSec = int32(defaultInstanceSize.MaxConnectionAttemptsPerSec)
				kafkaRequest.ClusterId = &clusterID
				kafkaRequest.CreatedAt = nowTime
				expireTime := kafkaRequest.CreatedAt.Add(time.Duration(*defaultInstanceSize.LifespanSeconds) * time.Second)
				kafkaRequest.ExpiresAt = &expireTime

				dataRetentionSizeQuantity := config.Quantity(kafkaStorageSize)
				dataRetentionSizeBytes, err := dataRetentionSizeQuantity.ToInt64()
				g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to convert kafka data retention size '%s' to bytes", kafkaStorageSize)

				kafkaRequest.MaxDataRetentionSize = public.SupportedKafkaSizeBytesValueItem{
					Bytes: dataRetentionSizeBytes,
				}
				kafkaRequest.BillingModel = constants.BillingModelEnterprise.String()
			}),
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: mocks.DefaultInstanceType,
								Sizes: []config.KafkaInstanceSize{
									defaultInstanceSize,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(PresentKafkaRequest(tt.args.dbKafkaRequest, &tt.config)).To(gomega.Equal(tt.want))
		})
	}
}

func TestSetBootstrapServerHost(t *testing.T) {
	type args struct {
		bootstrapServerHost string
	}
	nonEmptyBootstrapServerHost := "http://some-url.com"

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "non empty bootstrap server host conversion should return its value with colon and port number as a suffix",
			args: args{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
			},
			want: fmt.Sprintf("%s:443", nonEmptyBootstrapServerHost),
		},
		{
			name: "empty bootstrap server host conversion should return empty value",
			args: args{
				bootstrapServerHost: "",
			},
			want: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(setBootstrapServerHost(tt.args.bootstrapServerHost)).To(gomega.Equal(tt.want))
		})
	}
}

func TestCapacityLimitReports(t *testing.T) {
	tests := []struct {
		name        string
		request     dbapi.KafkaRequest
		config      config.KafkaConfig
		negative    bool
		errExpected bool
	}{
		{
			name: "Size exists for the instance type",
			request: dbapi.KafkaRequest{
				Meta:             api.Meta{},
				Region:           "us-east-1",
				ClusterID:        "xyz",
				CloudProvider:    "aws",
				MultiAZ:          true,
				Name:             "test-cluster",
				Status:           "ready",
				KafkaStorageSize: "60GB",
				InstanceType:     "standard",
				QuotaType:        "rhosak",
				SizeId:           "x1",
			},
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
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
										CapacityConsumed:            0,
										MaxMessageSize:              "1Mi",
										MinInSyncReplicas:           2,
										ReplicationFactor:           3,
									},
								},
							},
						},
					},
				},
			},
			negative:    false,
			errExpected: false,
		},
		{
			name: "Size doesn't exist for the instance type",
			request: dbapi.KafkaRequest{
				Meta:             api.Meta{},
				Region:           "us-east-1",
				ClusterID:        "xyz",
				CloudProvider:    "aws",
				MultiAZ:          true,
				Name:             "test-cluster",
				Status:           "ready",
				KafkaStorageSize: "60GB",
				InstanceType:     "developer",
				QuotaType:        "rhosak",
				SizeId:           "x1",
			},
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
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
										CapacityConsumed:            0,
										MaxMessageSize:              "1Mi",
										MinInSyncReplicas:           2,
										ReplicationFactor:           3,
									},
								},
							},
						},
					},
				},
			},
			negative:    true,
			errExpected: true,
		},
		{
			name: "Size doesn't exist for the instance type",
			request: dbapi.KafkaRequest{
				Meta:             api.Meta{},
				Region:           "us-east-1",
				ClusterID:        "xyz",
				CloudProvider:    "aws",
				MultiAZ:          true,
				Name:             "test-cluster",
				Status:           "ready",
				KafkaStorageSize: "60GB",
				InstanceType:     "standard",
				QuotaType:        "rhosak",
				SizeId:           "x1",
			},
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id:          "standard",
								DisplayName: "Standard",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:                          "x2",
										IngressThroughputPerSec:     "30Mi",
										EgressThroughputPerSec:      "30Mi",
										TotalMaxConnections:         1000,
										MaxDataRetentionSize:        "100Gi",
										MaxPartitions:               1000,
										MaxDataRetentionPeriod:      "P14D",
										MaxConnectionAttemptsPerSec: 100,
										QuotaConsumed:               1,
										CapacityConsumed:            0,
										MaxMessageSize:              "1Mi",
										MinInSyncReplicas:           2,
										ReplicationFactor:           3,
									},
								},
							},
						},
					},
				},
			},
			negative: true,
		},
	}
	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			kafkaRequest, err := PresentKafkaRequest(&test.request, &test.config)
			if !test.errExpected {
				if !test.negative {
					g.Expect(kafkaRequest.DeprecatedIngressThroughputPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.DeprecatedEgressThroughputPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.DeprecatedTotalMaxConnections).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.DeprecatedMaxConnectionAttemptsPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.DeprecatedMaxDataRetentionPeriod).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.DeprecatedMaxPartitions).ToNot(gomega.BeNil())
				} else {
					g.Expect(kafkaRequest.DeprecatedIngressThroughputPerSec).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.DeprecatedEgressThroughputPerSec).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.DeprecatedTotalMaxConnections).To(gomega.BeZero())
					g.Expect(kafkaRequest.DeprecatedMaxConnectionAttemptsPerSec).To(gomega.BeZero())
					g.Expect(kafkaRequest.DeprecatedMaxDataRetentionPeriod).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.DeprecatedMaxPartitions).To(gomega.BeZero())
				}
			} else {
				g.Expect(err).ToNot(gomega.BeNil())
			}
		})
	}
}
