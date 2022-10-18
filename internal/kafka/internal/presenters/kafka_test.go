package presenters

import (
	"fmt"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
)

var (
	nonEmptyBootstrapServerHost = "http://test.com"
	sampleReadyRequest          = dbapi.KafkaRequest{
		BootstrapServerHost: nonEmptyBootstrapServerHost,
		Status:              constants.KafkaRequestStatusReady.String(),
	}
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
				kafkaRequest.Status = constants.KafkaRequestStatusReady.String()
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(&sampleReadyRequest)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = mock.DefaultInstanceType
				kafkaRequest.DeprecatedKafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
				kafkaRequest.SizeId = defaultInstanceSize.Id
				kafkaRequest.DeprecatedIngressThroughputPerSec = defaultInstanceSize.IngressThroughputPerSec.String()
				kafkaRequest.DeprecatedEgressThroughputPerSec = defaultInstanceSize.EgressThroughputPerSec.String()
				kafkaRequest.DeprecatedTotalMaxConnections = int32(defaultInstanceSize.TotalMaxConnections)
				kafkaRequest.DeprecatedMaxPartitions = int32(defaultInstanceSize.MaxPartitions)
				kafkaRequest.DeprecatedMaxDataRetentionPeriod = defaultInstanceSize.MaxDataRetentionPeriod
				kafkaRequest.DeprecatedMaxConnectionAttemptsPerSec = int32(defaultInstanceSize.MaxConnectionAttemptsPerSec)

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
		request *dbapi.KafkaRequest
	}
	type fields struct {
		bootstrapServerHost string
		requestStatus       string
	}

	sampleRequest := dbapi.KafkaRequest{}

	tests := []struct {
		name   string
		args   args
		fields fields
		want   string
	}{
		{
			name: "non empty bootstrap server host conversion should return its value with colon and port number as a suffix",
			args: args{
				request: &sampleReadyRequest,
			},
			fields: fields{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
				requestStatus:       constants.KafkaRequestStatusReady.String(),
			},
			want: fmt.Sprintf("%s:443", nonEmptyBootstrapServerHost),
		},
		{
			name: "empty bootstrap server host conversion should return empty value",
			args: args{
				request: &sampleRequest,
			},
			fields: fields{
				bootstrapServerHost: "",
			},
			want: "",
		},
		{
			name: "should return empty value if KafkaRequest is in suspended status",
			args: args{
				request: &sampleRequest,
			},
			fields: fields{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
				requestStatus:       constants.KafkaRequestStatusSuspended.String(),
			},
			want: "",
		},
		{
			name: "should return empty value if KafkaRequest is in suspending status",
			args: args{
				request: &sampleRequest,
			},
			fields: fields{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
				requestStatus:       constants.KafkaRequestStatusSuspending.String(),
			},
			want: "",
		},
		{
			name: "should return empty value if KafkaRequest is in resuming status",
			args: args{
				request: &sampleRequest,
			},
			fields: fields{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
				requestStatus:       constants.KafkaRequestStatusResuming.String(),
			},
			want: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(setBootstrapServerHost(tt.args.request)).To(gomega.Equal(tt.want))
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
