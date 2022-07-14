package handlers

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_OpenapiHandler(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "Should create OpenAPIHandler",
			wantNil: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewOpenAPIHandler(nil)
			g.Expect(handler == nil).To(gomega.Equal(tt.wantNil))

			req, rw := GetHandlerParams("GET", "/", nil, t)

			handler.Get(rw, req) //nolint
			g.Expect(rw.Code).ToNot(gomega.Equal(0))
		})
	}
}
