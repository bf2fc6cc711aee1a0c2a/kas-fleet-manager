package handlers

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_MetricsMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		wantNotNil bool
	}{
		{
			name:       "Should create metrics middleware handler",
			wantNotNil: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rw := GetHandlerParams("GET", "/", nil)
			m := http.NewServeMux()
			handler := MetricsMiddleware(m)
			Expect(handler != nil).To(Equal(tt.wantNotNil))
			handler.ServeHTTP(rw, req)
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
