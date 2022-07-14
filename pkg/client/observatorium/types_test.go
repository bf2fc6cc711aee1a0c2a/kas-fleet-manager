package observatorium

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.fields.config.FillDefaults()
			g.Expect(tt.fields.config.Step).To(gomega.Equal(30 * time.Second))
		})
	}
}
