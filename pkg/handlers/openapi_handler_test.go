package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			handler := NewOpenAPIHandler(nil)
			Expect(handler == nil).To(Equal(tt.wantNil))

			req, rw := GetHandlerParams("GET", "/", nil)

			handler.Get(rw, req) //nolint
			Expect(rw.Code).ToNot(Equal(0))
		})
	}
}
