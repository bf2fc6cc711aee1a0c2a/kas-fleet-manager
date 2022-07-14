package types

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/onsi/gomega"
)

func Test_ValidKafkaInstanceTypes(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(ValidKafkaInstanceTypes).To(gomega.Equal([]string{DEVELOPER.String(), STANDARD.String()}))
}

func Test_GetQuotaType(t *testing.T) {
	type fields struct {
		t KafkaInstanceType
	}

	tests := []struct {
		name   string
		fields fields
		want   ocm.KafkaQuotaType
	}{
		{
			name: "Should return ocm.StandardQuota for STANDARD KafkaInstanceType",
			fields: fields{
				t: STANDARD,
			},
			want: ocm.StandardQuota,
		},
		{
			name: "Should return ocm.DeveloperQuota for DEVELOPER KafkaInstanceType",
			fields: fields{
				t: DEVELOPER,
			},
			want: ocm.DeveloperQuota,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.t.GetQuotaType()).To(gomega.Equal(tt.want))
		})
	}
}
