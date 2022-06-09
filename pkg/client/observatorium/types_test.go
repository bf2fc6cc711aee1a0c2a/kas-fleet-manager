package observatorium

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test_FillDefaults(t *testing.T) {
	type fields struct {
		config *MetricsReqParams
	}

	tests := []struct {
		name   string
		fields fields
		want   *MetricsReqParams
	}{
		{
			name: "should return MetricsReqParams filled with default values",
			fields: fields{
				config: &MetricsReqParams{},
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			tt.fields.config.FillDefaults()
			Expect(tt.fields.config.Step).To(Equal(30 * time.Second))
		})
	}
}
