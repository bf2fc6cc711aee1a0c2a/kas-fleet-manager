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
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			Expect(NewKafkaLifespanConfig()).To(Equal(tt.want))
		})
	}
}
