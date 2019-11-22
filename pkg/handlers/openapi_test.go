package handlers

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestOpenAPIHandler(t *testing.T) {
	json := []byte(`{"openapi": "json"}`)
	handler := NewOpenAPIHandler(json)

	request := httptest.NewRequest("GET", "/openapi", nil)
	writer := &mockResponseWriter{}

	handler.Get(writer, request)

	fmt.Printf("output: %s", writer.written)

}
