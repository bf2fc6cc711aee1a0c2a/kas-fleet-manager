package handlers

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			req, rw := GetHandlerParams("GET", "/", nil)
			m := http.NewServeMux()
			handler := MetricsMiddleware(m)
			Expect(handler == nil).To(Equal(tt.wantNil))
			handler.ServeHTTP(rw, req)
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
