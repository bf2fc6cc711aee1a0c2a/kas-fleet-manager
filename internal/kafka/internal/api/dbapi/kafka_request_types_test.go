package dbapi

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/onsi/gomega"
)

func TestKafkaRequest_DesiredBillingModelIsEnterprise(t *testing.T) {
	type fields struct {
		DesiredKafkaBillingModel string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if enterprise is the desired billing model",
			fields: fields{
				DesiredKafkaBillingModel: constants.BillingModelEnterprise.String(),
			},
			want: true,
		},
		{
			name: "return false if enterprise is not the desired billing model",
			fields: fields{
				DesiredKafkaBillingModel: "something else",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaRequest{
				DesiredKafkaBillingModel: testcase.fields.DesiredKafkaBillingModel,
			}
			got := k.DesiredBillingModelIsEnterprise()
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}
