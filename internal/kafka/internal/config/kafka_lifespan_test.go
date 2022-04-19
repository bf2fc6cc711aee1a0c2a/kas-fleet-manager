package config

import (
	"testing"

	. "github.com/onsi/gomega"
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
				KafkaLifespanInHours:         48,
			},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewKafkaLifespanConfig()).To(Equal(tt.want))
		})
	}
}
