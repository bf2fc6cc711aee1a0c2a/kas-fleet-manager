package converters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"

	. "github.com/onsi/gomega"
)

func Test_ConvertKafkaRequest(t *testing.T) {
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "ConvertKafkaRequest returning KafkaRequest in form of []map[string]interface{}",
			args: args{
				kafka: mocks.BuildKafkaRequest(nil),
			},
			want: mocks.BuildKafkaRequestMap(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertKafkaRequest(tt.args.kafka)).To(Equal(tt.want))
		})
	}
}

func Test_ConvertKafkaRequestList(t *testing.T) {
	type args struct {
		kafkaList dbapi.KafkaList
	}
	tests := []struct {
		name string
		args args
		want []map[string]interface{}
	}{
		{
			name: "ConvertKafkaRequestList returning KafkaRequest list in form of []map[string]interface{}",
			args: args{
				kafkaList: dbapi.KafkaList{
					mocks.BuildKafkaRequest(nil),
				},
			},
			want: mocks.BuildKafkaRequestMap(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertKafkaRequestList(tt.args.kafkaList)).To(Equal(tt.want))
		})
	}
}
