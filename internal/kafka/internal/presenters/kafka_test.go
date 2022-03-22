package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

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
