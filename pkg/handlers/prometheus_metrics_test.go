package handlers

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_PrometheusHandler(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "Should create PrometheusMetricsHandler",
			wantNil: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewPrometheusMetricsHandler().Handler()
			g.Expect(handler == nil).To(gomega.Equal(tt.wantNil))
			req, rw := GetHandlerParams("GET", "/", nil, t)
			handler.ServeHTTP(rw, req)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}
