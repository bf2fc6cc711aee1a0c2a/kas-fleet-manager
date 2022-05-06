package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_OpenapiHandler(t *testing.T) {
	tests := []struct {
		name       string
		wantNotNil bool
	}{
		{
			name:       "Should create OpenAPIHandler",
			wantNotNil: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewOpenAPIHandler(nil)
			Expect(handler != nil).To(Equal(tt.wantNotNil))

			req, rw := GetHandlerParams("GET", "/", nil)

			handler.Get(rw, req)
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
