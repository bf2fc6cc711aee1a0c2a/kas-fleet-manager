package test

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

func NewQuotaListTestKafkaSupportedInstanceTypesConfig() *config.KafkaSupportedInstanceTypesConfig {
	cfg := quotaListKafkaSupportedInstanceTypesConfig
	return &cfg
}

func NewAMSTestKafkaSupportedInstanceTypesConfig() *config.KafkaSupportedInstanceTypesConfig {
	cfg := amsKafkaSupportedInstanceTypesConfig
	return &cfg
}

var quotaListKafkaSupportedInstanceTypesConfig = config.KafkaSupportedInstanceTypesConfig{
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
						DeprecatedQuotaType:         "rhosak",
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
						DeprecatedQuotaType:         "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
		},
	},
}

var amsKafkaSupportedInstanceTypesConfig = config.KafkaSupportedInstanceTypesConfig{
	Configuration: config.SupportedKafkaInstanceTypesConfig{
		SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
			{
				Id:          "standard",
				DisplayName: "Standard",
				SupportedBillingModels: []config.KafkaBillingModel{
					{
						ID:               "standard",
						AMSResource:      "rhosak",
						AMSProduct:       "RHOSAK",
						AMSBillingModels: []string{"standard"},
					},
					{
						ID:               "marketplace",
						AMSResource:      "rhosak",
						AMSProduct:       "RHOSAK",
						AMSBillingModels: []string{"marketplace", "marketplace-aws", "marketplace-rhm"},
					},
					{
						ID:               "eval",
						AMSResource:      "rhosak",
						AMSProduct:       "RHOSAKEval",
						AMSBillingModels: []string{"standard"},
					},
				},
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
						DeprecatedQuotaType:         "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
			{
				Id:          "developer",
				DisplayName: "Trial",
				SupportedBillingModels: []config.KafkaBillingModel{
					{
						ID:               "standard",
						AMSResource:      "rhosak",
						AMSProduct:       "RHOSAKTrial",
						AMSBillingModels: []string{"standard"},
					},
				},
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
						DeprecatedQuotaType:         "rhosak",
						CapacityConsumed:            1,
					},
				},
			},
		},
	},
}
