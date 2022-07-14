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
	"github.com/onsi/gomega"
)

func GetHandlerParams(method string, url string, body io.Reader, t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	g := gomega.NewWithT(t)
	req, err := http.NewRequest(method, url, body)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	return req, httptest.NewRecorder()
}

func Test_ListErrors(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{
			name:    "Should create new errors handler and list all the errors",
			wantNil: false,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			errorsLen := strconv.Itoa(len(errors.Errors()))
			req, rw := GetHandlerParams("GET", "/", nil, t)
			handler := NewErrorsHandler()
			g.Expect(handler == nil).To(gomega.Equal(tt.wantNil))
			handler.List(rw, req) //nolint
			g.Expect(rw.Code).To(gomega.Equal(http.StatusOK))
			bodyStr, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			// body response should contain the size (len) of the errors
			g.Expect(bodyStr).To(gomega.ContainSubstring(errorsLen))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("GET", fmt.Sprintf("/%s", tt.args.id), nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": tt.args.id})
			handler.Get(rw, req)
			g.Expect(rw.Code).To(gomega.Equal(tt.wantStatusCode))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("POST", "/", nil, t)
			handler.Create(rw, req)
			g.Expect(rw.Code).To(gomega.Equal(tt.wantStatusCode))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("DELETE", "/", nil, t)
			handler.Delete(rw, req)
			g.Expect(rw.Code).To(gomega.Equal(tt.wantStatusCode))
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

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			handler := NewErrorsHandler()
			req, rw := GetHandlerParams("PATCH", "/", nil, t)
			handler.Patch(rw, req)
			g.Expect(rw.Code).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}
