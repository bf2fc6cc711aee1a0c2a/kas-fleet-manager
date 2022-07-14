package presenters

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
)

func TestConvertKafkaRequest(t *testing.T) {
	type args struct {
		kafkaRequestPayload public.KafkaRequestPayload
		dbKafkaRequests     []*dbapi.KafkaRequest
	}

	reauthEnabled := true
	reauthDisabled := false

	tests := []struct {
		name string
		args args
		want *dbapi.KafkaRequest
	}{
		{
			name: "should return converted KafkaRequest from empty db KafkaRequest",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(nil),
				dbKafkaRequests:     []*dbapi.KafkaRequest{},
			},
			want: mock.BuildKafkaRequest(
				mock.With(mock.REGION, mock.DefaultKafkaRequestRegion),
				mock.With(mock.CLOUD_PROVIDER, mock.DefaultKafkaRequestProvider),
				mock.With(mock.NAME, mock.DefaultKafkaRequestName),
				mock.WithReauthenticationEnabled(reauthEnabled),
			),
		},
		{
			name: "Should return empty kafka request with reauthentication disabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					payload.ReauthenticationEnabled = &reauthDisabled
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(
					mock.With(mock.REGION, mock.DefaultKafkaRequestRegion),
					mock.With(mock.CLOUD_PROVIDER, mock.DefaultKafkaRequestProvider),
					mock.With(mock.NAME, mock.DefaultKafkaRequestName),
					mock.WithReauthenticationEnabled(reauthDisabled),
				)},
			},
			want: mock.BuildKafkaRequest(
				mock.With(mock.REGION, mock.DefaultKafkaRequestRegion),
				mock.With(mock.CLOUD_PROVIDER, mock.DefaultKafkaRequestProvider),
				mock.With(mock.NAME, mock.DefaultKafkaRequestName),
				mock.WithReauthenticationEnabled(reauthDisabled),
			),
		},
		{
			name: "should convert and return kafka request from non-empty db kafka request slice with reauth enabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(nil),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(
					mock.WithPredefinedTestValues(),
					mock.WithReauthenticationEnabled(reauthEnabled),
				)},
			},
			want: mock.BuildKafkaRequest(
				mock.WithPredefinedTestValues(),
				mock.WithReauthenticationEnabled(reauthEnabled),
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

	defaultInstanceSize := *mocksupportedinstancetypes.BuildKafkaInstanceSize()

	tests := []struct {
		name   string
		args   args
		want   public.KafkaRequest
		config config.KafkaConfig
	}{
		{
			name: "should return kafka request as presented to an end user",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(
					mock.WithPredefinedTestValues(),
					mock.WithReauthenticationEnabled(reauthEnabled),
					mock.With(mock.BOOTSTRAP_SERVER_HOST, bootstrapServer),
					mock.With(mock.FAILED_REASON, failedReason),
					mock.With(mock.ACTUAL_KAFKA_VERSION, version),
					mock.With(mock.STORAGE_SIZE, kafkaStorageSize),
				),
			},
			want: *mock.BuildPublicKafkaRequest(func(kafkaRequest *public.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(bootstrapServer)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = mock.DefaultInstanceType
				kafkaRequest.DeprecatedKafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
				kafkaRequest.SizeId = defaultInstanceSize.Id
				kafkaRequest.IngressThroughputPerSec = defaultInstanceSize.IngressThroughputPerSec.String()
				kafkaRequest.EgressThroughputPerSec = defaultInstanceSize.EgressThroughputPerSec.String()
				kafkaRequest.TotalMaxConnections = int32(defaultInstanceSize.TotalMaxConnections)
				kafkaRequest.MaxPartitions = int32(defaultInstanceSize.MaxPartitions)
				kafkaRequest.MaxDataRetentionPeriod = defaultInstanceSize.MaxDataRetentionPeriod
				kafkaRequest.MaxConnectionAttemptsPerSec = int32(defaultInstanceSize.MaxConnectionAttemptsPerSec)

				expireTime := kafkaRequest.CreatedAt.Add(time.Duration(*defaultInstanceSize.LifespanSeconds) * time.Second)
				kafkaRequest.ExpiresAt = &expireTime

				dataRetentionSizeQuantity := config.Quantity(kafkaStorageSize)
				dataRetentionSizeBytes, err := dataRetentionSizeQuantity.ToInt64()
				g.Expect(err).ToNot(gomega.HaveOccurred(), "failed to convert kafka data retention size '%s' to bytes", kafkaStorageSize)

				kafkaRequest.MaxDataRetentionSize = public.SupportedKafkaSizeBytesValueItem{
					Bytes: dataRetentionSizeBytes,
				}
			}),
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: mock.DefaultInstanceType,
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
					g.Expect(kafkaRequest.IngressThroughputPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.EgressThroughputPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.TotalMaxConnections).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.MaxConnectionAttemptsPerSec).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.MaxDataRetentionPeriod).ToNot(gomega.BeNil())
					g.Expect(kafkaRequest.MaxPartitions).ToNot(gomega.BeNil())
				} else {
					g.Expect(kafkaRequest.IngressThroughputPerSec).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.EgressThroughputPerSec).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.TotalMaxConnections).To(gomega.BeZero())
					g.Expect(kafkaRequest.MaxConnectionAttemptsPerSec).To(gomega.BeZero())
					g.Expect(kafkaRequest.MaxDataRetentionPeriod).To(gomega.BeEmpty())
					g.Expect(kafkaRequest.MaxPartitions).To(gomega.BeZero())
				}
			} else {
				g.Expect(err).ToNot(gomega.BeNil())
			}
		})
	}
}
