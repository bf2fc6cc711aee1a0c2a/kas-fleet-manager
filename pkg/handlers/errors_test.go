package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	. "github.com/onsi/gomega"
)

func GetHandlerParams(method string, url string, body io.Reader) (*http.Request, *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, body)
	Expect(err != nil).To(BeFalse())

	return req, httptest.NewRecorder()
}

func Test_ListErrors(t *testing.T) {
	tests := []struct {
		name       string
		wantNotNil bool
	}{
		{
			name:       "Should create new errors handler and list all the errors",
			wantNotNil: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorsLen := strconv.Itoa(len(errors.Errors()))
			req, rw := GetHandlerParams("GET", "/", nil)
			handler := NewErrorsHandler()
			Expect(handler != nil).To(Equal(tt.wantNotNil))
			handler.List(rw, req) //nolint
			Expect(rw.Code).To(Equal(http.StatusOK))
			bodyStr, err := io.ReadAll(rw.Body)
			Expect(err).ToNot(HaveOccurred())
			// body response should contain the size (len) of the errors
			Expect(bodyStr).To(ContainSubstring(errorsLen))
		})
	}
}

func Test_GetError(t *testing.T) {
	invalidErrorId := "invalid"
	notFoundErrorId := "999999"
	errorId := strconv.Itoa(int(errors.Errors()[0].Code))
	type args struct {
		id string
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
	}{
		{
			name: "Should create new errors handler and get an existent error by id without failure",
			args: args{
				id: errorId,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "Should create new errors handler and get not found if providing invalid id",
			args: args{
				id: invalidErrorId,
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "Should create new errors handler and get not found if providing id of non-existent error",
			args: args{
				id: notFoundErrorId,
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("GET", fmt.Sprintf("/%s", tt.args.id), nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.args.id})
			handler.Get(rw, req)
			Expect(rw.Code).To(Equal(tt.wantStatusCode))
		})
	}
}

func Test_CreateError(t *testing.T) {
	tests := []struct {
		name           string
		wantStatusCode int
	}{
		{
			name:           "Should call Create error endpoint",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("POST", "/", nil)
			handler.Create(rw, req)
			Expect(rw.Code).To(Equal(tt.wantStatusCode))
		})
	}
}

func Test_DeleteError(t *testing.T) {
	tests := []struct {
		name           string
		wantStatusCode int
	}{
		{
			name:           "Should call Delete error endpoint",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("DELETE", "/", nil)
			handler.Delete(rw, req)
			Expect(rw.Code).To(Equal(tt.wantStatusCode))
		})
	}
}

func Test_PatchError(t *testing.T) {
	tests := []struct {
		name           string
		wantStatusCode int
	}{
		{
			name:           "Should call Patch error endpoint",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("PATCH", "/", nil)
			handler.Patch(rw, req)
			Expect(rw.Code).To(Equal(tt.wantStatusCode))
		})
	}
}
