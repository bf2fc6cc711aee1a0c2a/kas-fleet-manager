package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
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
				AllowDeveloperInstance: true,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewKafkaQuotaConfig()).To(gomega.Equal(tt.want))
		})
	}
}
