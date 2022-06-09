package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			handler := NewPrometheusMetricsHandler().Handler()
			Expect(handler == nil).To(Equal(tt.wantNil))
			req, rw := GetHandlerParams("GET", "/", nil)
			handler.ServeHTTP(rw, req)
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
