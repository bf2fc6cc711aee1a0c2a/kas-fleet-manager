package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewKafkaLifespanConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KafkaLifespanConfig
	}{
		{
			name: "should return new KafkaLifespanConfig",
			want: &KafkaLifespanConfig{
				EnableDeletionOfExpiredKafka: true,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewKafkaLifespanConfig()).To(gomega.Equal(tt.want))
		})
	}
}
