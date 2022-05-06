package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_PrometheusHandler(t *testing.T) {
	tests := []struct {
		name       string
		wantNotNil bool
	}{
		{
			name:       "Should create PrometheusMetricsHandler",
			wantNotNil: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPrometheusMetricsHandler().Handler()
			Expect(handler != nil).To(Equal(tt.wantNotNil))
			req, rw := GetHandlerParams("GET", "/", nil)
			handler.ServeHTTP(rw, req)
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
