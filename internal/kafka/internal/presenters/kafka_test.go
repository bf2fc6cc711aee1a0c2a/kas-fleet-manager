package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"

	. "github.com/onsi/gomega"
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
			// want: dbKafkaRequest, //mock.BuildKafkaRequest(nil),
			want: mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
				kafkaRequest.Status = ""
				kafkaRequest.Owner = ""
				kafkaRequest.ClusterID = ""
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.Meta.DeletedAt.Valid = false
			}),
		},
		{
			name: "Should return empty kafka request with reauthentication disabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					payload.ReauthenticationEnabled = &reauthDisabled
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ReauthenticationEnabled = reauthDisabled
				})},
			},
			want: mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthDisabled
			}),
		},
		{
			name: "should convert and return kafka request from non-empty db kafka request slice with reauth enabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(nil),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ReauthenticationEnabled = reauthEnabled
				})},
			},
			want: mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
			}),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertKafkaRequest(tt.args.kafkaRequestPayload, tt.args.dbKafkaRequests...)).To(Equal(tt.want))
		})
	}
}

func TestPresentKafkaRequest(t *testing.T) {
	type args struct {
		dbKafkaRequest *dbapi.KafkaRequest
	}

	bootstrapServer := "http://test.com"
	failedReason := ""
	version := "2.8.0"
	instanceType := types.STANDARD.String()
	reauthEnabled := true
	kafkaStorageSize := "1000Gi"

	tests := []struct {
		name   string
		args   args
		want   public.KafkaRequest
		config config.KafkaConfig
	}{
		{
			name: "should return kafka request as presented to an end user",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
					kafkaRequest.ReauthenticationEnabled = reauthEnabled
					kafkaRequest.BootstrapServerHost = bootstrapServer
					kafkaRequest.FailedReason = failedReason
					kafkaRequest.ActualKafkaVersion = version
					kafkaRequest.InstanceType = instanceType
					kafkaRequest.KafkaStorageSize = kafkaStorageSize
				}),
			},
			want: *mock.BuildPublicKafkaRequest(func(kafkaRequest *public.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(bootstrapServer)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = instanceType
				kafkaRequest.KafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
			}),
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "standard",
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
									},
								},
							},
						},
					},
				},
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentKafkaRequest(tt.args.dbKafkaRequest, &tt.config)).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(setBootstrapServerHost(tt.args.bootstrapServerHost)).To(Equal(tt.want))
		})
	}
}

func TestCapacityLimitReports(t *testing.T) {
	tests := []struct {
		name     string
		request  dbapi.KafkaRequest
		config   config.KafkaConfig
		negative bool
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
								Id: "standard",
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
									},
								},
							},
						},
					},
				},
			},
			negative: false,
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
				InstanceType:     "eval",
				QuotaType:        "rhosak",
				SizeId:           "x1",
			},
			config: config.KafkaConfig{
				SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
					Configuration: config.SupportedKafkaInstanceTypesConfig{
						SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
							{
								Id: "standard",
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
									},
								},
							},
						},
					},
				},
			},
			negative: true,
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
								Id: "standard",
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			kafkaRequest := PresentKafkaRequest(&test.request, &test.config)
			if !test.negative {
				gomega.Expect(kafkaRequest.IngressThroughputPerSec).ToNot(gomega.BeNil())
				gomega.Expect(kafkaRequest.EgressThroughputPerSec).ToNot(gomega.BeNil())
				gomega.Expect(kafkaRequest.TotalMaxConnections).ToNot(gomega.BeNil())
				gomega.Expect(kafkaRequest.MaxConnectionAttemptsPerSec).ToNot(gomega.BeNil())
				gomega.Expect(kafkaRequest.MaxDataRetentionPeriod).ToNot(gomega.BeNil())
				gomega.Expect(kafkaRequest.MaxPartitions).ToNot(gomega.BeNil())
			} else {
				gomega.Expect(kafkaRequest.IngressThroughputPerSec).To(gomega.BeEmpty())
				gomega.Expect(kafkaRequest.EgressThroughputPerSec).To(gomega.BeEmpty())
				gomega.Expect(kafkaRequest.TotalMaxConnections).To(gomega.BeZero())
				gomega.Expect(kafkaRequest.MaxConnectionAttemptsPerSec).To(gomega.BeZero())
				gomega.Expect(kafkaRequest.MaxDataRetentionPeriod).To(gomega.BeEmpty())
				gomega.Expect(kafkaRequest.MaxPartitions).To(gomega.BeZero())
			}
		})
	}
}
