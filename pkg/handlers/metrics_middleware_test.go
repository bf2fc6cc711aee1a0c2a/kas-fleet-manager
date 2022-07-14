package handlers

import (
	"net/http"
	"testing"

	"github.com/onsi/gomega"
)

func Test_MetricsMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "Should create metrics middleware handler",
			wantNil: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", "/", nil, t)
			m := http.NewServeMux()
			handler := MetricsMiddleware(m)
			g.Expect(handler == nil).To(gomega.Equal(tt.wantNil))
			handler.ServeHTTP(rw, req)
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}
