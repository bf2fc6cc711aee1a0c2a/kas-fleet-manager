package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	. "github.com/onsi/gomega"
)

func Test_NewKafkaQuotaConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KafkaQuotaConfig
	}{
		{
			name: "should return new KafkaQuotaConfig",
			want: &KafkaQuotaConfig{
				Type:                   api.QuotaManagementListQuotaType.String(),
				AllowEvaluatorInstance: true,
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewKafkaQuotaConfig()).To(Equal(tt.want))
		})
	}
}
